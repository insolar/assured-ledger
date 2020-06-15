// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SMVStateReport struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VStateReport

	// dependencies
	objectCatalog object.Catalog
}

var dSMVStateReportInstance smachine.StateMachineDeclaration = &dSMVStateReport{}

type dSMVStateReport struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVStateReport) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMVStateReport)

	injector.MustInject(&s.objectCatalog)
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

type stateAlreadyExistsErrorMsg struct {
	*log.Msg  `txt:"State already exists"`
	Reference string
	GotState  string
}

func (s *SMVStateReport) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.Payload.Status == payload.Unknown {
		return ctx.Error(throw.IllegalValue())
	}

	if s.Payload.Status >= payload.Empty && s.gotLatestDirty() {
		return ctx.Error(throw.IllegalValue())
	}

	objectRef := s.Payload.Object
	sharedObjectState := s.objectCatalog.GetOrCreate(ctx, objectRef)
	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if state.IsReady() {
			ctx.Log().Trace(stateAlreadyExistsErrorMsg{
				Reference: objectRef.String(),
			})
			return false
		}
		s.updateSharedState(state)
		return true
	}

	switch sharedObjectState.PrepareAccess(setStateFunc).TryUse(ctx).GetDecision() {
	case smachine.Passed:
	case smachine.NotPassed:
		return ctx.WaitShared(sharedObjectState.SharedDataLink).ThenRepeat()
	default:
		panic(throw.Impossible())
	}

	return ctx.Stop()
}

func (s *SMVStateReport) updateSharedState(
	state *object.SharedState,
) {
	objectRef := s.Payload.Object

	var objState object.State
	switch s.Payload.Status {
	case payload.Unknown:
		panic(throw.IllegalValue())
	case payload.Ready:
		if !s.gotLatestDirty() {
			panic(throw.IllegalState())
		}
		objState = object.HasState
	case payload.Empty:
		objState = object.Empty
	case payload.Inactive:
		objState = object.Inactive
	case payload.Missing:
		objState = object.Missing
	default:
		panic(throw.IllegalValue())
	}

	if s.Payload.Status >= payload.Empty && s.gotLatestDirty() {
		panic(throw.IllegalState())
	}

	state.ActiveUnorderedPendingCount = uint8(s.Payload.UnorderedPendingCount)
	state.ActiveOrderedPendingCount = uint8(s.Payload.OrderedPendingCount)

	state.OrderedPendingEarliestPulse = s.Payload.OrderedPendingEarliestPulse
	state.UnorderedPendingEarliestPulse = s.Payload.UnorderedPendingEarliestPulse

	if s.gotLatestDirty() {
		dirty := *s.Payload.ProvidedContent.LatestDirtyState
		desc := buildObjectDescriptor(objectRef, dirty)
		state.SetDescriptor(desc)
		state.Deactivated = dirty.Deactivated
	}

	state.SetState(objState)

}

func (s *SMVStateReport) gotLatestDirty() bool {
	content := s.Payload.ProvidedContent
	return content != nil && content.LatestDirtyState != nil
}

func buildObjectDescriptor(headRef reference.Global, state payload.ObjectState) descriptor.Object {
	return descriptor.NewObject(
		headRef,
		state.Reference,
		state.Class,
		state.State,
		state.Parent,
	)
}
