// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
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
	pulseSlot     *conveyor.PulseSlot
}

var dSMVStateReportInstance smachine.StateMachineDeclaration = &dSMVStateReport{}

type dSMVStateReport struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVStateReport) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVStateReport)

	injector.MustInject(&s.objectCatalog)
	injector.MustInject(&s.pulseSlot)
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
	if s.pulseSlot.State() != conveyor.Present {
		ctx.Log().Warn("stop processing VStateReport since we are not in present pulse")
		return ctx.Stop()
	}

	ctx.SetDefaultMigration(s.migrationDefault)
	return ctx.Jump(s.stepProcess)
}

func (s *SMVStateReport) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.Log().Trace("stop processing VStateReport since pulse was changed")
	return ctx.Stop()
}

type stateAlreadyExistsErrorMsg struct {
	*log.Msg  `txt:"State already exists"`
	Reference string
	GotState  string
}

func (s *SMVStateReport) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.Payload.Status == payload.StateStatusInvalid {
		return ctx.Error(throw.IllegalValue())
	}

	if s.Payload.Status >= payload.StateStatusEmpty && s.gotLatestDirty() {
		return ctx.Error(throw.IllegalValue())
	}

	asOfPulse := s.Payload.AsOf
	prevPulse := s.pulseSlot.PrevOperationPulseNumber()

	// We expected state report only from previous executor and previous pulse.
	if asOfPulse < prevPulse {
		ctx.Log().Trace(struct {
			*log.Msg  `txt:"stop processing VStateReport, AsOf pulse is outdated"`
			asOfPulse string
			prevPulse string
		}{
			asOfPulse: asOfPulse.String(),
			prevPulse: prevPulse.String(),
		})

		return ctx.Jump(s.stepAsOfOutdated)
	}

	objectRef := s.Payload.Object.GetValue()

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

func (s *SMVStateReport) stepAsOfOutdated(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (s *SMVStateReport) updateSharedState(
	state *object.SharedState,
) {
	objectRef := s.Payload.Object.GetValue()

	var objState object.State
	switch s.Payload.Status {
	case payload.StateStatusInvalid:
		panic(throw.IllegalValue())
	case payload.StateStatusReady:
		if !s.gotLatestDirty() {
			panic(throw.IllegalState())
		}
		objState = object.HasState
	case payload.StateStatusEmpty:
		objState = object.Empty
	case payload.StateStatusInactive:
		objState = object.Inactive
	case payload.StateStatusMissing:
		objState = object.Missing
	default:
		panic(throw.IllegalValue())
	}

	if s.Payload.Status >= payload.StateStatusEmpty && s.gotLatestDirty() {
		panic(throw.IllegalState())
	}

	state.PreviousExecutorUnorderedPendingCount = uint8(s.Payload.UnorderedPendingCount)
	state.PreviousExecutorOrderedPendingCount = uint8(s.Payload.OrderedPendingCount)

	state.OrderedPendingEarliestPulse = s.Payload.OrderedPendingEarliestPulse
	state.UnorderedPendingEarliestPulse = s.Payload.UnorderedPendingEarliestPulse

	if s.gotLatestDirty() {
		dirty := *s.Payload.ProvidedContent.LatestDirtyState
		desc := buildObjectDescriptor(objectRef, dirty)
		state.SetDescriptorDirty(desc)
	}
	if s.gotLatestValidated() {
		validated := *s.Payload.ProvidedContent.LatestValidatedState
		desc := buildObjectDescriptor(objectRef, validated)
		state.SetDescriptorValidated(desc)
	}

	state.SetState(objState)
}

func (s *SMVStateReport) gotLatestDirty() bool {
	content := s.Payload.ProvidedContent
	return content != nil && content.LatestDirtyState != nil
}

func (s *SMVStateReport) gotLatestValidated() bool {
	content := s.Payload.ProvidedContent
	return content != nil && content.LatestValidatedState != nil
}

func buildObjectDescriptor(headRef reference.Global, state payload.ObjectState) descriptor.Object {
	return descriptor.NewObject(
		headRef,
		state.Reference.GetValueWithoutBase(),
		state.Class.GetValue(),
		state.State.GetBytes(),
		state.Deactivated,
	)
}
