// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
//go:generate sm-uml-gen -f $GOFILE
package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SMVObjectValidationReport struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VObjectValidationReport

	// dependencies
	pulseSlot     *conveyor.PulseSlot
	messageSender messageSenderAdapter.MessageSender
	objectCatalog object.Catalog

	validatedObjectDescriptor descriptor.Object
	objectSharedState         object.SharedStateAccessor
}

/* -------- Declaration ------------- */

var dSMVObjectValidationReportInstance smachine.StateMachineDeclaration = &dSMVObjectValidationReport{}

type dSMVObjectValidationReport struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVObjectValidationReport) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVObjectValidationReport)

	injector.MustInject(&s.objectCatalog)
	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
}

func (*dSMVObjectValidationReport) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVObjectValidationReport)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVObjectValidationReport) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVObjectValidationReportInstance
}

func (s *SMVObjectValidationReport) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.Log().Trace("stop processing SMVObjectValidationReport since pulse was changed")
	return ctx.Stop()
}

func (s *SMVObjectValidationReport) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if s.Payload.Object.IsEmpty() || s.Payload.Validated.IsEmpty() {
		panic(throw.IllegalState())
	}

	ctx.SetDefaultMigration(s.migrationDefault)

	return ctx.Jump(s.stepGetObject)
}

func (s *SMVObjectValidationReport) stepGetObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.objectSharedState = s.objectCatalog.GetOrCreate(ctx, s.Payload.Object.GetValue())

	var (
		semaphoreReadyToWork smachine.SyncLink
		descriptorDirty      descriptor.Object
	)

	action := func(state *object.SharedState) {
		semaphoreReadyToWork = state.ReadyToWork
		descriptorDirty = state.DescriptorDirty()
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	if ctx.AcquireForThisStep(semaphoreReadyToWork).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	if descriptorDirty.HeadRef().Equal(s.Payload.Object.GetValue()) && descriptorDirty.State().GetLocal().Equal(s.Payload.Validated.GetValue()) {
		s.validatedObjectDescriptor = descriptorDirty
		return ctx.Jump(s.stepUpdateSharedState)
	}

	return ctx.Jump(s.stepGetMemory)
}

func (s *SMVObjectValidationReport) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

func (s *SMVObjectValidationReport) stepGetMemory(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subSM := &statemachine.SMGetCachedMemory{
		Object: s.Payload.Object.GetValue(), State: s.Payload.Validated.GetValue(),
	}
	return ctx.CallSubroutine(subSM, s.migrationDefault, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		if subSM.Result == nil {
			return ctx.Jump(s.stepWaitIndefinitely)
		}
		s.validatedObjectDescriptor = subSM.Result
		return ctx.Jump(s.stepUpdateSharedState)
	})
}

func (s *SMVObjectValidationReport) stepUpdateSharedState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if !state.IsReady() {
			ctx.Log().Trace(stateIsNotReady{Object: s.Payload.Object.GetValue()})
			return false
		}

		state.SetDescriptorValidated(s.validatedObjectDescriptor)

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
