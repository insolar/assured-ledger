package runner

import (
	"bytes"
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/builtin"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine/machinetype"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

type ErrorDetail struct {
	Type int
}

const (
	DetailEmptyClassRef = iota
	DetailBadClassRef
)

type UnmanagedService interface {
	ExecutionStart(execution execution.Context) RunState
	ExecutionContinue(run RunState, outgoingResult requestresult.OutgoingExecutionResult)
	ExecutionAbort(run RunState)
}

type Service interface {
	CreateAdapter(ctx context.Context) ServiceAdapter
}

type DefaultService struct {
	Cache descriptor.Cache

	Manager machine.Manager

	lastPosition           call.ID
	eventSinkInProgressMap map[call.ID]*awaitedRun
	eventSinkMap           map[call.ID]*execution.EventSink
	eventSinkMapLock       sync.Mutex
}

func (r *DefaultService) stopExecution(id call.ID) error { // nolint
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	if val, ok := r.eventSinkMap[id]; ok {
		delete(r.eventSinkMap, id)
		val.InternalStop()
	}

	return nil
}

func (r *DefaultService) getExecutionSink(id call.ID) *execution.EventSink {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	return r.eventSinkMap[id]
}

func (r *DefaultService) awaitedRunFinish(id call.ID, possibleAbsent bool) {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	if run, ok := r.eventSinkInProgressMap[id]; ok {
		run.resumeFn()
		delete(r.eventSinkInProgressMap, id)
	} else if !possibleAbsent {
		panic(throw.IllegalState())
	}
}

func (r *DefaultService) awaitedRunAdd(sink *execution.EventSink, resumeFn func()) {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	run := &awaitedRun{
		run:      sink,
		resumeFn: resumeFn,
	}

	if _, ok := r.eventSinkInProgressMap[sink.ID()]; ok {
		panic(throw.IllegalState())
	}
	r.eventSinkInProgressMap[sink.ID()] = run
}

func (r *DefaultService) createExecutionSink(executionContext execution.Context) *execution.EventSink {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	id := r.lastPosition
	r.lastPosition++

	eventSink := execution.NewEventSink(id, executionContext)
	r.eventSinkMap[id] = eventSink

	return eventSink
}

func (r *DefaultService) destroyExecutionSink(id call.ID) {
	r.eventSinkMapLock.Lock()
	defer r.eventSinkMapLock.Unlock()

	sink, ok := r.eventSinkMap[id]
	if !ok {
		panic(throw.IllegalState())
	}

	if _, ok := r.eventSinkInProgressMap[id]; ok {
		panic(throw.IllegalState())
	}

	sink.InternalStop()
	delete(r.eventSinkMap, id)
}

func generateCallContext(
	ctx context.Context,
	id call.ID,
	execution execution.Context,
	classDesc descriptor.Class,
	codeDesc descriptor.Code,
) *call.LogicContext {
	request := execution.Request
	res := &call.LogicContext{
		ID:   id,
		Mode: call.Execute,

		Class: classDesc.HeadRef(),
		Code:  codeDesc.Ref(),

		Caller:      request.Caller.GetValue(),
		CallerClass: classDesc.HeadRef(),

		Request: execution.Incoming,

		TraceID: inslogger.TraceID(ctx),
	}

	if oDesc := execution.ObjectDescriptor; oDesc != nil {
		// should be the same as request.Object
		res.Callee = oDesc.HeadRef()
	} else {
		res.Callee = execution.Object
	}

	return res
}

func (r *DefaultService) executeMethod(
	ctx context.Context,
	id call.ID,
	eventSink *execution.EventSink,
) (
	*requestresult.RequestResult,
	error,
) {
	var (
		executionContext = eventSink.Context()
		request          = executionContext.Request

		objectDescriptor = executionContext.ObjectDescriptor
	)

	classReference := objectDescriptor.Class()
	if classReference.IsEmpty() {
		panic(throw.IllegalState())
	}

	classDescriptor, codeDescriptor, err := r.Cache.ByClassRef(ctx, classReference)
	if err != nil {
		return nil, throw.W(err, "couldn't get descriptors", ErrorDetail{DetailBadClassRef})
	}

	codeExecutor, err := r.Manager.GetExecutor(codeDescriptor.MachineType())
	if err != nil {
		return nil, throw.W(err, "couldn't get executor")
	}

	logicContext := generateCallContext(ctx, id, executionContext, classDescriptor, codeDescriptor)

	newData, result, err := codeExecutor.CallMethod(
		ctx, logicContext, codeDescriptor.Ref(), objectDescriptor.Memory(), request.CallSiteMethod, request.Arguments.GetBytes(),
	)
	if err != nil {
		return nil, throw.W(err, "execution error")
	}
	if len(result) == 0 {
		return nil, throw.E("return of method is empty")
	}
	if len(newData) == 0 {
		return nil, throw.E("object state is empty")
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
	eventSink *execution.EventSink,
) (
	*requestresult.RequestResult, error,
) {
	var (
		executionContext = eventSink.Context()
		request          = executionContext.Request
	)

	classDescriptor, codeDescriptor, err := r.Cache.ByClassRef(ctx, request.Callee.GetValue())
	if err != nil {
		return nil, throw.W(err, "couldn't get descriptors", ErrorDetail{DetailBadClassRef})
	}

	codeExecutor, err := r.Manager.GetExecutor(codeDescriptor.MachineType())
	if err != nil {
		return nil, throw.W(err, "couldn't get executor")
	}

	logicContext := generateCallContext(ctx, id, executionContext, classDescriptor, codeDescriptor)

	newState, executionResult, err := codeExecutor.CallConstructor(ctx, logicContext, codeDescriptor.Ref(), request.CallSiteMethod, request.Arguments.GetBytes())
	if err != nil {
		return nil, throw.W(err, "execution error")
	}
	if len(executionResult) == 0 {
		return nil, throw.E("return of constructor is empty")
	}

	// form and return executionResult
	res := requestresult.New(executionResult, executionContext.Object)
	if newState != nil {
		res.SetActivate(logicContext.CallerClass, newState)
	}

	return res, nil
}

func (r *DefaultService) execute(ctx context.Context, id call.ID) {
	executionSink := r.getExecutionSink(id)
	if executionSink == nil {
		panic(throw.Impossible())
	}

	var (
		result *requestresult.RequestResult
		err    error
	)

	defer func() {
		var (
			recoveredError = recover()
			possibleAbsent = false
		)

		// replace with custom error, not RecoverSlotPanicWithStack
		switch {
		case recoveredError != nil:
			// panic was catched
			err := throw.R(recoveredError, throw.E("ContractRunnerService panic"))
			executionSink.Error(err)
		case (result == nil && err == nil) || executionSink.IsAborted():
			// cancellation
			possibleAbsent = true
		default:
			// ok execution
		}

		r.awaitedRunFinish(id, possibleAbsent)
		r.destroyExecutionSink(id)
	}()

	switch executionSink.Context().Request.CallType {
	case rms.CallTypeMethod:
		result, err = r.executeMethod(ctx, id, executionSink)
	case rms.CallTypeConstructor:
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

func (r *DefaultService) runPrepare(execution execution.Context) *execution.EventSink {
	return r.createExecutionSink(execution)
}

func (r *DefaultService) runStart(run *execution.EventSink, resumeFn func()) {
	r.awaitedRunAdd(run, resumeFn)

	go r.execute(context.Background(), run.ID())
}

func (r *DefaultService) runContinue(run *execution.EventSink, resumeFn func()) {
	r.awaitedRunAdd(run, resumeFn)

	run.InputFlush()
}

func (r *DefaultService) runAbort(run *execution.EventSink, resumeFn func()) {
	run.InternalAbort()

	resumeFn()
}

func (r *DefaultService) ExecutionClassify(executionContext execution.Context) (contract.MethodIsolation, error) {
	var (
		request = executionContext.Request

		objectDescriptor = executionContext.ObjectDescriptor
	)

	classReference := objectDescriptor.Class()
	if classReference.IsEmpty() {
		panic(throw.IllegalState())
	}
	return r.classifyCall(executionContext.Context, classReference, request.CallSiteMethod)
}

func (r *DefaultService) classifyCall(ctx context.Context, classReference reference.Global, method string) (contract.MethodIsolation, error) {
	_, codeDescriptor, err := r.Cache.ByClassRef(ctx, classReference)
	if err != nil {
		return contract.MethodIsolation{}, throw.W(err, "couldn't get descriptors", ErrorDetail{DetailBadClassRef})
	}

	codeExecutor, err := r.Manager.GetExecutor(codeDescriptor.MachineType())
	if err != nil {
		return contract.MethodIsolation{}, throw.W(err, "couldn't get executor")
	}

	return codeExecutor.ClassifyMethod(ctx, codeDescriptor.Ref(), method)
}

func NewService() *DefaultService {
	return &DefaultService{
		Cache:   NewDescriptorsCache(),
		Manager: machine.NewManager(),

		lastPosition:           0,
		eventSinkMap:           make(map[call.ID]*execution.EventSink),
		eventSinkInProgressMap: make(map[call.ID]*awaitedRun),
		eventSinkMapLock:       sync.Mutex{},
	}
}

func (r *DefaultService) Init() error {
	exec := builtin.New(r)
	if err := r.Manager.RegisterExecutor(machinetype.Builtin, exec); err != nil {
		panic(throw.W(err, "failed to register executor", nil))
	}

	r.Cache.RegisterCallback(exec.GetDescriptor)

	return nil
}

func (r *DefaultService) CreateAdapter(ctx context.Context) ServiceAdapter {
	return createRunnerAdapter(ctx, r)
}
