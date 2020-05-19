// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
)

type SMVDelegatedRequestFinished struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VDelegatedRequestFinished

	objectSharedState object.SharedStateAccessor

	// dependencies
	objectCatalog object.Catalog
}

var dSMVDelegatedRequestFinishedInstance smachine.StateMachineDeclaration = &dSMVDelegatedRequestFinished{}

type dSMVDelegatedRequestFinished struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVDelegatedRequestFinished) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMVDelegatedRequestFinished)

	injector.MustInject(&s.objectCatalog)
}

func (*dSMVDelegatedRequestFinished) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVDelegatedRequestFinished)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVDelegatedRequestFinished) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVDelegatedRequestFinishedInstance
}

func (s *SMVDelegatedRequestFinished) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepGetObject)
}

func (s *SMVDelegatedRequestFinished) stepGetObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.objectSharedState = s.objectCatalog.GetOrCreate(ctx, s.Payload.Callee)

	var semaphoreReadyToWork smachine.SyncLink

	action := func(state *object.SharedState) {
		semaphoreReadyToWork = state.ReadyToWork
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		ctx.Log().Fatal("failed to get object state: already dead")
	case smachine.Passed:
	default:
		panic(throw.NotImplemented())
	}

	if ctx.AcquireForThisStep(semaphoreReadyToWork).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepProcess)
}

func (s *SMVDelegatedRequestFinished) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	objectRef := s.Payload.Callee

	var objectDescriptor *descriptor.Object

	if s.Payload.LatestState != nil {
		state := s.Payload.LatestState
		desc := descriptor.NewObject(
			objectRef,
			state.Reference,
			state.Prototype,
			state.State,
			state.Parent,
		)
		objectDescriptor = &desc
	}

	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if !state.IsReady() {
			ctx.Log().Trace(stateIsNotReadyErrorMsg{
				Reference: objectRef.String(),
			})
			return false
		}

		// Update object state.
		if objectDescriptor != nil {
			state.SetDescriptor(*objectDescriptor)
		}

		switch s.Payload.CallFlags.GetTolerance() {
		case payload.CallIntolerable:
			state.ActiveImmutablePendingCount--
		case payload.CallTolerable:
			if state.ActiveMutablePendingCount > 0 {
				state.ActiveMutablePendingCount--

				if state.ActiveMutablePendingCount == 0 {
					// If we do not have pending ordered, release sync object.
					if !ctx.CallBargeIn(state.AwaitPendingOrdered) {
						ctx.Log().Trace("AwaitPendingOrdered BargeIn receive false")
					}
				}
			}
		}

		return true
	}

	switch s.objectSharedState.PrepareAccess(setStateFunc).TryUse(ctx).GetDecision() {
	case smachine.Passed:
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	default:
		panic(throw.Impossible())
	}

	return ctx.Stop()
}
