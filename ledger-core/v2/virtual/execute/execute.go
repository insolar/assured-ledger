// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	runnerAdapter "github.com/insolar/assured-ledger/ledger-core/v2/runner/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
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
	executionNewState *executionupdate.ContractExecutionStateUpdate
	executionID       uuid.UUID
	executionError    error
	outgoingResult    []byte
	deactivate        bool

	// dependencies
	runner        *runnerAdapter.Runner
	messageSender *messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot

	migrationHappened bool
	objectCatalog     object.Catalog
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

func (s *SMExecute) prepareExecution(ctx smachine.InitializationContext) {
	s.execution.Context = ctx.GetContext()
	s.execution.Sequence = 0
	s.execution.Request = s.Payload
	s.execution.Pulse = s.pulseSlot.PulseData()

	s.execution.Object = s.Payload.Callee
	s.execution.Incoming = reference.NewRecordOf(s.Payload.Caller, s.Payload.CallOutgoing)
	s.execution.Outgoing = reference.NewRecordOf(s.Payload.Callee, s.Payload.CallOutgoing)
}

func (s *SMExecute) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	s.prepareExecution(ctx)

	return ctx.Jump(s.stepUpdatePendingCounters)
}

func (s *SMExecute) stepUpdatePendingCounters(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		callType = s.Payload.CallType

		isConstructor     bool
		objectSharedState object.SharedStateAccessor
	)

	reason := object.InitReasonCTMethod

	switch callType {
	case payload.CTConstructor:
		isConstructor = true
		s.execution.Object = reference.NewSelf(s.Payload.CallOutgoing)
		reason = object.InitReasonCTConstructor

	case payload.CTMethod:
		isConstructor = false

	case payload.CTInboundAPICall, payload.CTOutboundAPICall, payload.CTNotifyCall:
		fallthrough
	case payload.CTSAGACall, payload.CTParallelCall, payload.CTScheduleCall:
		panic(throw.NotImplemented())
	default:
		panic(throw.IllegalValue())
	}

	if s.Payload.CallFlags.GetTolerance() == payload.CallIntolerable {
		s.execution.Unordered = true
	}

	objectSharedState = s.objectCatalog.GetOrCreate(ctx, s.execution.Object, reason)

	switch objectSharedState.Prepare(func(state *object.SharedState) {
		state.IncrementPotentialPendingCounter(!s.execution.Unordered)
	}).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		ctx.Log().Fatal("failed to get object state: already dead")
	case smachine.Passed:
	default:
		panic(throw.NotImplemented())
	}

	s.objectSharedState = objectSharedState
	s.isConstructor = isConstructor

	ctx.SetDefaultMigration(s.migrateDuringExecution)
	return ctx.Jump(s.stepWaitObjectReady)
}

func (s *SMExecute) stepWaitObjectReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		isConstructor     = s.isConstructor
		objectSharedState = s.objectSharedState
	)

	var (
		semaphoreReadyToWork    smachine.SyncLink
		objectDescriptorIsEmpty bool
		semaphoreOrdered        smachine.SyncLink
		semaphoreUnordered      smachine.SyncLink
	)

	action := func(state *object.SharedState) {
		semaphoreReadyToWork = state.ReadyToWork

		semaphoreOrdered = state.MutableExecute
		semaphoreUnordered = state.ImmutableExecute

		objectDescriptor := state.Descriptor()
		objectDescriptorIsEmpty = objectDescriptor == nil
	}

	switch objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		ctx.Log().Fatal("failed to get object state: already dead")
	case smachine.Passed:
		// go further
	default:
		// TODO[bigbes]: handle object is gone here the right way
		panic(throw.NotImplemented())
	}

	if ctx.AcquireForThisStep(semaphoreReadyToWork).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	if (isConstructor && !objectDescriptorIsEmpty) || (!isConstructor && objectDescriptorIsEmpty) {
		// TODO[bigbes]: handle different errors here correctly
		panic(throw.NotImplemented())
	}

	s.semaphoreOrdered = semaphoreOrdered
	s.semaphoreUnordered = semaphoreUnordered

	// TODO[bigbes]: we're ready to execute here, so lets execute
	return ctx.Jump(s.stepTakeLock)
}

