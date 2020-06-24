// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package execute

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
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
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

/* -------- Utilities ------------- */

type SMExecute struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCallRequest

	// internal data
	isConstructor      bool
	semaphoreOrdered   smachine.SyncLink
	semaphoreUnordered smachine.SyncLink
	execution          execution.Context
	objectSharedState  object.SharedStateAccessor

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

	outgoing        *payload.VCallRequest
	outgoingObject  reference.Global
	outgoingWasSent bool

	migrationHappened bool
	objectCatalog     object.Catalog

	delegationTokenSpec payload.CallDelegationToken
	stepAfterTokenGet   smachine.SlotStep
}

/* -------- Declaration ------------- */

var dSMExecuteInstance smachine.StateMachineDeclaration = &dSMExecute{}

type dSMExecute struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMExecute) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMExecute)

	injector.MustInject(&s.runner)
	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.objectCatalog)
	injector.MustInject(&s.authenticationService)
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

	if s.Payload.CallType == payload.CTConstructor {
		s.isConstructor = true
		s.execution.Object = reference.NewSelf(s.Payload.CallOutgoing)
	} else {
		s.execution.Object = s.Payload.Callee
	}

	s.execution.Incoming = reference.NewRecordOf(s.Payload.Caller, s.Payload.CallOutgoing)
	s.execution.Outgoing = reference.NewRecordOf(s.Payload.Callee, s.Payload.CallOutgoing)

	s.execution.Isolation = contract.MethodIsolation{
		Interference: s.Payload.CallFlags.GetInterference(),
		State:        s.Payload.CallFlags.GetState(),
	}
}

func (s *SMExecute) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (s *SMExecute) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	s.prepareExecution(ctx.GetContext())

	ctx.SetDefaultMigration(s.migrationDefault)

	return ctx.Jump(s.stepCheckRequest)
}

func (s *SMExecute) stepCheckRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch s.Payload.CallType {
	case payload.CTConstructor:
	case payload.CTMethod:

	case payload.CTInboundAPICall, payload.CTOutboundAPICall, payload.CTNotifyCall,
		payload.CTSAGACall, payload.CTParallelCall, payload.CTScheduleCall:
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

		switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
		case smachine.NotPassed:
			return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
		case smachine.Impossible:
			panic(throw.NotImplemented())
		case smachine.Passed:
			// go further
		default:
			panic(throw.Impossible())
		}
	}

	return ctx.Jump(s.stepWaitObjectReady)
}

func (s *SMExecute) outgoingFromSlotPulse() bool {
	outgoingPulse := s.Payload.CallOutgoing.GetPulseNumber()
	slotPulse := s.pulseSlot.PulseData().GetPulseNumber()
	return outgoingPulse == slotPulse
}

func (s *SMExecute) intolerableCall() bool {
	return s.execution.Isolation.Interference == contract.CallIntolerable
}

func (s *SMExecute) stepWaitObjectReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		isConstructor     = s.isConstructor
		objectSharedState = s.objectSharedState
	)

	var (
		semaphoreReadyToWork smachine.SyncLink
		objectDescriptor     descriptor.Object
		semaphoreOrdered     smachine.SyncLink
		semaphoreUnordered   smachine.SyncLink

		objectState object.State
	)

	action := func(state *object.SharedState) {
		semaphoreReadyToWork = state.ReadyToWork

		semaphoreOrdered = state.OrderedExecute
		semaphoreUnordered = state.UnorderedExecute

		objectDescriptor = state.Descriptor()

		objectState = state.GetState()
	}

	switch objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	if ctx.AcquireForThisStep(semaphoreReadyToWork).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	if isConstructor {
		switch objectState {
		case object.Unknown:
			panic(throw.Impossible())
		case object.Inactive:
			// attempt to create object that is deactivated :(
			panic(throw.NotImplemented())
		}
	} else if objectState != object.HasState {
		panic(throw.E("no state on object after readyToWork", struct {
			ObjectReference string
			State           object.State
		}{
			ObjectReference: s.execution.Object.String(),
			State:           objectState,
		}))
	}

	s.semaphoreOrdered = semaphoreOrdered
	s.semaphoreUnordered = semaphoreUnordered

	s.execution.ObjectDescriptor = objectDescriptor

	if isConstructor {
		// default isolation for constructors
		s.methodIsolation = contract.ConstructorIsolation()
	}
	return ctx.Jump(s.stepIsolationNegotiation)
}

