// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package execute

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
)

/* -------- Utilities ------------- */

const MaxOutgoingSendCount = 3

type SMExecute struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCallRequest

	// internal data
	pendingConstructorFinished smachine.SyncLink

	isConstructor     bool
	execution         execution.Context
	objectSharedState object.SharedStateAccessor
	hasState          bool
	duplicateFinished bool

	// execution step
	executionNewState   *execution.Update
	outgoingResult      []byte
	deactivate          bool
	run                 runner.RunState
	newObjectDescriptor descriptor.Object

	methodIsolation contract.MethodIsolation

	// dependencies
	runner                runner.ServiceAdapter
	messageSender         messageSenderAdapter.MessageSender
	pulseSlot             *conveyor.PulseSlot
	authenticationService authentication.Service
	globalSemaphore       tool.RunnerLimiter

	outgoing            *payload.VCallRequest
	outgoingObject      reference.Global
	outgoingSentCounter int

	migrationHappened bool
	objectCatalog     object.Catalog

	delegationTokenSpec payload.CallDelegationToken
	stepAfterTokenGet   smachine.SlotStep

	findCallResponse *payload.VFindCallResponse
}

/* -------- Declaration ------------- */

var dSMExecuteInstance smachine.StateMachineDeclaration = &dSMExecute{}

type dSMExecute struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMExecute) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMExecute)

	injector.MustInject(&s.runner)
	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.objectCatalog)
	injector.MustInject(&s.authenticationService)
	injector.MustInject(&s.globalSemaphore)
}

func (*dSMExecute) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMExecute)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMExecute) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMExecuteInstance
}

func (s *SMExecute) prepareExecution(ctx context.Context) {
	s.execution.Context = ctx
	s.execution.Sequence = 0
	s.execution.Request = s.Payload
	s.execution.Pulse = s.pulseSlot.PulseData()

	if s.Payload.CallType == payload.CallTypeConstructor {
		s.isConstructor = true
		s.execution.Object = reference.NewSelf(s.Payload.CallOutgoing.GetLocal())
	} else {
		s.execution.Object = s.Payload.Callee
	}

	s.execution.Incoming = reference.NewRecordOf(s.Payload.Callee, s.Payload.CallOutgoing.GetLocal())
	s.execution.Outgoing = s.Payload.CallOutgoing

	s.execution.Isolation = contract.MethodIsolation{
		Interference: s.Payload.CallFlags.GetInterference(),
		State:        s.Payload.CallFlags.GetState(),
	}
}

func (s *SMExecute) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.Log().Trace("stop processing SMExecute since pulse was changed")
	return ctx.Stop()
}

func (s *SMExecute) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	s.prepareExecution(ctx.GetContext())

	ctx.SetDefaultMigration(s.migrationDefault)

	return ctx.Jump(s.stepCheckRequest)
}

func (s *SMExecute) stepCheckRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch s.Payload.CallType {
	case payload.CallTypeConstructor:
	case payload.CallTypeMethod:

	case payload.CallTypeInboundAPI, payload.CallTypeOutboundAPI, payload.CallTypeNotify,
		payload.CallTypeSAGA, payload.CallTypeParallel, payload.CallTypeSchedule:
		panic(throw.NotImplemented())
	default:
		panic(throw.IllegalValue())
	}

	return ctx.Jump(s.stepGetObject)
}

func (s *SMExecute) stepGetObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.objectSharedState = s.objectCatalog.GetOrCreate(ctx, s.execution.Object)

	if s.isConstructor && s.outgoingFromSlotPulse() {
		action := func(state *object.SharedState) {
			if state.GetState() == object.Unknown || state.GetState() == object.Missing {
				state.SetState(object.Empty)
			}
		}

		if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
			return stepUpdate
		}
	}

	return ctx.Jump(s.stepWaitObjectReady)
}

func (s *SMExecute) outgoingFromSlotPulse() bool {
	outgoingPulse := s.Payload.CallOutgoing.GetLocal().GetPulseNumber()
	slotPulse := s.pulseSlot.PulseData().GetPulseNumber()
	return outgoingPulse == slotPulse
}