func (s *SMExecute) stepTakeLock(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var executionSemaphore smachine.SyncLink

	if s.execution.Unordered {
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

	return ctx.Jump(s.stepGetObjectDescriptor)
}

func (s *SMExecute) stepGetObjectDescriptor(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var objectDescriptor descriptor.Object

	action := func(state *object.SharedState) {
		objectDescriptor = state.Descriptor()
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		ctx.Log().Fatal("failed to get object state: already dead")
	case smachine.Passed:
		// go further
	default:
		// TODO[bigbes]: handle object is gone here the right way
		panic(throw.NotImplemented())
	}

	s.execution.ObjectDescriptor = objectDescriptor

	return ctx.Jump(s.stepExecuteStart)
}

func (s *SMExecute) migrateDuringExecution(ctx smachine.MigrationContext) smachine.StateUpdate {
	s.migrationHappened = true
	return ctx.Stay()
}

func (s *SMExecute) stepExecuteStart(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		executionContext = s.execution
		asyncLogger      = ctx.LogAsync()
		goCtx            = ctx.GetContext()
	)

	return s.runner.PrepareAsync(ctx, func(svc runner.Service) smachine.AsyncResultFunc {
		defer statemachine.LogAsyncTime(asyncLogger, time.Now(), "ExecutionStart")

		newState, id, err := svc.ExecutionStart(goCtx, executionContext)

		return func(ctx smachine.AsyncResultContext) {
			s.executionNewState = newState
			s.executionID = id
			s.executionError = err
		}
	}).DelayedStart().Sleep().ThenJump(s.stepExecuteDecideNextStep)
}

func (s *SMExecute) stepExecuteDecideNextStep(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch s.executionNewState.Type {
	case executionupdate.TypeDone:
		// send VCallResult here
		return ctx.Jump(s.stepSaveNewObject)
	case executionupdate.TypeError:
		panic(throw.W(s.executionNewState.Error, "failed to execute request", nil))
	case executionupdate.TypeAborted:
		return ctx.Jump(s.stepExecuteAborted)
	case executionupdate.TypeOutgoingCall:
		return ctx.Jump(s.stepExecuteOutgoing)
	default:
		panic(throw.IllegalValue())
	}
}

func (s *SMExecute) stepExecuteAborted(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (s *SMExecute) stepExecuteOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		msg    *payload.VCallRequest
		object reference.Global

		goCtx       = ctx.GetContext()
		pulseNumber = s.pulseSlot.PulseData().PulseNumber
	)

	switch outgoing := s.executionNewState.Outgoing.(type) {
	case executionevent.GetCode:
		panic(throw.Unsupported())
	case executionevent.Deactivate:
		s.deactivate = true
	case executionevent.CallConstructor:
		msg = outgoing.ConstructVCallRequest(s.execution)
		msg.CallOutgoing = gen.UniqueIDWithPulse(pulseNumber)
		object = reference.NewSelf(msg.CallOutgoing)
	case executionevent.CallMethod:
		msg = outgoing.ConstructVCallRequest(s.execution)
		msg.CallOutgoing = gen.UniqueIDWithPulse(pulseNumber)
		object = msg.Callee
	default:
		panic(throw.IllegalValue())
	}

	if msg != nil {
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

		outgoingRef := reference.NewRecordOf(msg.Caller, msg.CallOutgoing)

		if !ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bargeInCallback) {
			return ctx.Error(errors.New("failed to publish bargeInCallback"))
		}

		return s.messageSender.PrepareNotify(ctx, func(svc messagesender.Service) {
			err := svc.SendRole(goCtx, msg, insolar.DynamicRoleVirtualExecutor, object, pulseNumber)
			if err != nil {
				panic(err)
			}
		}).DelayedSend().Sleep().ThenJump(s.stepExecuteContinue) // we'll wait for barge-in WakeUp here, not adapter
	}

	return ctx.Jump(s.stepExecuteContinue)
}

func (s *SMExecute) stepExecuteContinue(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		outgoingResult = s.outgoingResult
		asyncLogger    = ctx.LogAsync()
		goCtx          = ctx.GetContext()
		executionID    = s.executionID
	)

	return s.runner.PrepareAsync(ctx, func(svc runner.Service) smachine.AsyncResultFunc {
		defer statemachine.LogAsyncTime(asyncLogger, time.Now(), "ExecutionStart")

		newState, err := svc.ExecutionContinue(goCtx, executionID, outgoingResult)

		return func(ctx smachine.AsyncResultContext) {
			s.executionNewState = newState
			s.executionError = err
		}
	}).DelayedStart().Sleep().ThenJump(s.stepExecuteDecideNextStep)
}