func (s *SMExecute) stepIsolationNegotiation(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.methodIsolation.IsZero() {
		return s.runner.PrepareExecutionClassify(ctx, s.execution, func(isolation contract.MethodIsolation, err error) {
			if err != nil {
				panic(throw.W(err, "failed to classify method"))
			}
			s.methodIsolation = isolation
		}).DelayedStart().Sleep().ThenRepeat()
	}

	negotiatedIsolation, err := negotiateIsolation(s.methodIsolation, s.execution.Isolation)
	if err != nil {
		return ctx.Error(throw.W(err, "failed to negotiate", struct {
			methodIsolation contract.MethodIsolation
			callIsolation   contract.MethodIsolation
		}{
			methodIsolation: s.methodIsolation,
			callIsolation:   s.execution.Isolation,
		}))
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
	case methodIsolation.Interference == contract.CallIntolerable && callIsolation.Interference == contract.CallTolerable:
		res.Interference = methodIsolation.Interference
	default:
		return contract.MethodIsolation{}, throw.IllegalValue()
	}
	switch {
	case methodIsolation.State == callIsolation.State:
		// ok case
	case methodIsolation.State == contract.CallValidated && callIsolation.State == contract.CallDirty:
		res.State = callIsolation.State
	default:
		return contract.MethodIsolation{}, throw.IllegalValue()
	}

	return res, nil
}

func (s *SMExecute) stepDeduplicate(ctx smachine.ExecutionContext) smachine.StateUpdate {
	duplicate := false

	action := func(state *object.SharedState) {
		req := s.execution.Outgoing

		workingList := state.KnownRequests.GetList(s.execution.Isolation.Interference)

		requestState := workingList.GetState(req)
		switch requestState {
		// processing started but not yet processing
		case object.RequestStarted:
			duplicate = true
			return
		case object.RequestProcessing:
			duplicate = true
			// TODO: we may have result already, should resend
			return
		}

		workingList.Add(req)
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	if duplicate {
		ctx.Log().Warn("duplicate found")
		return ctx.Stop()
	}

	if !s.outgoingFromSlotPulse() {
		return ctx.Jump(s.stepDeduplicateUsingPendingsTable)
	}
	return ctx.Jump(s.stepTakeLock)
}

func (s *SMExecute) stepDeduplicateUsingPendingsTable(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		isDuplicate       bool
		pendingListFilled smachine.SyncLink
	)
	action := func(state *object.SharedState) {
		if s.execution.Isolation.Interference == contract.CallIntolerable {
			pendingListFilled = state.UnorderedPendingListFilled
		} else {
			pendingListFilled = state.OrderedPendingListFilled
		}

		pendingList := state.PendingTable.GetList(s.execution.Isolation.Interference)

		if pendingList.Exist(s.execution.Outgoing) {
			isDuplicate = true
			// TODO: we may have result already, should resend
			return
		}
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
	default:
		panic(throw.NotImplemented())
	}

	if isDuplicate {
		// TODO: do we have result? if we have result then we should resend result
		ctx.Log().Warn("duplicate found as pending request")
		return ctx.Stop()
	}

	if ctx.AcquireForThisStep(pendingListFilled).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepTakeLock)
}

func (s *SMExecute) stepTakeLock(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var executionSemaphore smachine.SyncLink

	if s.execution.Isolation.Interference == contract.CallIntolerable {
		executionSemaphore = s.semaphoreUnordered
	} else {
		executionSemaphore = s.semaphoreOrdered
	}

	if ctx.Acquire(executionSemaphore).IsNotPassed() {
		// wait for semaphore to be released
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepStartRequestProcessing)
}

