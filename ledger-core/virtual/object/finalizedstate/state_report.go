// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package finalizedstate

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SMReport struct {
	ReadyToWork smachine.SyncLink

	Reference reference.Global
}

type SMStateReport struct {
	SMReport

	report payload.VStateReport

	smachine.StateMachineDeclTemplate

	reportShared SharedReportAccessor

	// dependencies
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
}

/* -------- Declaration ------------- */

func (sm *SMStateReport) InjectDependencies(stateMachine smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := stateMachine.(*SMStateReport)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.pulseSlot)
}

func (sm *SMStateReport) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

/* -------- Instance ------------- */

func (sm *SMStateReport) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *SMStateReport) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.Unpublish(BuildReportKey(sm.Reference, sm.pulseSlot.PulseData().PulseNumber))
	return ctx.Stop()
}

func (sm *SMStateReport) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	reportShared, ok := GetSharedStateReport(ctx, sm.Reference, sm.pulseSlot.PulseData().PulseNumber)
	if !ok {
		panic(throw.IllegalState())
	}

	sm.reportShared = reportShared

	return ctx.Jump(sm.stepFillReport)
}

func (sm *SMStateReport) stepFillReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	action := func(report payload.VStateReport) {
		sm.report = report
	}

	switch sm.reportShared.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(sm.reportShared.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	return ctx.Jump(sm.stepSendVStateReport)
}

func (sm *SMStateReport) stepSendVStateReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		currentPulseNumber = sm.pulseSlot.CurrentPulseNumber()
	)

	msg := sm.report

	msg.AsOf = sm.pulseSlot.PulseData().PulseNumber

	sm.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, node.DynamicRoleVirtualExecutor, sm.Reference, currentPulseNumber)
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

func (sm *SMStateReport) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
