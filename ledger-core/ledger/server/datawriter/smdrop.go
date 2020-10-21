// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datafinder"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMDropBuilder{}

type SMDropBuilder struct {
	smachine.StateMachineDeclTemplate

	// injected
	pulseSlot *conveyor.PulseSlot
	adapter    buildersvc.WriteAdapter

	sd         DropSharedData

	prevReport catalog.DropReport
	nextReport catalog.DropReport
	nextDrops  []jet.DropID
}

func (p *SMDropBuilder) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMDropBuilder) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMDropBuilder) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.adapter)
}

func (p *SMDropBuilder) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	if p.pulseSlot.State() != conveyor.Present {
		return ctx.Error(throw.E("not a present pulse"))
	}

	p.sd.ready = smsync.NewConditionalBool(false, fmt.Sprintf("StreamDrop{%d}.ready", ctx.SlotLink().SlotID()))
	p.sd.finalize = smsync.NewExclusive(fmt.Sprintf("StreamDrop{%d}.finalize", ctx.SlotLink().SlotID()))

	p.sd.prevReportBargein = ctx.NewBargeInWithParam(func(v interface{}) smachine.BargeInCallbackFunc {
		report := v.(catalog.DropReport)
		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			if p.receivePrevReport(report, ctx) {
				return ctx.WakeUp()
			}
			return ctx.Stay()
		}
	})

	if !RegisterJetDrop(ctx, &p.sd) {
		panic(throw.IllegalState())
	}

	ctx.SetDefaultMigration(p.migratePresent)
	return ctx.Jump(p.stepWaitPrevDrop)
}

const passiveWaitPortion = 0.02 // wait for 1/50th of pulse

func (p *SMDropBuilder) stepWaitPrevDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO this is a temporary stub
	p.prevReport.ReportRec = &rms.RCtlDropReport{}
	if !p.prevReport.IsZero() {
		return ctx.Jump(p.stepDropStart)
	}

	passiveUntil := p.pulseSlot.PulseRelativeDeadline(passiveWaitPortion)
	return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if !p.prevReport.IsZero() {
			return ctx.Jump(p.stepDropStart)
		}
		return ctx.WaitAnyUntil(passiveUntil).ThenRepeatOrJump(p.stepFindPrevDrop)
	})
}

func (p *SMDropBuilder) stepFindPrevDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.NewChild(func(smachine.ConstructionContext) smachine.StateMachine {
		return &datafinder.SMFindDrop{
			// this is safe as relevant fields are immutable
			// and inside this method there is a barge-in
			ReportFn: p.sd.SetPrevDropReport,
		}
	})

	return ctx.Jump(p.stepDropStart)
}

func (p *SMDropBuilder) stepDropStart(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.prevReport.IsZero() {
		return ctx.Sleep().ThenRepeat()
	}

	ctx.ApplyAdjustment(p.sd.enableAccess())
	return ctx.Jump(p.stepWaitPast)
}

func (p *SMDropBuilder) stepWaitPast(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

func (p *SMDropBuilder) migratePresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migratePast)
	ctx.SetDefaultFlags(smachine.StepPriority)
	return ctx.Jump(p.stepFinalize)
}

func (p *SMDropBuilder) migratePast(ctx smachine.MigrationContext) smachine.StateUpdate {
	// can't get here normally
	return ctx.Error(throw.IllegalState())
}

func (p *SMDropBuilder) stepFinalize(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !ctx.AcquireExt(p.sd.finalize, smachine.NoPriorityAcquire) {
		return ctx.Sleep().ThenRepeat()
	}

	if _, inactive := p.sd.finalize.GetCounts(); inactive > 0 {
		ctx.ReleaseAll()
		return ctx.Yield().ThenRepeat()
	}
	ctx.ReleaseAll()

	jetID := p.sd.info.ID
	p.adapter.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
		dropReport := svc.FinalizeDropSummary(jetID)

		return func(ctx smachine.AsyncResultContext) {
			p.nextReport = dropReport
			ctx.WakeUp()
		}
	}).Start()

	return ctx.Sleep().ThenJump(p.stepWaitNextReadyAndSendReport)
}

func (p *SMDropBuilder) stepWaitNextReadyAndSendReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.nextReport.IsZero() {
		return ctx.Sleep().ThenRepeat()
	}

	jetAssist := p.sd.info.AssistData.jetAssist
	// wait for confirmation from Plash of the next pulse / different slot
	ready := jetAssist.GetNextReadySync()
	if !ctx.Acquire(ready) {
		return ctx.Sleep().ThenRepeat()
	}

	p.nextDrops = jetAssist.CalculateNextDrops(p.sd.info.ID)
	return ctx.Jump(p.sendReport)
}

func (p *SMDropBuilder) receivePrevReport(report catalog.DropReport, ctx smachine.BargeInContext) (wakeup bool) {
	switch {
	case !p.prevReport.IsZero():
		if p.prevReport.Equal(report) {
			// all the same - no worries
			return false
		}
		// TODO report error
		panic(throw.NotImplemented())

	case !p.verifyPrevReport(report):
		ctx.Log().Error("invalid drop report", nil)
		return false
	}

	p.prevReport = report
	p.sd.setPrevReport(report)
	return true
}

func (p *SMDropBuilder) verifyPrevReport(report catalog.DropReport) bool {
	// TODO verification vs jet tree etc
	return !report.IsZero()
}

func (p *SMDropBuilder) sendReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO send dropReport to next LME
	return ctx.Stop()
}

