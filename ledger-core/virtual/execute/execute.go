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
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute/shared"
	"github.com/insolar/assured-ledger/ledger-core/virtual/lmn"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
)

/* -------- Utilities ------------- */

const MaxOutgoingSendCount = 3

type SMExecute struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VCallRequest

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
	outgoingVCallResult *rms.VCallResult
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
	memoryCache           memoryCacheAdapter.MemoryCache

	outgoing            *rms.VCallRequest
	outgoingObject      reference.Global
	outgoingSentCounter int

	migrationHappened bool
	objectCatalog     object.Catalog

	delegationTokenSpec rms.CallDelegationToken
	stepAfterTokenGet   smachine.SlotStep

	findCallResponse *rms.VFindCallResponse

	// registration in LMN
	lmnLastFilamentRef         reference.Global
	lmnLastLifelineRef         reference.Global
	lmnSafeResponseCounter     shared.SafeResponseCounter
	lmnSafeResponseCounterLink smachine.SharedDataLink
	incomingRegistered         bool
	referenceBuilder           lmn.RecordReferenceBuilderService
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
	injector.MustInject(&s.memoryCache)
	injector.MustInject(&s.objectCatalog)
	injector.MustInject(&s.authenticationService)
	injector.MustInject(&s.globalSemaphore)
	injector.MustInject(&s.referenceBuilder)
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
	currentPulse := s.pulseSlot.CurrentPulseNumber()

	s.execution.Context = ctx
	s.execution.Sequence = 0
	s.execution.Request = s.Payload
	s.execution.Pulse = s.pulseSlot.PulseData()

	if s.Payload.CallType == rms.CallTypeConstructor {
		s.isConstructor = true
		s.execution.Object = lmn.GetLifelineAnticipatedReference(s.referenceBuilder, s.Payload, currentPulse)
	} else {
		s.execution.Object = s.Payload.Callee.GetValue()
	}

	s.execution.Incoming = reference.NewRecordOf(s.Payload.Callee.GetValue(), s.Payload.CallOutgoing.GetValue().GetLocal())
	s.execution.Outgoing = s.Payload.CallOutgoing.GetValue()

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

	s.lmnSafeResponseCounterLink = ctx.Share(&s.lmnSafeResponseCounter, 0)

	return ctx.Jump(s.stepCheckRequest)
}

func (s *SMExecute) stepCheckRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.pulseSlot.State() != conveyor.Present {
		panic(throw.IllegalState())
	}

	switch s.Payload.CallType {
	case rms.CallTypeConstructor:
	case rms.CallTypeMethod:

	case rms.CallTypeInboundAPI, rms.CallTypeOutboundAPI, rms.CallTypeNotify,
		rms.CallTypeSAGA, rms.CallTypeParallel, rms.CallTypeSchedule:
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
	outgoingPulse := s.Payload.CallOutgoing.GetPulseOfLocal()
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
		msg               *rms.VCallResult
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

	msg := rms.VFindCallRequest{
		LookAt:   prevPulse,
		Callee:   rms.NewReference(s.execution.Object),
		Outgoing: rms.NewReference(s.execution.Outgoing),
	}

	bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*rms.VFindCallResponse)
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
		Callee:   s.execution.Object,
		Outgoing: s.execution.Outgoing,
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
	case s.findCallResponse.Status == rms.CallStateFound && s.findCallResponse.CallResult == nil:
		ctx.Log().Trace("request found on previous executor, but there was no result")

		if s.isConstructor && (s.hasState || s.duplicateFinished) {
			panic(throw.NotImplemented())
		}

		return ctx.Stop()

	case s.findCallResponse.Status == rms.CallStateFound && s.findCallResponse.CallResult != nil:
		ctx.Log().Trace("request found on previous executor, resending result")

		target := s.Meta.Sender.GetValue()
		s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
			err := svc.SendTarget(goCtx, s.findCallResponse.CallResult, target)
			return func(ctx smachine.AsyncResultContext) {
				if err != nil {
					ctx.Log().Error("failed to send message", err)
				}
			}
		}).WithoutAutoWakeUp().Start()

		return ctx.Stop()

	case s.findCallResponse.Status == rms.CallStateMissing:
		fallthrough
	case s.findCallResponse.Status == rms.CallStateUnknown:
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

	if s.isConstructor {
		return ctx.Jump(s.stepRegisterObjectLifeLine)
	}

	return ctx.Jump(s.stepStartRequestProcessing)
}

