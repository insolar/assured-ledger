// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"github.com/google/uuid"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	runnerAdapter "github.com/insolar/assured-ledger/ledger-core/v2/runner/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
)

/* -------- Utilities ------------- */

type SMExecute struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCallRequest

	// internal data
	isConstructor      bool
	isOrdered          bool
	semaphoreOrdered   smachine.SyncLink
	semaphoreUnordered smachine.SyncLink
	execution          execution.Context
	objectSharedState  object.SharedStateAccessor

	// execution step
	executionNewState *executionupdate.ContractExecutionStateUpdate
	executionID       uuid.UUID
	executionError    error

	// dependencies
	runner        *runnerAdapter.Runner
	messageSender *messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
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
}

func (*dSMExecute) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMExecute)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMExecute) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMExecuteInstance
}

func (s *SMExecute) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepWaitObjectReady)
}

func (s *SMExecute) stepWaitObjectReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		objectCatalog   = object.Catalog{}
		callType        = s.Payload.CallType
		objectID        = s.Payload.Callee.GetLocal()
		objectReference = reference.NewGlobalSelf(*objectID)

		isOrdered         bool
		isConstructor     bool
		objectSharedState object.SharedStateAccessor
	)

	switch callType {
	case payload.CTConstructor:
		objectSharedState = objectCatalog.GetOrCreate(ctx, objectReference)
		isConstructor = true
		isOrdered = true

	case payload.CTInboundAPICall, payload.CTOutboundAPICall, payload.CTMethod, payload.CTNotifyCall:
		fallthrough
	case payload.CTSAGACall, payload.CTParallelCall, payload.CTScheduleCall:
		objectSharedState = objectCatalog.Get(ctx, objectReference)

	default:
		panic(throw.IllegalValue())
	}

	var (
		semaphoreReadyToWork    smachine.SyncLink
		objectDescriptorIsEmpty bool
		semaphoreOrdered        smachine.SyncLink
		semaphoreUnordered      smachine.SyncLink
	)

	action := func(state *object.SharedState) {
		semaphoreReadyToWork = state.ReadyToWork
		objectDescriptorIsEmpty = state.ObjectLatestDescriptor == nil
		semaphoreOrdered = state.MutableExecute
		semaphoreUnordered = state.ImmutableExecute
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

	s.isOrdered = isOrdered
	s.isConstructor = isConstructor
	s.semaphoreOrdered = semaphoreOrdered
	s.semaphoreUnordered = semaphoreUnordered
	s.objectSharedState = objectSharedState

	// TODO[bigbes]: we're ready to execute here, so lets execute
	return ctx.Jump(s.stepTakeLock)
}

func (s *SMExecute) stepTakeLock(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO[bigbes]: we should classify call here
	var executionSemaphore = s.semaphoreOrdered

	if ctx.Acquire(executionSemaphore).IsNotPassed() {
		if s.isConstructor {
			panic(throw.NotImplemented())
		}

		// wait for semaphore to be released
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepExecute)
}

func (s *SMExecute) prepareExecution(ctx smachine.ExecutionContext) {
	s.execution.Context = ctx.GetContext()
	s.execution.Sequence = 0
	s.execution.Request = s.Payload
}

func (s *SMExecute) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.prepareExecution(ctx)
	execution := s.execution

	goCtx := ctx.GetContext()
	return s.runner.PrepareAsync(ctx, func(svc runner.Service) smachine.AsyncResultFunc {
		newState, id, err := svc.ExecutionStart(goCtx, execution)

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
		ctx.Log().Error("failed to execute: "+s.executionNewState.Error.Error(), nil)
		fallthrough
	case executionupdate.TypeAborted:
		fallthrough
	case executionupdate.TypeOutgoingCall:
		fallthrough
	default:
		panic(throw.W(throw.NotImplemented(), s.executionNewState.Type.String(), nil))
	}
}

func (s *SMExecute) stepSaveNewObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		executionNewState = s.executionNewState.Result

		memory    []byte
		prototype insolar.Reference
	)

	action := func(state *object.SharedState) {
		state.Info.SetDescriptor(&prototype, memory)
	}

	switch s.executionNewState.Result.Type() {
	case insolar.RequestSideEffectNone:
	case insolar.RequestSideEffectActivate:
		_, prototype, memory = executionNewState.Activate()
	case insolar.RequestSideEffectAmend:
		_, prototype, memory = executionNewState.Amend()
	case insolar.RequestSideEffectDeactivate:
		s.execution.Deactivate = true
		action = func(state *object.SharedState) {
			state.Info.Deactivate()
		}
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

	return ctx.Jump(s.stepSendCallResult)
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
		Callee:             s.Payload.Callee,
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