func (s *SMExecute) intolerableCall() bool {
	return s.execution.Isolation.Interference == isolation.CallIntolerable
}

func (s *SMExecute) stepWaitObjectReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		semaphoreReadyToWork                smachine.SyncLink
		semaphorePendingConstructorFinished smachine.SyncLink

		objectDescriptor descriptor.Object
		objectState      object.State
	)

	action := func(state *object.SharedState) {
		semaphoreReadyToWork = state.ReadyToWork
		semaphorePendingConstructorFinished = state.PendingConstructorFinished

		objectDescriptor = s.getDescriptor(state)

		objectState = state.GetState()
	}

	if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
		return stepUpdate
	}

	if ctx.AcquireForThisStep(semaphoreReadyToWork).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	s.execution.ObjectDescriptor = objectDescriptor
	s.pendingConstructorFinished = semaphorePendingConstructorFinished

	if s.isConstructor {
		if objectState == object.Unknown {
			panic(throw.Impossible())
		}

		// default isolation for constructors
		s.methodIsolation = contract.ConstructorIsolation()

		return ctx.Jump(s.stepIsolationNegotiation)
	}

	switch objectState {
	case object.Unknown:
		panic(throw.Impossible())
	case object.Missing:
		s.prepareExecutionError(throw.E("object does not exist", struct {
			ObjectReference string
			State           object.State
		}{
			ObjectReference: s.execution.Object.String(),
			State:           objectState,
		}))
		return ctx.Jump(s.stepSendCallResult)
	case object.HasState:
		// ok
	}

	if s.pendingConstructorFinished.IsZero() {
		return ctx.Jump(s.stepIsolationNegotiation)
	}
	return ctx.Jump(s.stepWaitPendingConstructorFinished)
}

type markerPendingConstructorWait struct{}

func (s *SMExecute) stepWaitPendingConstructorFinished(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if ctx.Acquire(s.pendingConstructorFinished).IsNotPassed() {
		ctx.Log().Test(markerPendingConstructorWait{})
		return ctx.Sleep().ThenRepeat()
	}
	// set descriptor when pending constructor is finished
	var (
		objectDescriptor descriptor.Object
	)
	action := func(state *object.SharedState) {
		objectDescriptor = s.getDescriptor(state)
	}

	if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
		return stepUpdate
	}

	s.execution.ObjectDescriptor = objectDescriptor

	return ctx.Jump(s.stepIsolationNegotiation)
}

func (s *SMExecute) stepIsolationNegotiation(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// if ExecutionError was prepared
	if s.executionNewState != nil {
		return ctx.Jump(s.stepSendCallResult)
	}

	if s.methodIsolation.IsZero() {
		return s.runner.PrepareExecutionClassify(ctx, s.execution, func(isolation contract.MethodIsolation, err error) {
			if err != nil {
				s.prepareExecutionError(throw.W(err, "failed to classify method"))
			}
			s.methodIsolation = isolation
		}).DelayedStart().Sleep().ThenRepeat()
	}

	negotiatedIsolation, err := negotiateIsolation(s.methodIsolation, s.execution.Isolation)
	if err != nil {
		s.prepareExecutionError(throw.W(err, "failed to negotiate call isolation params", struct {
			methodIsolation contract.MethodIsolation
			callIsolation   contract.MethodIsolation
		}{
			methodIsolation: s.methodIsolation,
			callIsolation:   s.execution.Isolation,
		}))
		return ctx.Jump(s.stepSendCallResult)
	}

	// forbidden isolation
	// it requires special processing path that will be implemented later on
	if negotiatedIsolation.Interference == isolation.CallTolerable && negotiatedIsolation.State == isolation.CallValidated {
		panic(throw.NotImplemented())
	}

	s.execution.Isolation = negotiatedIsolation

	return ctx.Jump(s.stepDeduplicate)
}

