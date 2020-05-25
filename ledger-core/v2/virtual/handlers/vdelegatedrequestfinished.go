// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
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
	objectReadyToWork smachine.SyncLink

	// dependencies
	objectCatalog object.Catalog
}

type stateIsNotReadyErrorMsg struct {
	*log.Msg  `txt:"State is not ready"`
	Reference string
}

type unExpectedVDelegateRequestFinished struct {
	*log.Msg  `txt:"Unexpected VDelegateRequestFinished"`
	Reference string
	ordered   bool
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

	action := func(state *object.SharedState) {
		s.objectReadyToWork = state.ReadyToWork
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.Impossible())
	case smachine.Passed:
	default:
		panic(throw.NotImplemented())
	}

	return ctx.Jump(s.awaitObjectReady)
}

func (s *SMVDelegatedRequestFinished) awaitObjectReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if ctx.AcquireForThisStep(s.objectReadyToWork).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepProcess)
}

func (s *SMVDelegatedRequestFinished) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if !state.IsReady() {
			ctx.Log().Trace(stateIsNotReadyErrorMsg{
				Reference: s.Payload.Callee.String(),
			})
			return false
		}

		s.updateSharedState(ctx, state)

		return false
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

func (s *SMVDelegatedRequestFinished) updateSharedState(
	ctx smachine.ExecutionContext,
	state *object.SharedState,
) {
	objectRef := s.Payload.Callee

	// Update object state.
	if s.hasLatestState() {
		state.SetDescriptor(s.latestState())
	}

	switch s.Payload.CallFlags.GetInterference() {
	case contract.CallIntolerable:
		if state.ActiveImmutablePendingCount > 0 {
			state.ActiveImmutablePendingCount--
		} else {
			ctx.Log().Warn(unExpectedVDelegateRequestFinished{
				Reference: objectRef.String(),
				ordered:   false,
			})
		}
	case contract.CallTolerable:
		if state.ActiveMutablePendingCount > 0 {
			state.ActiveMutablePendingCount--

			if state.ActiveMutablePendingCount == 0 {
				// If we do not have pending ordered, release sync object.
				if !ctx.CallBargeIn(state.AwaitPendingOrdered) {
					ctx.Log().Warn("AwaitPendingOrdered BargeIn receive false")
				}
			}
		} else {
			ctx.Log().Warn(unExpectedVDelegateRequestFinished{
				Reference: objectRef.String(),
				ordered:   true,
			})
		}
	}
}

func (s *SMVDelegatedRequestFinished) hasLatestState() bool {
	return s.Payload.LatestState != nil
}

func (s *SMVDelegatedRequestFinished) latestState() descriptor.Object {
	state := s.Payload.LatestState
	if state == nil {
		panic(throw.IllegalState())
	}

	return descriptor.NewObject(
		s.Payload.Callee,
		state.Reference,
		state.Prototype,
		state.State,
		state.Parent,
	)
}
