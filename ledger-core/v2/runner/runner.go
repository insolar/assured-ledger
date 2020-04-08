// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	runner2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/builtin"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type runner struct {
	cache   descriptor.DescriptorsCache
	manager executor.Manager

	executions     map[uuid.UUID]*executionContext
	executionsLock sync.Mutex
}

func (r *runner) stopExecution(id uuid.UUID) error {
	r.executionsLock.Lock()
	defer r.executionsLock.Unlock()

	if val, ok := r.executions[id]; ok {
		delete(r.executions, id)
		val.Stop()
	}

	return nil
}

func (r *runner) getExecutionContext(id uuid.UUID) *executionContext {
	r.executionsLock.Lock()
	defer r.executionsLock.Unlock()

	return r.executions[id]
}

func (r *runner) waitForReply(id uuid.UUID) (*runner2.ContractExecutionStateUpdate, uuid.UUID, error) {
	executionContext := r.getExecutionContext(id)
	if executionContext == nil {
		panic("failed to find ExecutionContext")
	}

	switch update := <-executionContext.output; update.Type {
	case runner2.ContractDone, runner2.ContractAborted:
		_ = r.stopExecution(id)
		fallthrough
	case runner2.ContractError, runner2.ContractOutgoingCall:
		return update, id, nil
	default:
		panic(fmt.Sprintf("unknown return type %v", update.Type))
	}
}

func (r *runner) createExecutionContext(execution runner2.Execution) uuid.UUID {
	r.executionsLock.Lock()
	defer r.executionsLock.Unlock()

	// TODO[bigbes]: think how to change from UUID to natural key, here (execution deduplication)
	var id uuid.UUID
	for {
		id := uuid.New()

		if _, ok := r.executions[id]; !ok {
			break
		}
	}

	r.executions[id] = newExecutionContext(execution)

	return id
}

func (r *runner) executionRecover(ctx context.Context, id uuid.UUID) {
	if err := recover(); err != nil {
		// replace with custom error, not RecoverSlotPanicWithStack
		err := smachine.RecoverSlotPanicWithStack("ContractRunnerService panic", err, nil, smachine.AsyncCallArea)

		executionContext := r.getExecutionContext(id)
		if executionContext == nil {
			inslogger.FromContext(ctx).Errorf("[executionRecover] Failed to find a job execution context %s", id.String())
			inslogger.FromContext(ctx).Errorf("[executionRecover] Failed to execute a job, panic: %v", r)
			return
		}

		executionContext.Error(err)
	}
}

func generateCallContext(
	ctx context.Context,
	id uuid.UUID,
	execution runner2.Execution,
	protoDesc descriptor.PrototypeDescriptor,
	codeDesc descriptor.CodeDescriptor,
) *insolar.LogicCallContext {
	request := execution.Request
	res := &insolar.LogicCallContext{
		ID:   id,
		Mode: insolar.ExecuteCallMode,

		Callee:    nil, // below
		Prototype: protoDesc.HeadRef(),
		Code:      codeDesc.Ref(),

		Caller:          &request.Caller,
		CallerPrototype: &request.CallerPrototype,

		TraceID: inslogger.TraceID(ctx),
	}

	if oDesc := execution.Object; oDesc != nil {
		res.Parent = oDesc.Parent()
		// should be the same as request.Object
		res.Callee = oDesc.HeadRef()
	} else {
		res.Callee = execution.Request.Object
	}

	return res
}

func (r *runner) executeMethod(_ context.Context, _ uuid.UUID, _ *executionContext) bool {
	panic(throw.NotImplemented())
}

func (r *runner) executeConstructor(ctx context.Context, id uuid.UUID, eCtx *executionContext) bool {
	var (
		execution = eCtx.execution
		request   = execution.Request
	)

	protoDesc, codeDesc, err := r.cache.ByPrototypeRef(ctx, *request.Prototype)
	if err != nil {
		return eCtx.ErrorWrapped(err, "couldn't get descriptors")
	}

	executor, err := r.manager.GetExecutor(codeDesc.MachineType())
	if err != nil {
		return eCtx.ErrorWrapped(err, "couldn't get executor")
	}

	logicContext := generateCallContext(ctx, id, execution, protoDesc, codeDesc)

	newData, result, err := executor.CallConstructor(ctx, logicContext, *codeDesc.Ref(), request.Method, request.Arguments)
	if err != nil {
		return eCtx.ErrorWrapped(err, "execution error")
	}
	if len(result) == 0 {
		return eCtx.ErrorString("return of constructor is empty")
	}

	// form and return result
	// TODO: think how to provide ObjectReference here (== RequestReference)
	res := requestresult.New(result, insolar.Reference{})
	if newData != nil {
		res.SetActivate(*request.Base, *request.Prototype, newData)
	}

	return eCtx.Result(res)
}

func (r *runner) execute(ctx context.Context, id uuid.UUID) {
	defer r.executionRecover(ctx, id)

	eCtx := r.getExecutionContext(id)
	if eCtx == nil {
		panic(throw.Impossible())
	}

	var rv bool
	switch eCtx.execution.Request.CallType {
	case record.CTMethod:
		rv = r.executeMethod(ctx, id, eCtx)
	case record.CTSaveAsChild:
		rv = r.executeConstructor(ctx, id, eCtx)
	}

	if !rv {
		panic(throw.Impossible())
	}
}

func (r *runner) ExecutionStart(ctx context.Context, execution runner2.Execution) (*runner2.ContractExecutionStateUpdate, uuid.UUID, error) {
	id := r.createExecutionContext(execution)

	go r.execute(ctx, id)

	return r.waitForReply(id)
}

func (r runner) ExecutionContinue(ctx context.Context, id uuid.UUID, result interface{}) (*runner2.ContractExecutionStateUpdate, error) {
	panic(throw.NotImplemented())
}

func (r runner) ExecutionAbort(ctx context.Context, id uuid.UUID) {
	panic(throw.NotImplemented())
}

func (r runner) ContractCompile(ctx context.Context, contract interface{}) {
	panic(throw.NotImplemented())
}

func NewRunner() (runner2.Runner, error) {
	return &runner{
		cache:          nil,
		manager:        executor.NewManager(),
		executions:     make(map[uuid.UUID]*executionContext),
		executionsLock: sync.Mutex{},
	}, nil
}

func (r *runner) Init() error {
	exec := builtin.NewBuiltIn(nil, r)
	if err := r.manager.RegisterExecutor(insolar.MachineTypeBuiltin, exec); err != nil {
		panic(throw.W(err, "failed to register executor", nil))
	}

	return nil
}