func negotiateIsolation(methodIsolation, callIsolation contract.MethodIsolation) (contract.MethodIsolation, error) {
	if methodIsolation == callIsolation {
		return methodIsolation, nil
	}
	res := methodIsolation
	switch {
	case methodIsolation.Interference == callIsolation.Interference:
		// ok case
	case methodIsolation.Interference == isolation.CallIntolerable && callIsolation.Interference == isolation.CallTolerable:
		res.Interference = methodIsolation.Interference
	default:
		return contract.MethodIsolation{}, throw.IllegalValue()
	}
	switch {
	case methodIsolation.State == callIsolation.State:
		// ok case
	case methodIsolation.State == isolation.CallValidated && callIsolation.State == isolation.CallDirty:
		res.State = callIsolation.State
	default:
		return contract.MethodIsolation{}, throw.IllegalValue()
	}

	return res, nil
}

type DeduplicationAction byte

const (
	Stop DeduplicationAction = iota + 1
	SendResultAndStop
	DeduplicateThroughPreviousExecutor
	ContinueExecute
)

func (s *SMExecute) stepDeduplicate(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		deduplicateAction DeduplicationAction
		msg               *payload.VCallResult
		err               error
	)

	// deduplication algorithm
	action := func(state *object.SharedState) {
		deduplicateAction, msg, err = s.deduplicate(state)
	}

	if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
		return stepUpdate
	}

	if err != nil {
		panic(err)
	}

	switch deduplicateAction {
	case Stop:
		return ctx.Stop()
	case SendResultAndStop:
		s.sendResult(ctx, msg)
		return ctx.Stop()
	case DeduplicateThroughPreviousExecutor:
		return ctx.Jump(s.stepDeduplicateThroughPreviousExecutor)
	case ContinueExecute:
		return ctx.Jump(s.stepTakeLock)
	default:
		panic(throw.Unsupported())
	}
}

type DeduplicationBargeInKey struct {
	LookAt   pulse.Number
	Callee   reference.Global
	Outgoing reference.Global
}

func (s *SMExecute) stepDeduplicateThroughPreviousExecutor(ctx smachine.ExecutionContext) smachine.StateUpdate {
	prevPulse := s.pulseSlot.PrevOperationPulseNumber()
	if prevPulse.IsUnknown() {
		// unable to identify exact prev pulse
		panic(throw.NotImplemented())
	}

	msg := payload.VFindCallRequest{
		LookAt:   prevPulse,
		Callee:   s.execution.Object,
		Outgoing: s.execution.Outgoing,
	}

	bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*payload.VFindCallResponse)
		if !ok || res == nil {
			panic(throw.IllegalValue())
		}

		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			s.findCallResponse = res
			return ctx.WakeUp()
		}
	})

	bargeInKey := DeduplicationBargeInKey{
		LookAt:   prevPulse,
		Callee:   msg.Callee,
		Outgoing: msg.Outgoing,
	}

	if !ctx.PublishGlobalAliasAndBargeIn(bargeInKey, bargeInCallback) {
		return ctx.Error(throw.E("failed to publish bargeInCallback"))
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, affinity.DynamicRoleVirtualExecutor, s.execution.Object, prevPulse)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Jump(s.stepWaitFindCallResponse)
}

func (s *SMExecute) stepWaitFindCallResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.findCallResponse == nil {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(s.stepProcessFindCallResponse)
}

func (s *SMExecute) stepProcessFindCallResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch {
	case s.findCallResponse.Status == payload.CallStateFound && s.findCallResponse.CallResult == nil:
		ctx.Log().Trace("request found on previous executor, but there was no result")

		if s.isConstructor && (s.hasState || s.duplicateFinished) {
			panic(throw.NotImplemented())
		}

		return ctx.Stop()

	case s.findCallResponse.Status == payload.CallStateFound && s.findCallResponse.CallResult != nil:
		ctx.Log().Trace("request found on previous executor, resending result")

		target := s.Meta.Sender
		s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
			err := svc.SendTarget(goCtx, s.findCallResponse.CallResult, target)
			return func(ctx smachine.AsyncResultContext) {
				if err != nil {
					ctx.Log().Error("failed to send message", err)
				}
			}
		}).WithoutAutoWakeUp().Start()

		return ctx.Stop()

	case s.findCallResponse.Status == payload.CallStateMissing:
		fallthrough
	case s.findCallResponse.Status == payload.CallStateUnknown:
		if s.isConstructor {
			panic(throw.Impossible())
		}

		return ctx.Jump(s.stepTakeLock)
	default:
		panic(throw.Impossible())
	}

}

