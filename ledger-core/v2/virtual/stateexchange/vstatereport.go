// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package stateexchange

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
)

type SMVStateReport struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VStateReport
}

var dSMVStateReportInstance smachine.StateMachineDeclaration = &dSMVStateReport{}

type dSMVStateReport struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVStateReport) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (*dSMVStateReport) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVStateReport)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVStateReport) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVStateReportInstance
}

func (s *SMVStateReport) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVStateReport) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	catalog := object.Catalog{}

	incomingObjectState := s.Payload.ProvidedContent.LatestDirtyCode

	objectRef := incomingObjectState.Reference
	smObject := catalog.GetOrCreate(ctx, objectRef)

	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if state.Descriptor() != nil {
			state.SetDescriptor(&incomingObjectState.Prototype, incomingObjectState.State)
		} else {
			inslogger.FromContext(ctx.GetContext()).Infom(struct{ Msg string }{Msg: "State already exists"})
		}
		return true
	}

	switch smObject.PrepareAccess(setStateFunc).TryUse(ctx).GetDecision() {
	case smachine.Passed:
	case smachine.NotPassed:
		return ctx.WaitShared(smObject.SharedDataLink).ThenRepeat()
	default:
		panic(throw.NotImplemented())
	}

	return ctx.Stop()
}
