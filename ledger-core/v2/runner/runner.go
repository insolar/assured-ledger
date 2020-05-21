// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"bytes"
	"context"
	"sync"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/builtin"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type Service interface {
	ExecutionStart(execution execution.Context) *RunState
	ExecutionContinue(run *RunState, outgoingResult []byte)
	ExecutionAbort(run *RunState)
}

type UnmanagedService interface {
	ExecutionClassify(execution execution.Context) interface{}
}

type DefaultService struct {
	Cache descriptor.Cache

	Manager machine.Manager

	lastPosition           call.ID
	eventSinkInProgressMap map[call.ID]*awaitedRun
	eventSinkMap           map[call.ID]*executionEventSink
	eventSinkMapLock       sync.Mutex
}

func (r *DefaultService) stopExecution(id call.ID) error { // nolint
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	if val, ok := r.eventSinkMap[id]; ok {
		delete(r.eventSinkMap, id)
		val.Stop()
	}

	return nil
}

func (r *DefaultService) getExecutionSink(id call.ID) *executionEventSink {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	return r.eventSinkMap[id]
}

func (r *DefaultService) awaitedRunFinish(id call.ID) {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	r.eventSinkInProgressMap[id].resumeFn()
	delete(r.eventSinkInProgressMap, id)
}

func (r *DefaultService) awaitedRunAdd(sink *executionEventSink, resumeFn func()) {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	r.eventSinkInProgressMap[sink.id] = &awaitedRun{
		run:      sink,
		resumeFn: resumeFn,
	}
}

func (r *DefaultService) createExecutionSink(execution execution.Context) *executionEventSink {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	id := r.lastPosition
	r.lastPosition++

	eventSink := newEventSink(execution)
	eventSink.id = id
	r.eventSinkMap[id] = eventSink

	return eventSink
}

func (r *DefaultService) executionRecover(ctx context.Context, id call.ID) {
	if rec := recover(); rec != nil {
		// replace with custom error, not RecoverSlotPanicWithStack
		err := throw.R(rec, throw.E("ContractRunnerService panic"))

		executionContext := r.getExecutionSink(id)
		if executionContext == nil {
			logger := inslogger.FromContext(ctx)
			logger.Errorf("[executionRecover] Failed to find a job execution context %d", id)
			logger.Errorf("[executionRecover] Failed to execute a job, panic: %v", r)
			return
		}

		executionContext.Error(err)
	}
}

func generateCallContext(
	ctx context.Context,
	id call.ID,
	execution execution.Context,
	protoDesc descriptor.Prototype,
	codeDesc descriptor.Code,
) *call.LogicContext {
	request := execution.Request
	res := &call.LogicContext{
		ID:   id,
		Mode: call.Execute,

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
	id call.ID,
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
	id call.ID,
	eventSink *executionEventSink,
) (
	*requestresult.RequestResult, error,
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

func (r *DefaultService) execute(ctx context.Context, id call.ID) {
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

	r.awaitedRunFinish(id)
}

func (r *DefaultService) runPrepare(execution execution.Context) *executionEventSink {
	return r.createExecutionSink(execution)
}

func (r *DefaultService) runStart(run *executionEventSink, resumeFn func()) {
	r.awaitedRunAdd(run, resumeFn)

	go r.execute(context.Background(), run.id)
}

func (r *DefaultService) runContinue(run *executionEventSink, resumeFn func()) {
	r.awaitedRunAdd(run, resumeFn)
}

func (r *DefaultService) runAbort(_ *executionEventSink, _ func()) {
	panic(throw.NotImplemented())
}

func (r *DefaultService) ExecutionClassify(_ execution.Context) interface{} {
	panic(throw.NotImplemented())
}

func NewService() *DefaultService {
	return &DefaultService{
		Cache:   NewDescriptorsCache(),
		Manager: machine.NewManager(),

		lastPosition:           0,
		eventSinkMap:           make(map[call.ID]*executionEventSink),
		eventSinkInProgressMap: make(map[call.ID]*awaitedRun),
		eventSinkMapLock:       sync.Mutex{},
	}
}

func (r *DefaultService) Init() error {
	exec := builtin.New(r)
	if err := r.Manager.RegisterExecutor(machine.Builtin, exec); err != nil {
		panic(throw.W(err, "failed to register executor", nil))
	}

	r.Cache.RegisterCallback(exec.GetDescriptor)

	return nil
}