func (s *SMExecute) stepTakeLock(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var executionSemaphore smachine.SyncLink
	action := func(state *object.SharedState) {
		if s.execution.Isolation.Interference == isolation.CallIntolerable {
			executionSemaphore = state.UnorderedExecute
		} else {
			executionSemaphore = state.OrderedExecute
		}
	}

	if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
		return stepUpdate
	}

	if ctx.Acquire(executionSemaphore).IsNotPassed() {
		// wait for semaphore to be released
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepStartRequestProcessing)
}

func (s *SMExecute) getDescriptor(state *object.SharedState) descriptor.Object {
	switch s.execution.Isolation.State {
	case isolation.CallDirty:
		return state.DescriptorDirty()
	case isolation.CallValidated:
		return state.DescriptorValidated()
	default:
		panic(throw.IllegalState())
	}

}

func (s *SMExecute) stepStartRequestProcessing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		objectDescriptor descriptor.Object
		isDeactivated    bool
	)
	action := func(state *object.SharedState) {
		if !state.KnownRequests.SetActive(s.execution.Isolation.Interference, s.execution.Outgoing) {
			// if we come here then request should be in RequestStarted
			// if it is not it is either somehow lost or it is already processing
			panic(throw.Impossible())
		}

		objectDescriptor = s.getDescriptor(state)
		if state.GetState() == object.Inactive || (objectDescriptor != nil && objectDescriptor.Deactivated()) {
			isDeactivated = true
			return
		}
	}

	if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
		return stepUpdate
	}

	if isDeactivated {
		s.prepareExecutionError(throw.E("try to call method on deactivated object", struct {
			ObjectReference string
		}{
			ObjectReference: s.execution.Object.String(),
		}))
		return ctx.Jump(s.stepSendCallResult)
	}

	if s.execution.Isolation.State == isolation.CallValidated && s.execution.ObjectDescriptor == nil {
		panic(throw.NotImplemented())
	}

	ctx.SetDefaultMigration(s.migrateDuringExecution)
	s.execution.ObjectDescriptor = objectDescriptor

	return ctx.Jump(s.stepExecuteStart)
}

func (s *SMExecute) migrateDuringExecution(ctx smachine.MigrationContext) smachine.StateUpdate {
	if s.migrationHappened && s.delegationTokenSpec.IsZero() {
		return ctx.Error(throw.E("failed to get token in previous migration"))
	}

	s.migrationHappened = true

	s.stepAfterTokenGet = ctx.AffectedStep()

	return ctx.Jump(s.stepGetDelegationToken)
}

func (s *SMExecute) stepGetDelegationToken(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var requestPayload = payload.VDelegatedCallRequest{
		Callee:         s.execution.Object,
		CallFlags:      payload.BuildCallFlags(s.execution.Isolation.Interference, s.execution.Isolation.State),
		CallOutgoing:   s.execution.Outgoing,
		CallIncoming:   s.execution.Incoming,
		DelegationSpec: s.delegationTokenSpec,
	}

	// reset token
	s.delegationTokenSpec = payload.CallDelegationToken{}

	subroutineSM := &SMDelegatedTokenRequest{Meta: s.Meta, RequestPayload: requestPayload}
	return ctx.CallSubroutine(subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		if subroutineSM.response == nil {
			panic(throw.IllegalState())
		}
		s.delegationTokenSpec = subroutineSM.response.ResponseDelegationSpec
		if s.outgoingSentCounter > 0 {
			return ctx.Jump(s.stepSendOutgoing)
		}
		return ctx.JumpExt(s.stepAfterTokenGet)
	})
}

func (s *SMExecute) stepExecuteStart(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.run = nil
	return s.runner.PrepareExecutionStart(ctx, s.execution, func(state runner.RunState) {
		if state == nil {
			panic(throw.IllegalValue())
		}
		s.run = state

		s.executionNewState = state.GetResult()
		if s.executionNewState == nil {
			panic(throw.IllegalValue())
		}
	}).DelayedStart().ThenJump(s.StepWaitExecutionResult)
}

func (s *SMExecute) StepWaitExecutionResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.executionNewState == nil {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(s.stepExecuteDecideNextStep)
}