func (s *SMExecute) stepRegisterObjectLifeLine(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subroutineSM := s.constructSubSMRegister(RegisterLifeLine)
	subroutineSM.Incoming = s.Payload

	ctx.Release(s.globalSemaphore.PartialLink())

	return ctx.CallSubroutine(&subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		if ctx.GetError() != nil {
			// TODO: we should understand here what's happened, but for now we'll drop request execution here
			return ctx.Error(ctx.GetError())
		}

		if subroutineSM.NewObjectRef != s.execution.Object {
			// TODO: do nothing for now, later we should replace that mechanism
			panic(throw.NotImplemented())
		}

		s.lmnLastLifelineRef = reference.NewRecordOf(s.execution.Object, subroutineSM.NewLastLifelineRef.GetLocal())

		return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
			if ctx.Acquire(s.globalSemaphore.PartialLink()).IsNotPassed() {
				return ctx.Sleep().ThenRepeat()
			}

			return ctx.Jump(s.stepRegisterObjectLifelineAfter)
		})
	})
}

func (s *SMExecute) stepRegisterObjectLifelineAfter(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO: we must set initial (empty) descriptor
	action := func(state *object.SharedState) {
		state.SetDescriptorDirty(descriptor.NewObject(
			s.execution.Object,
			s.lmnLastLifelineRef.GetLocal(),
			s.Payload.Callee.GetValue(),
			[]byte(""),
			false,
		))
	}

	if stepUpdate := s.shareObjectAccess(ctx, action); !stepUpdate.IsEmpty() {
		return stepUpdate
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

	if objectDescriptor == nil {
		panic(throw.IllegalState())
	}

	s.execution.ObjectDescriptor = objectDescriptor
	s.lmnLastLifelineRef = reference.NewRecordOf(objectDescriptor.HeadRef(), objectDescriptor.StateID())

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
	var requestPayload = rms.VDelegatedCallRequest{
		Callee:         rms.NewReference(s.execution.Object),
		CallFlags:      rms.BuildCallFlags(s.execution.Isolation.Interference, s.execution.Isolation.State),
		CallOutgoing:   rms.NewReference(s.execution.Outgoing),
		CallIncoming:   rms.NewReference(s.execution.Incoming),
		DelegationSpec: s.delegationTokenSpec,
	}

	// reset token
	s.delegationTokenSpec = rms.CallDelegationToken{}

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
		return ctx.Jump(s.stepWaitSafeAnswersRelease)
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
		s.execution.Sequence++
		s.outgoing.CallSequence = s.execution.Sequence
	case execution.CallMethod:
		if s.intolerableCall() && outgoing.Interference() == isolation.CallTolerable {
			err := throw.E("interference violation: ordered call from unordered call")
			ctx.Log().Warn(err)
			s.prepareOutgoingError(err)
			return ctx.Jump(s.stepExecuteContinue)
		}

		s.outgoing = outgoing.ConstructVCallRequest(s.execution)
		s.execution.Sequence++
		s.outgoing.CallSequence = s.execution.Sequence
		s.outgoingObject = s.outgoing.Callee.GetValue()
	default:
		panic(throw.IllegalValue())
	}

	if s.outgoing != nil {
		return ctx.Jump(s.stepRegisterOutgoing)
	}

	return ctx.Jump(s.stepExecuteContinue)
}

func (s *SMExecute) stepExecuteAborted(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.Log().Warn("aborting execution")
	return s.runner.PrepareExecutionAbort(ctx, s.run).DelayedStart().ThenJump(s.stepSendCallResult)
}

func (s *SMExecute) stepRegisterOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// someone else can process other requests while we registering outgoing and waiting for outgoing result
	ctx.Release(s.globalSemaphore.PartialLink())

	subroutineSM := s.constructSubSMRegister(RegisterOutgoingRequest)
	subroutineSM.Outgoing = s.outgoing

	return ctx.CallSubroutine(&subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		s.lmnLastLifelineRef = subroutineSM.NewLastLifelineRef
		s.lmnLastFilamentRef = subroutineSM.NewLastFilamentRef

		s.outgoing.CallOutgoing = rms.NewReference(s.lmnLastFilamentRef)

		return ctx.Jump(s.stepSendOutgoing)
	})
}

