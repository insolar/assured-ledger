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

	objectRef := s.Payload.Callee
	var (
		objectDescriptor descriptor.Object

		deactivated *bool
	)

	if s.Payload.ProvidedContent == nil || s.Payload.ProvidedContent.LatestDirtyState == nil {
		panic(throw.IllegalValue())
	}

	dirtyState := s.Payload.ProvidedContent.LatestDirtyState
	objectDescriptor = descriptor.NewObject(
		objectRef,
		dirtyState.Reference,
		dirtyState.Prototype,
		dirtyState.State,
		dirtyState.Parent,
	)

	deactivated = &s.Payload.ProvidedContent.LatestDirtyState.Deactivated

	sharedObjectState := catalog.GetOrCreate(ctx, objectRef, object.InitReasonVStateReport)
	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if state.IsReady() {
			ctx.Log().Trace(stateAlreadyExistsErrorMsg{
				Reference: objectRef.String(),
			})
			return false
		}

		state.ActiveImmutablePendingCount = uint8(s.Payload.ImmutablePendingCount)
		state.ActiveMutablePendingCount = uint8(s.Payload.MutablePendingCount)

		state.SetDescriptor(objectDescriptor)
		state.SetState(object.HasState)

		if deactivated != nil {
			state.Deactivated = *deactivated
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