func (s *SMExecute) stepExecuteDecideNextStep(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.executionNewState == nil {
		panic(throw.IllegalState())
	}

	newState := s.executionNewState

	switch newState.Type {
	case execution.Done:
		// send VCallResult here
		return ctx.Jump(s.stepSaveNewObject)
	case execution.Error:
		if d := new(runner.ErrorDetail); throw.FindDetail(newState.Error, d) {
			switch d.Type {
			case runner.DetailBadClassRef, runner.DetailEmptyClassRef:
				s.prepareExecutionError(throw.E("bad class reference"))
			}
		} else {
			s.prepareExecutionError(throw.W(newState.Error, "failed to execute request"))
		}
		ctx.Log().Warn(struct {
			string
			err error
		}{"Failed to execute request", newState.Error})
		return ctx.Jump(s.stepExecuteAborted)
	case execution.Abort:
		err := throw.E("execution aborted")
		ctx.Log().Warn(err)
		s.prepareExecutionError(err)
		return ctx.Jump(s.stepExecuteAborted)
	case execution.OutgoingCall:
		return ctx.Jump(s.stepExecuteOutgoing)
	default:
		panic(throw.IllegalValue())
	}
}

func (s *SMExecute) prepareExecutionError(err error) {
	resultWithErr, err := foundation.MarshalMethodErrorResult(err)
	if err != nil {
		panic(throw.W(err, "can't create error result"))
	}

	s.executionNewState = &execution.Update{
		Type:     execution.Error,
		Error:    err,
		Result:   requestresult.New(resultWithErr, s.outgoingObject),
		Outgoing: nil,
	}
}

func (s *SMExecute) prepareOutgoingError(err error) {
	resultWithErr, err := foundation.MarshalMethodErrorResult(err)
	if err != nil {
		panic(throw.W(err, "can't create error result"))
	}

	s.outgoingResult = resultWithErr
}

func (s *SMExecute) stepExecuteOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	pulseNumber := s.pulseSlot.CurrentPulseNumber()

	switch outgoing := s.executionNewState.Outgoing.(type) {
	case execution.Deactivate:
		if s.intolerableCall() {
			err := throw.E("interference violation: deactivate call from intolerable call")
			ctx.Log().Warn(err)
			s.prepareOutgoingError(err)
			return ctx.Jump(s.stepExecuteContinue)
		}
		s.deactivate = true
	case execution.CallConstructor:
		if s.intolerableCall() {
			err := throw.E("interference violation: constructor call from unordered call")
			ctx.Log().Warn(err)
			s.prepareOutgoingError(err)
			return ctx.Jump(s.stepExecuteContinue)
		}

		s.outgoing = outgoing.ConstructVCallRequest(s.execution)
		s.outgoing.CallOutgoing = reference.NewRecordOf(s.outgoing.Caller, gen.UniqueLocalRefWithPulse(pulseNumber))
		s.execution.Sequence++
		s.outgoing.CallSequence = s.execution.Sequence
		s.outgoingObject = s.outgoing.CallOutgoing
	case execution.CallMethod:
		if s.intolerableCall() && outgoing.Interference() == isolation.CallTolerable {
			err := throw.E("interference violation: ordered call from unordered call")
			ctx.Log().Warn(err)
			s.prepareOutgoingError(err)
			return ctx.Jump(s.stepExecuteContinue)
		}

		s.outgoing = outgoing.ConstructVCallRequest(s.execution)
		s.outgoing.CallOutgoing = reference.NewRecordOf(s.outgoing.Caller, gen.UniqueLocalRefWithPulse(pulseNumber))
		s.execution.Sequence++
		s.outgoing.CallSequence = s.execution.Sequence
		s.outgoingObject = s.outgoing.Callee
	default:
		panic(throw.IllegalValue())
	}

	if s.outgoing != nil {
		return ctx.Jump(s.stepSendOutgoing)
	}

	return ctx.Jump(s.stepExecuteContinue)
}

func (s *SMExecute) stepExecuteAborted(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.Log().Warn("aborting execution")
	return s.runner.PrepareExecutionAbort(ctx, s.run).DelayedStart().ThenJump(s.stepSendCallResult)
}

