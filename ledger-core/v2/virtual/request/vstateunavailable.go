// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package request

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
)

type SMVStateUnavailable struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VStateUnavailable
}

var dSMVStateUnavailableInstance smachine.StateMachineDeclaration = &dSMVStateUnavailable{}

type dSMVStateUnavailable struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVStateUnavailable) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (*dSMVStateUnavailable) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVStateUnavailable)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVStateUnavailable) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVStateUnavailableInstance
}

func (s *SMVStateUnavailable) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

type noObjectErrorMsg struct {
	*log.Msg  `txt:"There is no such object"`
	Reference string
}

type stateAlreadyExistsErrorMsg struct {
	*log.Msg  `txt:"State already exists"`
	Reference string
	GotState  string
}

func (s *SMVStateUnavailable) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	catalog := object.Catalog{}
	objectRef := s.Payload.Lifeline
	sharedObjectState, ok := catalog.TryGet(ctx, objectRef)
	if !ok {
		ctx.Log().Error(noObjectErrorMsg{Reference: objectRef.String()}, nil)
		return ctx.Stop()
	}

	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if state.IsReady() {
			ctx.Log().Trace(stateAlreadyExistsErrorMsg{
				Reference: objectRef.String(),
				GotState:  s.Payload.Reason.String(),
			})
			return false
		}

		switch s.Payload.Reason {
		case payload.Missing:
			state.SetState(object.Missing)
		case payload.Inactive:
			state.SetState(object.Inactive)
		default:
			panic(throw.IllegalState())
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
