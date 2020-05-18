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

	sharedObjectState := s.objectCatalog.Get(ctx, objectRef)
	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if !state.IsReady() {
			ctx.Log().Trace(stateAlreadyExistsErrorMsg{
				Reference: objectRef.String(),
			})
			return false
		}

		state.ActiveMutablePendingCount--

		if state.ActiveMutablePendingCount < 0 {
			//TODO how to handle?
		}

		if state.ActiveMutablePendingCount == 0 {
			// If we do not have pending ordered, release sync object.
			if !ctx.CallBargeIn(state.AwaitPendingOrdered) {
				//TODO how to handle?
			}
		}

		// Update object state.
		if objectDescriptor != nil {
			state.SetDescriptor(*objectDescriptor)
		}

		return true
	}

	switch sharedObjectState.PrepareAccess(setStateFunc).TryUse(ctx).GetDecision() {
	case smachine.Passed:
	case smachine.NotPassed:
		return ctx.WaitShared(sharedObjectState.SharedDataLink).ThenRepeat()
	default:
		panic(throw.NotImplemented())
	}

	return ctx.Stop()
}