func (s *SMExecute) stepSendOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.outgoingSentCounter == 0 {
		bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
			res, ok := param.(*payload.VCallResult)
			if !ok || res == nil {
				panic(throw.IllegalValue())
			}

			return func(ctx smachine.BargeInContext) smachine.StateUpdate {
				s.outgoingResult = res.ReturnArguments

				return ctx.WakeUp()
			}
		})

		outgoingRef := s.outgoing.CallOutgoing

		if !ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bargeInCallback) {
			return ctx.Error(throw.E("failed to publish bargeInCallback"))
		}
	} else {
		if s.outgoingSentCounter >= MaxOutgoingSendCount {
			// TODO when CallSummary will live longer than one pulse it needs to be updated
			s.sendDelegatedRequestFinished(ctx, nil)
			return ctx.Error(throw.E("outgoing retries limit"))
		}

		s.outgoing.CallRequestFlags = payload.BuildCallRequestFlags(payload.SendResultDefault, payload.RepeatedCall)
	}

	s.outgoing.DelegationSpec = s.getToken()

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, s.outgoing, affinity.DynamicRoleVirtualExecutor, s.outgoingObject, s.pulseSlot.CurrentPulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	s.outgoingSentCounter++

	// someone else can process other requests while we  waiting for outgoing results
	ctx.Release(s.globalSemaphore.PartialLink())

	// we'll wait for barge-in WakeUp here, not adapter
	return ctx.Sleep().ThenJump(s.stepTakeLockAfterOutgoing)
}

func (s *SMExecute) stepTakeLockAfterOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// parent semaphore was released in stepSendOutgoing
	// acquire it again
	if ctx.Acquire(s.globalSemaphore.PartialLink()).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepExecuteContinue)
}

func (s *SMExecute) stepExecuteContinue(ctx smachine.ExecutionContext) smachine.StateUpdate {
	outgoingResult := s.outgoingResult
	switch s.executionNewState.Outgoing.(type) {
	case execution.CallConstructor, execution.CallMethod:
		if outgoingResult == nil {
			panic(throw.IllegalValue())
		}
	}

	// unset all outgoing fields in case we have new outgoing request
	s.outgoingSentCounter = 0
	s.outgoingObject = reference.Global{}
	s.outgoing = nil
	s.outgoingResult = []byte{}
	ctx.SetDefaultMigration(s.migrateDuringExecution)

	s.executionNewState = nil

	executionResult := requestresult.NewOutgoingExecutionResult(outgoingResult, nil)
	return s.runner.PrepareExecutionContinue(ctx, s.run, executionResult, func() {
		if s.run == nil {
			panic(throw.IllegalState())
		}

		s.executionNewState = s.run.GetResult()
		if s.executionNewState == nil {
			panic(throw.IllegalState())
		}
	}).DelayedStart().ThenJump(s.StepWaitExecutionResult)
}

func (s *SMExecute) stepSaveNewObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.isIntolerableCallChangeState() {
		s.prepareExecutionError(throw.E("intolerable call trying to change object state"))
		return ctx.Jump(s.stepSendCallResult)
	}

	if s.deactivate {
		oldRequestResult := s.executionNewState.Result

		// we should overwrite old side effect with new one - deactivation of object
		s.executionNewState.Result = requestresult.New(oldRequestResult.Result(), oldRequestResult.ObjectReference())
		s.executionNewState.Result.SetDeactivate(s.execution.ObjectDescriptor)
	}

	switch s.executionNewState.Result.Type() {
	case requestresult.SideEffectNone:
	case requestresult.SideEffectActivate:
		_, class, memory := s.executionNewState.Result.Activate()
		s.newObjectDescriptor = s.makeNewDescriptor(class, memory, false)
	case requestresult.SideEffectAmend:
		_, class, memory := s.executionNewState.Result.Amend()
		s.newObjectDescriptor = s.makeNewDescriptor(class, memory, false)
	case requestresult.SideEffectDeactivate:
		class, memory := s.executionNewState.Result.Deactivate()
		s.newObjectDescriptor = s.makeNewDescriptor(class, memory, true)
	default:
		panic(throw.IllegalValue())
	}

	if s.migrationHappened || s.newObjectDescriptor == nil {
		return ctx.Jump(s.stepSendCallResult)
	}

	action := func(state *object.SharedState) {
		state.SetDescriptorDirty(s.newObjectDescriptor)

		switch state.GetState() {
		case object.HasState:
			// ok
		case object.Empty, object.Missing:
			state.SetState(object.HasState)
		default:
			panic(throw.IllegalState())
		}
	}

	if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
		return stepUpdate
	}

	return ctx.Jump(s.stepSendCallResult)
}

