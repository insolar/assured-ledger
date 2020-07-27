// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
//go:generate sm-uml-gen -f $GOFILE

package finalizedstate

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SMStateFinalizer struct {
	Reference reference.Global

	Report payload.VStateReport

	smachine.StateMachineDeclTemplate

	// dependencies
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
}

/* -------- Declaration ------------- */

func (*SMStateFinalizer) InjectDependencies(stateMachine smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := stateMachine.(*SMStateFinalizer)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.pulseSlot)
}

func (*SMStateFinalizer) GetInitStateFor(stateMachine smachine.StateMachine) smachine.InitFunc {
	return stateMachine.(*SMStateFinalizer).Init
}

/* -------- Instance ------------- */

func (sm *SMStateFinalizer) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *SMStateFinalizer) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (sm *SMStateFinalizer) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if sm.Reference.IsZero() {
		panic(throw.IllegalState())
	}

	return ctx.Jump(sm.stepSendVStateReport)
}

func (sm *SMStateFinalizer) stepSendVStateReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		currentPulseNumber = sm.pulseSlot.CurrentPulseNumber()
	)

	msg := sm.Report

	msg.AsOf = sm.pulseSlot.PulseData().PulseNumber

	sm.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, affinity.DynamicRoleVirtualExecutor, sm.Reference, currentPulseNumber)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send state", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	// after report is sent we can stop
	ctx.SetDefaultMigration(sm.migrationDefault)

	return ctx.Jump(sm.stepWaitIndefinitely)
}

func (sm *SMStateFinalizer) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
