// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package execute

import (
	"context"
	"errors"

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
	"github.com/insolar/assured-ledger/ledger-core/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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
	executionNewState   *executionupdate.ContractExecutionStateUpdate
	outgoingResult      []byte
	deactivate          bool
	run                 runner.RunState
	newObjectDescriptor descriptor.Object

	methodIsolation contract.MethodIsolation

	// dependencies
	runner        runner.ServiceAdapter
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot

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

	if s.isConstructor {
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
		// it could be Empty or even has state, we ll deal with special cases in deduplication step
		if objectState != object.Empty && objectState != object.HasState {
			panic(throw.IllegalState())
		}
	} else if objectState != object.HasState {
		panic(throw.IllegalState())
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

	return ctx.Jump(s.stepTakeLock)
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
		res.Interference = callIsolation.Interference
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

func (s *SMExecute) stepTakeLock(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var executionSemaphore smachine.SyncLink

	if s.execution.Isolation.Interference == contract.CallIntolerable {
		executionSemaphore = s.semaphoreUnordered
	} else {
		executionSemaphore = s.semaphoreOrdered
	}

	if ctx.Acquire(executionSemaphore).IsNotPassed() {
		if s.isConstructor {
			panic(throw.NotImplemented())
		}

		// wait for semaphore to be released
		return ctx.Sleep().ThenRepeat()
	}

	if s.Payload.CallRequestFlags.GetRepeatedCall() == payload.RepeatedCall {
		return ctx.Jump(s.stepDeduplicate)
	}

	return ctx.Jump(s.stepStartRequestProcessing)
}

func (s *SMExecute) stepDeduplicate(ctx smachine.ExecutionContext) smachine.StateUpdate {
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
		ctx.Log().Warn("duplicate found as pending request")
		return ctx.Stop()
	}

	if ctx.AcquireForThisStep(pendingListFilled).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepStartRequestProcessing)
}

func (s *SMExecute) stepStartRequestProcessing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		isDuplicate      bool
		objectDescriptor descriptor.Object
	)
	action := func(state *object.SharedState) {
		requestList := state.KnownRequests.GetList(s.execution.Isolation.Interference)
		if requestList.Exist(s.execution.Outgoing) {
			// found duplicate request, todo: deduplication algorithm
			isDuplicate = true
			return
		}

		requestList.Add(s.execution.Outgoing)
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

	if isDuplicate {
		ctx.Log().Warn("duplicate found as known request")
		return ctx.Stop()
	}

	ctx.SetDefaultMigration(s.migrateDuringExecution)
	s.execution.ObjectDescriptor = objectDescriptor

	return ctx.Jump(s.stepExecuteStart)
}

func (s *SMExecute) migrateDuringExecution(ctx smachine.MigrationContext) smachine.StateUpdate {
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

	subroutineSM := &SMDelegatedTokenRequest{Meta: s.Meta, RequestPayload: requestPayload}
	return ctx.CallSubroutine(subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		if subroutineSM.response == nil {
			panic(throw.IllegalState())
		}
		s.delegationTokenSpec = subroutineSM.response.DelegationSpec
		return ctx.Jump(s.stepAfterTokenGet.Transition)
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
	case executionupdate.Done:
		// send VCallResult here
		return ctx.Jump(s.stepSaveNewObject)
	case executionupdate.Error:
		panic(throw.W(newState.Error, "failed to execute request", nil))
	case executionupdate.Aborted:
		return ctx.Jump(s.stepExecuteAborted)
	case executionupdate.OutgoingCall:
		return ctx.Jump(s.stepExecuteOutgoing)
	default:
		panic(throw.IllegalValue())
	}
}

func (s *SMExecute) stepExecuteAborted(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Error(throw.NotImplemented())
}

func (s *SMExecute) stepExecuteOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	pulseNumber := s.pulseSlot.PulseData().PulseNumber

	switch outgoing := s.executionNewState.Outgoing.(type) {
	case executionevent.GetCode:
		panic(throw.Unsupported())
	case executionevent.Deactivate:
		s.deactivate = true
	case executionevent.CallConstructor:
		s.outgoing = outgoing.ConstructVCallRequest(s.execution)
		s.outgoing.CallOutgoing = gen.UniqueIDWithPulse(pulseNumber)
		s.outgoingObject = reference.NewSelf(s.outgoing.CallOutgoing)
	case executionevent.CallMethod:
		s.outgoing = outgoing.ConstructVCallRequest(s.execution)
		s.outgoing.CallOutgoing = gen.UniqueIDWithPulse(pulseNumber)
		s.outgoingObject = s.outgoing.Callee
	default:
		panic(throw.IllegalValue())
	}

	if s.outgoing != nil {
		return ctx.Jump(s.stepSendOutgoing)
	}

	return ctx.Jump(s.stepExecuteContinue)
}

func (s *SMExecute) stepSendOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !s.outgoingWasSent {
		bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
			res, ok := param.(*payload.VCallResult)
			if !ok || res == nil {
				panic(throw.IllegalValue())
			}
			s.outgoingResult = res.ReturnArguments

			return func(ctx smachine.BargeInContext) smachine.StateUpdate {
				return ctx.WakeUp()
			}
		})

		outgoingRef := reference.NewRecordOf(s.outgoing.Caller, s.outgoing.CallOutgoing)

		if !ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bargeInCallback) {
			return ctx.Error(errors.New("failed to publish bargeInCallback"))
		}
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, s.outgoing, node.DynamicRoleVirtualExecutor, s.outgoingObject, s.pulseSlot.PulseData().PulseNumber)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	s.outgoingWasSent = true
	// we'll wait for barge-in WakeUp here, not adapter
	return ctx.Sleep().ThenJumpExt(
		smachine.SlotStep{
			Transition: s.stepExecuteContinue,
			Migration:  s.migrateDuringSendOutgoing,
		})
}

func (s *SMExecute) migrateDuringSendOutgoing(ctx smachine.MigrationContext) smachine.StateUpdate {
	s.migrationHappened = true

	if s.outgoingWasSent {
		s.outgoing.CallRequestFlags = payload.BuildCallRequestFlags(payload.SendResultDefault, payload.RepeatedCall)
	}

	return ctx.Jump(s.stepSendOutgoing)
}

func (s *SMExecute) stepExecuteContinue(ctx smachine.ExecutionContext) smachine.StateUpdate {
	outgoingResult := s.outgoingResult

	// unset all outgoing fields in case we have new outgoing request
	s.outgoingWasSent = false
	s.outgoingObject = reference.Global{}
	s.outgoing = nil
	s.outgoingResult = []byte{}

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
		class, err := s.execution.ObjectDescriptor.Class()
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
		CallType:     s.Payload.CallType,
		CallFlags:    s.Payload.CallFlags,
		Callee:       s.execution.Object,
		ResultFlags:  nil,
		CallOutgoing: s.Payload.CallOutgoing,
		CallIncoming: reference.Local{},
		LatestState:  lastState,
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
	}
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
		state.FinishRequest(s.execution.Isolation, s.execution.Outgoing)
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