func (s *SMExecute) isIntolerableCallChangeState() bool {
	return s.intolerableCall() && (s.deactivate || s.executionNewState.Result.Type() != requestresult.SideEffectNone)
}

func (s *SMExecute) stepAwaitSMCallSummary(ctx smachine.ExecutionContext) smachine.StateUpdate {
	syncAccessor, ok := callsummary.GetSummarySMSyncAccessor(ctx, s.execution.Object)

	if ok {
		var syncLink smachine.SyncLink

		switch syncAccessor.Prepare(func(link *smachine.SyncLink) {
			syncLink = *link
		}).TryUse(ctx).GetDecision() {
		case smachine.NotPassed:
			return ctx.WaitShared(syncAccessor.SharedDataLink).ThenRepeat()
		case smachine.Passed:
			// go further
		default:
			panic(throw.Impossible())
		}

		if ctx.AcquireForThisStep(syncLink).IsNotPassed() {
			return ctx.Sleep().ThenRepeat()
		}
	}

	return ctx.Jump(s.stepPublishDataCallSummary)
}

func (s *SMExecute) stepPublishDataCallSummary(ctx smachine.ExecutionContext) smachine.StateUpdate {
	callSummaryAccessor, ok := callsummary.GetSummarySMSharedAccessor(ctx, s.pulseSlot.PulseData().PulseNumber)

	if ok {
		action := func(sharedCallSummary *callsummary.SharedCallSummary) {
			sharedCallSummary.Requests.AddObjectCallResult(
				s.execution.Object,
				s.execution.Outgoing,
				s.execution.Result,
			)
		}

		switch callSummaryAccessor.Prepare(action).TryUse(ctx).GetDecision() {
		case smachine.NotPassed:
			return ctx.WaitShared(callSummaryAccessor.SharedDataLink).ThenRepeat()
		case smachine.Passed:
			// go further
		default:
			panic(throw.Impossible())
		}
	}

	return ctx.Jump(s.stepSendDelegatedRequestFinished)
}

func (s *SMExecute) stepSendDelegatedRequestFinished(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var lastState *payload.ObjectState = nil

	if s.newObjectDescriptor != nil {
		class, err := s.newObjectDescriptor.Class()
		if err != nil {
			panic(throw.W(err, "failed to get class from descriptor", nil))
		}

		lastState = &payload.ObjectState{
			Reference:   s.executionNewState.Result.ObjectStateID,
			State:       s.executionNewState.Result.Memory,
			Class:       class,
			Deactivated: s.executionNewState.Result.SideEffectType == requestresult.SideEffectDeactivate,
		}
	}

	s.sendDelegatedRequestFinished(ctx, lastState)

	return ctx.Stop()
}