func (s *SMExecute) stepSendOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	currentPulse := s.pulseSlot.CurrentPulseNumber()

	if s.outgoing.CallType == rms.CallTypeConstructor {
		s.outgoingObject = lmn.GetLifelineAnticipatedReference(s.referenceBuilder, s.outgoing, currentPulse)
	}

	if s.outgoingSentCounter == 0 {
		bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
			res, ok := param.(*rms.VCallResult)
			if !ok || res == nil {
				panic(throw.IllegalValue())
			}

			return func(ctx smachine.BargeInContext) smachine.StateUpdate {
				s.outgoingVCallResult = res
				s.outgoingResult = res.ReturnArguments.GetBytes()

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

		s.outgoing.CallRequestFlags = rms.BuildCallRequestFlags(rms.SendResultDefault, rms.RepeatedCall)
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

	// we'll wait for barge-in WakeUp here, not adapter
	return ctx.Sleep().ThenJump(s.stepWaitAndRegisterOutgoingResult)
}

func (s *SMExecute) stepWaitAndRegisterOutgoingResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.outgoingVCallResult == nil {
		return ctx.Sleep().ThenRepeat()
	}

	subroutineSM := s.constructSubSMRegister(RegisterOutgoingResult)
	subroutineSM.OutgoingResult = s.outgoingVCallResult

	return ctx.CallSubroutine(&subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		s.lmnLastFilamentRef = subroutineSM.NewLastFilamentRef

		return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
			// parent semaphore was released in stepSendOutgoing
			// acquire it again
			if ctx.Acquire(s.globalSemaphore.PartialLink()).IsNotPassed() {
				return ctx.Sleep().ThenRepeat()
			}

			return ctx.Jump(s.stepExecuteContinue)
		})
	})
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
	s.outgoingVCallResult = nil
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

func (s *SMExecute) stepWaitSafeAnswersRelease(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.Release(s.globalSemaphore.PartialLink())

	if s.isIntolerableCallChangeState() {
		s.prepareExecutionError(throw.E("intolerable call trying to change object state"))
		return ctx.Jump(s.stepSendCallResult)
	}

	// waiting for all save responses to be there
	return ctx.Jump(s.stepWaitSafeAnswers)
}

func (s *SMExecute) stepWaitSafeAnswers(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// waiting for all save responses to be there
	stateUpdate := shared.CounterAwaitZero(ctx, s.lmnSafeResponseCounterLink)
	if !stateUpdate.IsEmpty() {
		return stateUpdate
	}

	// now it's time to write result
	return ctx.Jump(s.stepSaveExecutionResult)
}

func (s *SMExecute) stepSaveExecutionResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subroutineSM := s.constructSubSMRegister(RegisterIncomingResult)
	subroutineSM.IncomingResult = s.executionNewState

	return ctx.CallSubroutine(&subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		s.lmnLastLifelineRef = subroutineSM.NewLastLifelineRef
		s.lmnLastFilamentRef = subroutineSM.NewLastFilamentRef

		return ctx.Jump(s.stepSaveNewObject)
	})
}

func (s *SMExecute) stepSaveNewObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !ctx.Acquire(s.globalSemaphore.PartialLink()).IsPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	if s.deactivate {
		oldRequestResult := s.executionNewState.Result

		// we should overwrite old side effect with new one - deactivation of object
		s.executionNewState.Result = requestresult.New(oldRequestResult.Result(), oldRequestResult.ObjectReference())
		s.executionNewState.Result.SetDeactivate(s.execution.ObjectDescriptor)
	}

	switch s.executionNewState.Result.Type() {
	case requestresult.SideEffectNone:
		// do nothing
	case requestresult.SideEffectActivate:
		class, memory := s.executionNewState.Result.Activate()
		s.newObjectDescriptor = s.makeNewDescriptor(class, memory, false)
	case requestresult.SideEffectAmend:
		class, memory := s.executionNewState.Result.Amend()
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

	s.updateMemoryCache(ctx, s.newObjectDescriptor)

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

func (s *SMExecute) updateMemoryCache(ctx smachine.ExecutionContext, object descriptor.Object) {
	s.memoryCache.PrepareAsync(ctx, func(ctx context.Context, svc memorycache.Service) smachine.AsyncResultFunc {
		ref := reference.NewRecordOf(object.HeadRef(), object.StateID())

		err := svc.Set(ctx, ref, object)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to set dirty memory", err)
			}
		}
	}).WithoutAutoWakeUp().Start()
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
	var lastState *rms.ObjectState = nil

	if s.newObjectDescriptor != nil {
		class, err := s.newObjectDescriptor.Class()
		if err != nil {
			panic(throw.W(err, "failed to get class from descriptor", nil))
		}

		lastState = &rms.ObjectState{
			Reference:   rms.NewReferenceLocal(s.lmnLastLifelineRef),
			Memory:      rms.NewBytes(s.executionNewState.Result.Memory),
			Class:       rms.NewReference(class),
			Deactivated: s.executionNewState.Result.SideEffectType == requestresult.SideEffectDeactivate,
		}
	}

	s.sendDelegatedRequestFinished(ctx, lastState)

	return ctx.Stop()
}

