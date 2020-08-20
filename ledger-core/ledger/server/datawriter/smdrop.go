// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datareader"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMDropBuilder{}

type SMDropBuilder struct {
	smachine.StateMachineDeclTemplate

	pulseSlot *conveyor.PulseSlot
	// catalog   PlashCataloger

	sd         DropSharedData
	prevReport datareader.PrevDropReport
	// jetAssist  *PlashSharedData
}

func (p *SMDropBuilder) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMDropBuilder) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMDropBuilder) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	// injector.MustInject(&p.catalog)
}

func (p *SMDropBuilder) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	if p.pulseSlot.State() != conveyor.Present {
		return ctx.Error(throw.E("not a present pulse"))
	}

	p.sd.prevReportBargein = ctx.NewBargeInWithParam(func(v interface{}) smachine.BargeInCallbackFunc {
		report := v.(datareader.PrevDropReport)
		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			if p.receivePrevReport(report, ctx) {
				return ctx.WakeUp()
			}
			return ctx.Stay()
		}
	})

	// p.jetAssist = p.catalog.Get(ctx, p.sd.id.CreatedAt())

	if !RegisterJetDrop(ctx, &p.sd) {
		panic(throw.IllegalState())
	}

	ctx.SetDefaultMigration(p.migratePresent)
	return ctx.Jump(p.stepWaitPrevDrop)
}

const passiveWaitPortion = 50 // wait for 1/50th of pulse

func (p *SMDropBuilder) getPassiveDeadline(startedAt time.Time, pulseDelta uint16) time.Time {
	switch {
	case startedAt.IsZero():
		panic(throw.IllegalValue())
	case pulseDelta == 0:
		panic(throw.IllegalValue())
	}
	return startedAt.Add(time.Second * time.Duration(pulseDelta) / passiveWaitPortion)
}

func (p *SMDropBuilder) stepWaitPrevDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO this is a temporary stub
	p.prevReport.ReportRec = &rms.RPrevDropReport{}
	if !p.prevReport.IsZero() {
		return ctx.Jump(p.stepDropStart)
	}

	passiveUntil := p.getPassiveDeadline(p.pulseSlot.PulseStartedAt(), p.pulseSlot.PulseData().NextPulseDelta)
	return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if !p.prevReport.IsZero() {
			return ctx.Jump(p.stepDropStart)
		}
		return ctx.WaitAnyUntil(passiveUntil).ThenRepeatOrJump(p.stepFindPrevDrop)
	})
}

func (p *SMDropBuilder) stepFindPrevDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.NewChild(func(smachine.ConstructionContext) smachine.StateMachine {
		return &datareader.SMFindDrop{
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
	return ctx.Jump(p.stepFinalize)
}

func (p *SMDropBuilder) migratePast(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (p *SMDropBuilder) stepFinalize(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO drop finalization
	return ctx.Stop()
}

func (p *SMDropBuilder) receivePrevReport(report datareader.PrevDropReport, ctx smachine.BargeInContext) (wakeup bool) {
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

func (p *SMDropBuilder) verifyPrevReport(report datareader.PrevDropReport) bool {
	// TODO verification vs jet tree etc
	return !report.IsZero()
}