func (s *SMExecute) sendDelegatedRequestFinished(ctx smachine.ExecutionContext, lastState *payload.ObjectState) {
	msg := payload.VDelegatedRequestFinished{
		CallType:       s.Payload.CallType,
		CallFlags:      s.Payload.CallFlags,
		Callee:         s.execution.Object,
		CallOutgoing:   s.execution.Outgoing,
		CallIncoming:   s.execution.Incoming,
		DelegationSpec: s.getToken(),
		LatestState:    lastState,
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, affinity.DynamicRoleVirtualExecutor, s.execution.Object, s.pulseSlot.CurrentPulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()
}

func (s *SMExecute) makeNewDescriptor(class reference.Global, memory []byte, deactivated bool) descriptor.Object {
	var prevStateIDBytes []byte
	objDescriptor := s.execution.ObjectDescriptor
	if objDescriptor != nil {
		prevStateIDBytes = objDescriptor.StateID().AsBytes()
	}

	objectRefBytes := s.execution.Object.AsBytes()
	stateHash := append(memory, objectRefBytes...)
	stateHash = append(stateHash, prevStateIDBytes...)

	stateID := NewStateID(s.pulseSlot.PulseData().GetPulseNumber(), stateHash)
	return descriptor.NewObject(
		s.execution.Object,
		stateID,
		class,
		memory,
		deactivated,
	)
}

func (s *SMExecute) stepSendCallResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		executionNewState = s.executionNewState.Result
		executionResult   = executionNewState.Result()
	)

	msg := payload.VCallResult{
		CallType:        s.Payload.CallType,
		CallFlags:       s.Payload.CallFlags,
		Caller:          s.Payload.Caller,
		Callee:          s.execution.Object,
		CallOutgoing:    s.execution.Outgoing,
		CallIncoming:    s.execution.Incoming,
		ReturnArguments: executionResult,
		DelegationSpec:  s.getToken(),
	}

	// save result for future pass to SMObject
	s.execution.Result = &msg

	s.sendResult(ctx, &msg)

	return ctx.Jump(s.stepFinishRequest)
}

func (s *SMExecute) stepFinishRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.migrationHappened {
		// publish call result only if present
		if s.execution.Result != nil {
			return ctx.Jump(s.stepAwaitSMCallSummary)
		}
		return ctx.Jump(s.stepSendDelegatedRequestFinished)
	}

	action := func(state *object.SharedState) {
		state.FinishRequest(s.execution.Isolation, s.execution.Outgoing, s.execution.Result)
	}

	if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
		return stepUpdate
	}

	return ctx.Stop()
}

func NewStateID(pn pulse.Number, data []byte) reference.Local {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	hash := hasher.Hash(data)
	return reference.NewLocal(pn, 0, reference.BytesToLocalHash(hash))
}

func (s *SMExecute) getToken() payload.CallDelegationToken {
	if s.authenticationService != nil && !s.authenticationService.HasToSendToken(s.delegationTokenSpec) {
		return payload.CallDelegationToken{}
	}
	return s.delegationTokenSpec
}

func (s *SMExecute) sendResult(ctx smachine.ExecutionContext, message payload.Marshaler) {
	target := s.Meta.Sender

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendTarget(goCtx, message, target)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()
}

func (s *SMExecute) shareObjectAccess(
	ctx smachine.ExecutionContext,
	action func(state *object.SharedState),
) smachine.StateUpdate {
	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Passed:
		return smachine.StateUpdate{}
	default:
		panic(throw.Impossible())
	}
}

func (s *SMExecute) deduplicate(state *object.SharedState) (DeduplicationAction, *payload.VCallResult, error) {
	// if we can not add to request table, this mean that we already have operation in progress or completed
	if !state.KnownRequests.Add(s.execution.Isolation.Interference, s.execution.Outgoing) {
		results := state.KnownRequests.GetResults()

		summary, ok := results[s.execution.Outgoing]

		// get result only if exist, if result == nil this mean that other SM now during execution
		if ok && summary.Result != nil {
			return SendResultAndStop, summary.Result, nil
		}

		// stop current sm because other sm still in progress and not send result
		return Stop, nil, nil
	}

	// deduplicate through pending table
	if !s.outgoingFromSlotPulse() {
		pendingList := state.PendingTable.GetList(s.execution.Isolation.Interference)
		filledTable := uint8(pendingList.Count()) == state.PreviousExecutorOrderedPendingCount
		isActive, isDuplicate := pendingList.GetState(s.execution.Outgoing)

		switch state.GetState() {
		case object.HasState, object.Inactive:
			s.hasState = true
		default:
			s.hasState = false
		}
		s.duplicateFinished = isDuplicate && !isActive

		switch {
		case isDuplicate && isActive:
			return Stop, nil, nil
		case s.isConstructor && state.GetState() == object.Missing:
			return ContinueExecute, nil, nil
		case s.isConstructor && !s.hasState && filledTable && !isDuplicate:
			return Stop, nil, throw.NotImplemented()
		}

		return DeduplicateThroughPreviousExecutor, nil, nil
	}

	return ContinueExecute, nil, nil
}