func (s *SMExecute) stepStartRequestProcessing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		objectDescriptor descriptor.Object
	)
	action := func(state *object.SharedState) {
		reqRef := s.execution.Outgoing

		reqList := state.KnownRequests.GetList(s.execution.Isolation.Interference)
		if !reqList.SetActive(reqRef) {
			// if we come here then request should be in RequestStarted
			// if it is not it is either somehow lost or it is already processing
			panic(throw.Impossible())
		}

		state.IncrementPotentialPendingCounter(s.execution.Isolation)
		objectDescriptor = state.Descriptor()
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
	default:
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
		if s.outgoingWasSent {
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
	}).DelayedStart().ThenJump(s.stepWaitExecutionResult)
}

func (s *SMExecute) stepWaitExecutionResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
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
	pulseNumber := s.pulseSlot.PulseData().PulseNumber

	switch outgoing := s.executionNewState.Outgoing.(type) {
	case execution.Deactivate:
		s.deactivate = true
	case execution.CallConstructor:
		if s.intolerableCall() {
			err := throw.E("interference violation: constructor call from unordered call")
			ctx.Log().Warn(err)
			s.prepareOutgoingError(err)
			return ctx.Jump(s.stepExecuteContinue)
		}

		s.outgoing = outgoing.ConstructVCallRequest(s.execution)
		s.outgoing.CallOutgoing = gen.UniqueLocalRefWithPulse(pulseNumber)
		s.execution.Sequence++
		s.outgoing.CallSequence = s.execution.Sequence
		s.outgoingObject = reference.NewSelf(s.outgoing.CallOutgoing)
		s.outgoing.DelegationSpec = s.getToken()
	case execution.CallMethod:
		if s.intolerableCall() && outgoing.Interference() == contract.CallTolerable {
			err := throw.E("interference violation: ordered call from unordered call")
			ctx.Log().Warn(err)
			s.prepareOutgoingError(err)
			return ctx.Jump(s.stepExecuteContinue)
		}

		s.outgoing = outgoing.ConstructVCallRequest(s.execution)
		s.outgoing.CallOutgoing = gen.UniqueLocalRefWithPulse(pulseNumber)
		s.execution.Sequence++
		s.outgoing.CallSequence = s.execution.Sequence
		s.outgoingObject = s.outgoing.Callee
		s.outgoing.DelegationSpec = s.getToken()
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
	return s.runner.PrepareExecutionAbort(ctx, s.run, func() {}).DelayedStart().ThenJump(s.stepSendCallResult)
}

func (s *SMExecute) stepSendOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !s.outgoingWasSent {
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

		outgoingRef := reference.NewRecordOf(s.outgoing.Caller, s.outgoing.CallOutgoing)

		if !ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bargeInCallback) {
			return ctx.Error(throw.E("failed to publish bargeInCallback"))
		}
	} else {
		s.outgoing.CallRequestFlags = payload.BuildCallRequestFlags(payload.SendResultDefault, payload.RepeatedCall)
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, s.outgoing, node.DynamicRoleVirtualExecutor, s.outgoingObject, s.pulseSlot.CurrentPulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	s.outgoingWasSent = true
	// we'll wait for barge-in WakeUp here, not adapter
	return ctx.Sleep().ThenJump(s.stepExecuteContinue)
}

func (s *SMExecute) stepExecuteContinue(ctx smachine.ExecutionContext) smachine.StateUpdate {
	outgoingResult := s.outgoingResult

	// unset all outgoing fields in case we have new outgoing request
	s.outgoingWasSent = false
	s.outgoingObject = reference.Global{}
	s.outgoing = nil
	s.outgoingResult = []byte{}
	ctx.SetDefaultMigration(s.migrateDuringExecution)

	s.executionNewState = nil

	return s.runner.PrepareExecutionContinue(ctx, s.run, outgoingResult, func() {
		if s.run == nil {
			panic(throw.IllegalState())
		}

		s.executionNewState = s.run.GetResult()
		if s.executionNewState == nil {
			panic(throw.IllegalState())
		}
	}).DelayedStart().ThenJump(s.stepWaitExecutionResult)
}