func (s *SMExecute) sendDelegatedRequestFinished(ctx smachine.ExecutionContext, lastState *rms.ObjectState) {
	msg := rms.VDelegatedRequestFinished{
		CallType:       s.Payload.CallType,
		CallFlags:      s.Payload.CallFlags,
		Callee:         rms.NewReference(s.execution.Object),
		CallOutgoing:   rms.NewReference(s.execution.Outgoing),
		CallIncoming:   rms.NewReference(s.execution.Incoming),
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
	return descriptor.NewObject(s.execution.Object, s.lmnLastLifelineRef.GetLocal(), class, memory, deactivated)
}

func (s *SMExecute) stepSendCallResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		executionNewState = s.executionNewState.Result
		executionResult   = executionNewState.Result()
	)

	msg := rms.VCallResult{
		CallType:        s.Payload.CallType,
		CallFlags:       s.Payload.CallFlags,
		Caller:          s.Payload.Caller,
		Callee:          rms.NewReference(s.execution.Object),
		CallOutgoing:    rms.NewReference(s.execution.Outgoing),
		CallIncoming:    rms.NewReference(s.execution.Incoming),
		ReturnArguments: rms.NewBytes(executionResult),
		DelegationSpec:  s.getToken(),
	}

	// save result for future pass to SMObject
	s.execution.Result = &msg

	s.sendResult(ctx, &msg)

	return ctx.Jump(s.stepFinishRequest)
}

func (s *SMExecute) stepFinishRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch {
	case !s.migrationHappened:
		//
	case s.execution.Result != nil:
		// publish call result only if present
		return ctx.Jump(s.stepAwaitSMCallSummary)
	default:
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

func (s *SMExecute) getToken() rms.CallDelegationToken {
	if s.authenticationService != nil && !s.authenticationService.HasToSendToken(s.delegationTokenSpec) {
		return rms.CallDelegationToken{}
	}
	return s.delegationTokenSpec
}

func (s *SMExecute) sendResult(ctx smachine.ExecutionContext, message rmsreg.GoGoSerializable) {
	target := s.Meta.Sender.GetValue()

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

func (s *SMExecute) deduplicate(state *object.SharedState) (DeduplicationAction, *rms.VCallResult, error) {
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

type RegisterVariant int

const (
	RegisterLifeLine RegisterVariant = iota
	RegisterOutgoingRequest
	RegisterOutgoingResult
	RegisterIncomingResult
)

func (s *SMExecute) constructSubSMRegister(v RegisterVariant) lmn.SubSMRegister {
	subroutineSM := lmn.SubSMRegister{
		SafeResponseCounter: s.lmnSafeResponseCounterLink,
		Interference:        s.methodIsolation.Interference,
		Object:              s.execution.Object,
		LastLifelineRef:     s.lmnLastLifelineRef,
		LastFilamentRef:     s.lmnLastFilamentRef,
	}

	switch v {
	case RegisterLifeLine:
		if s.incomingRegistered {
			panic(throw.IllegalState())
		}
		subroutineSM.Incoming = s.Payload
		return subroutineSM

	case RegisterOutgoingRequest, RegisterIncomingResult:
		if !s.incomingRegistered {
			subroutineSM.Incoming = s.Payload

			s.incomingRegistered = true
		}

	case RegisterOutgoingResult:
		if !s.incomingRegistered {
			panic(throw.IllegalState())
		}

	default:
		panic(throw.IllegalValue())
	}

	return subroutineSM
}
