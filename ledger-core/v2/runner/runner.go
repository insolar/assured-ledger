// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/calltype"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/builtin"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type Service interface {
	ExecutionStart(ctx context.Context, execution execution.Context) (*executionupdate.ContractExecutionStateUpdate, uuid.UUID, error)
	ExecutionContinue(ctx context.Context, id uuid.UUID, result interface{}) (*executionupdate.ContractExecutionStateUpdate, error)
	ExecutionAbort(ctx context.Context, id uuid.UUID)
	ExecutionClassify(ctx context.Context, execution execution.Context) calltype.ContractCallType
	ContractCompile(ctx context.Context, contract interface{})
}

type DefaultService struct {
	Cache   descriptor.Cache
	Manager executor.Manager

	eventSinkMap     map[uuid.UUID]*executionEventSink
	eventSinkMapLock sync.Mutex
}

func (r *DefaultService) stopExecution(id uuid.UUID) error { // nolint
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	if val, ok := r.eventSinkMap[id]; ok {
		delete(r.eventSinkMap, id)
		val.Stop()
	}

	return nil
}

func (r *DefaultService) getExecutionSink(id uuid.UUID) *executionEventSink {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	return r.eventSinkMap[id]
}

func (r *DefaultService) waitForReply(id uuid.UUID) (*executionupdate.ContractExecutionStateUpdate, uuid.UUID, error) {
	executionContext := r.getExecutionSink(id)
	if executionContext == nil {
		panic("failed to find ExecutionContext")
	}

	switch update := <-executionContext.output; update.Type {
	case executionupdate.TypeDone, executionupdate.TypeAborted:
		_ = r.stopExecution(id)
		fallthrough
	case executionupdate.TypeError, executionupdate.TypeOutgoingCall:
		return update, id, nil
	default:
		panic(fmt.Sprintf("unknown return type %v", update.Type))
	}
}

func (r *DefaultService) createExecutionSink(execution execution.Context) uuid.UUID {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	// TODO[bigbes]: think how to change from UUID to natural key, here (execution deduplication)
	var id uuid.UUID
	for {
		id := uuid.New()

		if _, ok := r.eventSinkMap[id]; !ok {
			break
		}
	}

	r.eventSinkMap[id] = newEventSink(execution)

	return id
}

func (r *DefaultService) executionRecover(ctx context.Context, id uuid.UUID) {
	if err := recover(); err != nil {
		// replace with custom error, not RecoverSlotPanicWithStack
		err := smachine.RecoverSlotPanicWithStack("ContractRunnerService panic", err, nil, smachine.AsyncCallArea)

		executionContext := r.getExecutionSink(id)
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
	execution execution.Context,
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
		CallerPrototype: &request.CallSiteDeclaration,

		Request: &execution.Reference,

		TraceID: inslogger.TraceID(ctx),
	}

	if oDesc := execution.Object; oDesc != nil {
		res.Parent = oDesc.Parent()
		// should be the same as request.Object
		res.Callee = oDesc.HeadRef()
	} else {
		res.Callee = &execution.Request.Caller
	}

	return res
}

func (r *DefaultService) executeMethod(_ context.Context, _ uuid.UUID, _ *executionEventSink) {
	panic(throw.NotImplemented())
}

func (r *DefaultService) executeConstructor(ctx context.Context, id uuid.UUID, eventSink *executionEventSink) {
	var (
		executionContext = eventSink.context
		request          = executionContext.Request
	)

	protoDesc, codeDesc, err := r.Cache.ByPrototypeRef(ctx, request.CallSiteDeclaration)
	if err != nil {
		eventSink.ErrorWrapped(err, "couldn't get descriptors")
	}

	codeExecutor, err := r.Manager.GetExecutor(codeDesc.MachineType())
	if err != nil {
		eventSink.ErrorWrapped(err, "couldn't get executor")
	}

	logicContext := generateCallContext(ctx, id, executionContext, protoDesc, codeDesc)

	newData, result, err := codeExecutor.CallConstructor(ctx, logicContext, *codeDesc.Ref(), request.CallSiteMethod, request.Arguments)
	if err != nil {
		eventSink.ErrorWrapped(err, "execution error")
	}
	if len(result) == 0 {
		eventSink.ErrorString("return of constructor is empty")
	}

	// form and return result
	// TODO: think how to provide ObjectReference here (== RequestReference)
	res := requestresult.New(result, insolar.Reference{})
	if newData != nil {
		res.SetActivate(request.Callee, request.CallSiteDeclaration, newData)
	}

	eventSink.Result(res)
}

func (r *DefaultService) execute(ctx context.Context, id uuid.UUID) {
	defer r.executionRecover(ctx, id)

	eCtx := r.getExecutionSink(id)
	if eCtx == nil {
		panic(throw.Impossible())
	}

	switch eCtx.context.Request.CallType {
	case payload.CTMethod:
		r.executeMethod(ctx, id, eCtx)
	case payload.CTConstructor:
		r.executeConstructor(ctx, id, eCtx)
	default:
		panic(throw.Unsupported())
	}
}

func (r *DefaultService) ExecutionStart(ctx context.Context, execution execution.Context) (*executionupdate.ContractExecutionStateUpdate, uuid.UUID, error) {
	id := r.createExecutionSink(execution)

	go r.execute(ctx, id)

	return r.waitForReply(id)
}

func (r *DefaultService) ExecutionClassify(ctx context.Context, execution execution.Context) calltype.ContractCallType {
	return calltype.ContractCallOrdered
}

func (r *DefaultService) ExecutionContinue(ctx context.Context, id uuid.UUID, result interface{}) (*executionupdate.ContractExecutionStateUpdate, error) {
	panic(throw.NotImplemented())
}

func (r *DefaultService) ExecutionAbort(ctx context.Context, id uuid.UUID) {
	panic(throw.NotImplemented())
}

func (r *DefaultService) ContractCompile(ctx context.Context, contract interface{}) {
	panic(throw.NotImplemented())
}

func NewService() *DefaultService {
	return &DefaultService{
		Cache:   NewDescriptorsCache(),
		Manager: executor.NewManager(),

		eventSinkMap:     make(map[uuid.UUID]*executionEventSink),
		eventSinkMapLock: sync.Mutex{},
	}
}

func (r *DefaultService) Init() error {
	exec := builtin.New(r)
	if err := r.Manager.RegisterExecutor(insolar.MachineTypeBuiltin, exec); err != nil {
		panic(throw.W(err, "failed to register executor", nil))
	}
	r.Cache.RegisterCallback(exec.GetDescriptor)

	return nil
}