func (s *SMExecute) stepSaveNewObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		executionNewState = s.executionNewState.Result

		memory []byte
		class  reference.Global
	)

	if s.deactivate {
		oldRequestResult := s.executionNewState.Result

		// we should overwrite old side effect with new one - deactivation of object
		s.executionNewState.Result = requestresult.New(oldRequestResult.Result(), oldRequestResult.ObjectReference())
		s.executionNewState.Result.SetDeactivate(s.execution.ObjectDescriptor)
	}

	switch s.executionNewState.Result.Type() {
	case requestresult.SideEffectNone:
	case requestresult.SideEffectActivate:
		_, class, memory = executionNewState.Activate()
		s.newObjectDescriptor = s.makeNewDescriptor(class, memory)
	case requestresult.SideEffectAmend:
		_, class, memory = executionNewState.Amend()
		s.newObjectDescriptor = s.makeNewDescriptor(class, memory)
	case requestresult.SideEffectDeactivate:
		panic(throw.NotImplemented())
	default:
		panic(throw.IllegalValue())
	}

	if s.migrationHappened || s.newObjectDescriptor == nil {
		return ctx.Jump(s.stepSendCallResult)
	}

	action := func(state *object.SharedState) {
		state.Info.SetDescriptor(s.newObjectDescriptor)

		if state.GetState() == object.Empty {
			state.SetState(object.HasState)
		}
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	return ctx.Jump(s.stepSendCallResult)
}

func (s *SMExecute) stepSendDelegatedRequestFinished(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var lastState *payload.ObjectState = nil

	if s.newObjectDescriptor != nil {
		class, err := s.newObjectDescriptor.Class()
		if err != nil {
			panic(throw.W(err, "failed to get class from descriptor", nil))
		}

		lastState = &payload.ObjectState{
			Reference: s.executionNewState.Result.ObjectStateID,
			State:     s.executionNewState.Result.Memory,
			Parent:    s.executionNewState.Result.ParentReference,
			Class:     class,
		}
	}

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
		err := svc.SendRole(goCtx, &msg, node.DynamicRoleVirtualExecutor, s.execution.Object, s.pulseSlot.CurrentPulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Stop()
}

func (s *SMExecute) makeNewDescriptor(class reference.Global, memory []byte) descriptor.Object {
	parentReference := reference.Global{}
	var prevStateIDBytes []byte
	objDescriptor := s.execution.ObjectDescriptor
	if objDescriptor != nil {
		parentReference = objDescriptor.HeadRef()
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
		parentReference,
	)
}

func (s *SMExecute) stepSendCallResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		executionNewState = s.executionNewState.Result
		executionResult   = executionNewState.Result()
	)

	msg := payload.VCallResult{
		CallType:           s.Payload.CallType,
		CallFlags:          s.Payload.CallFlags,
		CallAsOf:           s.Payload.CallAsOf,
		Caller:             s.Payload.Caller,
		Callee:             s.execution.Object,
		ResultFlags:        nil,
		CallOutgoing:       s.Payload.CallOutgoing,
		CallIncoming:       reference.Local{},
		PayloadHash:        nil,
		CallIncomingResult: reference.Local{},
		EntryHeadHash:      nil,
		ReturnArguments:    executionResult,
		DelegationSpec:     s.getToken(),
	}

	// save result for future pass to SMObject
	s.execution.Result = &msg

	target := s.Meta.Sender

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendTarget(goCtx, &msg, target)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Jump(s.stepFinishRequest)
}

func (s *SMExecute) stepFinishRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.migrationHappened {
		return ctx.Jump(s.stepSendDelegatedRequestFinished)
	}

	action := func(state *object.SharedState) {
		state.FinishRequest(s.execution.Isolation, s.execution.Outgoing, s.execution.Result)
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
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
