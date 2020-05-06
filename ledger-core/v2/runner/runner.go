// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"

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
	Cache descriptor.Cache

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
		id = uuid.New()

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

		// Callee:    reference.Global{}, // is assigned below
		Prototype: protoDesc.HeadRef(),
		Code:      codeDesc.Ref(),

		Caller:          request.Caller,
		CallerPrototype: request.CallSiteDeclaration,

		Request: execution.Incoming,

		TraceID: inslogger.TraceID(ctx),
	}

	if oDesc := execution.ObjectDescriptor; oDesc != nil {
		res.Parent = oDesc.Parent()
		// should be the same as request.Object
		res.Callee = oDesc.HeadRef()
	} else {
		res.Callee = execution.Object
	}

	res.Unordered = execution.Unordered

	return res
}

func (r *DefaultService) executeMethod(
	ctx context.Context,
	id uuid.UUID,
	eventSink *executionEventSink,
) (
	*requestresult.RequestResult,
	error,
) {
	var (
		executionContext = eventSink.context
		request          = executionContext.Request

		objectDescriptor = executionContext.ObjectDescriptor
	)

	prototypeReference, err := objectDescriptor.Prototype()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get prototype reference")
	}
	if prototypeReference.IsEmpty() {
		panic(throw.IllegalState())
	}

	prototypeDescriptor, codeDescriptor, err := r.Cache.ByPrototypeRef(ctx, prototypeReference)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get descriptors")
	}

	codeExecutor, err := r.Manager.GetExecutor(codeDescriptor.MachineType())
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get executor")
	}

	logicContext := generateCallContext(ctx, id, executionContext, prototypeDescriptor, codeDescriptor)

	newData, result, err := codeExecutor.CallMethod(
		ctx, logicContext, codeDescriptor.Ref(), objectDescriptor.Memory(), request.CallSiteMethod, request.Arguments,
	)
	if err != nil {
		return nil, errors.Wrap(err, "execution error")
	}
	if len(result) == 0 {
		return nil, errors.New("return of method is empty")
	}
	if len(newData) == 0 {
		return nil, errors.New("object state is empty")
	}

	// form and return result
	res := requestresult.New(result, objectDescriptor.HeadRef())

	if !bytes.Equal(objectDescriptor.Memory(), newData) {
		res.SetAmend(objectDescriptor, newData)
	}

	return res, nil
}

func (r *DefaultService) executeConstructor(
	ctx context.Context,
	id uuid.UUID,
	eventSink *executionEventSink,
) (
	*requestresult.RequestResult,
	error,
) {
	var (
		executionContext = eventSink.context
		request          = executionContext.Request
	)

	prototypeDescriptor, codeDescriptor, err := r.Cache.ByPrototypeRef(ctx, request.CallSiteDeclaration)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get descriptors")
	}

	codeExecutor, err := r.Manager.GetExecutor(codeDescriptor.MachineType())
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get executor")
	}

	logicContext := generateCallContext(ctx, id, executionContext, prototypeDescriptor, codeDescriptor)

	newState, executionResult, err := codeExecutor.CallConstructor(ctx, logicContext, codeDescriptor.Ref(), request.CallSiteMethod, request.Arguments)
	if err != nil {
		return nil, errors.Wrap(err, "execution error")
	}
	if len(executionResult) == 0 {
		return nil, errors.New("return of constructor is empty")
	}

	// form and return executionResult
	res := requestresult.New(executionResult, executionContext.Object)
	if newState != nil {
		res.SetActivate(request.Callee, request.CallSiteDeclaration, newState)
	}

	return res, nil
}

func (r *DefaultService) execute(ctx context.Context, id uuid.UUID) {
	defer r.executionRecover(ctx, id)

	executionSink := r.getExecutionSink(id)
	if executionSink == nil {
		panic(throw.Impossible())
	}

	var (
		result *requestresult.RequestResult
		err    error
	)

	switch executionSink.context.Request.CallType {
	case payload.CTMethod:
		result, err = r.executeMethod(ctx, id, executionSink)
	case payload.CTConstructor:
		result, err = r.executeConstructor(ctx, id, executionSink)
	default:
		panic(throw.Unsupported())
	}

	switch {
	case err != nil:
		executionSink.Error(err)
	case result != nil:
		executionSink.Result(result)
	default:
		panic(throw.IllegalValue())
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