func (s *SMExecute) stepSaveNewObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		executionNewState = s.executionNewState.Result

		memory    []byte
		prototype reference.Global
		action    func(state *object.SharedState)
	)

	if s.deactivate {
		oldRequestResult := s.executionNewState.Result

		// we should overwrite old side effect with new one - deactivation of object
		// TODO[bigbes]: maybe we should move that logic to runner
		s.executionNewState.Result = requestresult.New(oldRequestResult.Result(), oldRequestResult.ObjectReference())
		s.executionNewState.Result.SetDeactivate(s.execution.ObjectDescriptor)
	}

	action = s.updateCounters()

	switch s.executionNewState.Result.Type() {
	case insolar.RequestSideEffectNone:
	case insolar.RequestSideEffectActivate:
		_, prototype, memory = executionNewState.Activate()
		action = s.setNewState(prototype, memory)
	case insolar.RequestSideEffectAmend:
		_, prototype, memory = executionNewState.Amend()
		action = s.setNewState(prototype, memory)
	case insolar.RequestSideEffectDeactivate:
		panic(throw.NotImplemented())
	default:
		panic(throw.IllegalValue())
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		ctx.Log().Fatal("failed to get object state: already dead")
	case smachine.Passed:
		// go further
	default:
		// TODO[bigbes]: handle object is gone here the right way
		panic(throw.NotImplemented())
	}

	if s.migrationHappened {
		return ctx.Jump(s.stepSendDelegatedRequestFinished)
	}

	return ctx.Jump(s.stepSendCallResult)
}

func (s *SMExecute) stepSendDelegatedRequestFinished(ctx smachine.ExecutionContext) smachine.StateUpdate {

	var lastState *payload.ObjectState = nil
	if !s.execution.Unordered {
		lastState = &payload.ObjectState{
			Reference: s.executionNewState.Result.ObjectStateID,
			State:     s.executionNewState.Result.Memory,
		}
	}

	msg := payload.VDelegatedRequestFinished{
		CallType:           s.Payload.CallType,
		CallFlags:          s.Payload.CallFlags,
		Callee:             s.execution.Object,
		ResultFlags:        nil,
		CallOutgoing:       s.Payload.CallOutgoing,
		CallIncoming:       reference.Local{},
		EntryHeadHash:      nil,
		DelegationSpec:     nil,
		DelegatorSignature: nil,
		LatestState:        lastState,
	}

	goCtx := ctx.GetContext()
	s.messageSender.PrepareNotify(ctx, func(svc messagesender.Service) {
		_ = svc.SendRole(goCtx, &msg, insolar.DynamicRoleVirtualExecutor, s.execution.Object, s.pulseSlot.CurrentPulseNumber())
	}).Send()

	return ctx.Jump(s.stepSendCallResult)
}

func (s *SMExecute) updateCounters() func(state *object.SharedState) {
	return func(state *object.SharedState) {
		if !s.migrationHappened {
			state.DecrementPotentialPendingCounter(!s.execution.Unordered)
		}
	}
}

func (s *SMExecute) setNewState(prototype reference.Global, memory []byte) func(state *object.SharedState) {
	return func(state *object.SharedState) {

		parentReference := reference.Global{}
		var prevStateIDBytes []byte
		if state.Descriptor() != nil {
			parentReference = state.Descriptor().Parent()
			prevStateIDBytes = state.Descriptor().StateID().AsBytes()
		}

		objectRefBytes := s.execution.Object.AsBytes()
		stateHash := append(memory, objectRefBytes...)
		stateHash = append(stateHash, prevStateIDBytes...)

		stateID := NewStateID(s.pulseSlot.PulseData().GetPulseNumber(), stateHash)
		state.Info.SetDescriptor(descriptor.NewObject(
			s.execution.Object,
			stateID,
			prototype,
			memory,
			parentReference,
		))

		s.updateCounters()
		state.SetState(object.HasState)
	}
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

	goCtx := ctx.GetContext()
	s.messageSender.PrepareNotify(ctx, func(svc messagesender.Service) {
		_ = svc.SendTarget(goCtx, &msg, target)
	}).Send()

	return ctx.Stop()
}

func NewStateID(pn pulse.Number, data []byte) reference.Local {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	hash := hasher.Hash(data)
	return reference.NewLocal(pn, 0, reference.BytesToLocalHash(hash))
}
