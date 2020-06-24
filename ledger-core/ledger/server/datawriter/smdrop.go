// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
<<<<<<< HEAD
<<<<<<< HEAD
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datareader"
=======
=======
>>>>>>> Ledger SMs
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMJetDropBuilder{}

type SMJetDropBuilder struct {
	smachine.StateMachineDeclTemplate

	pulseSlot *conveyor.PulseSlot

	sd DropSharedData
<<<<<<< HEAD
<<<<<<< HEAD
	prevReport datareader.PrevDropReport
=======
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
}

func (p *SMJetDropBuilder) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMJetDropBuilder) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMJetDropBuilder) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
}

func (p *SMJetDropBuilder) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
<<<<<<< HEAD
<<<<<<< HEAD
	if p.pulseSlot.State() != conveyor.Present {
		return ctx.Error(throw.E("not a present pulse"))
	}

	p.sd.prevReport = ctx.NewBargeInWithParam(func(v interface{}) smachine.BargeInCallbackFunc {
		report := v.(datareader.PrevDropReport)
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

const passiveWaitPortion = 50

func (p *SMJetDropBuilder) getPassiveDeadline(startedAt time.Time, pulseDelta uint16) time.Time {
	switch {
	case startedAt.IsZero():
		panic(throw.IllegalValue())
	case pulseDelta == 0:
		panic(throw.IllegalValue())
	}
	return startedAt.Add(time.Second * time.Duration(pulseDelta) / passiveWaitPortion)
}

func (p *SMJetDropBuilder) stepWaitPrevDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {

	passiveUntil := p.getPassiveDeadline(p.pulseSlot.PulseStartedAt(), p.pulseSlot.PulseData().NextPulseDelta)
	return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if !p.prevReport.IsZero() {
			return ctx.Jump(p.stepDropStart)
		}
		return ctx.WaitAnyUntil(passiveUntil).ThenRepeatOrJump(p.stepFindPrevDrop)
	})
}

func (p *SMJetDropBuilder) stepFindPrevDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.NewChild(func(smachine.ConstructionContext) smachine.StateMachine {
		return &datareader.SMFindDrop{
			Assistant: p.sd.updater,

			// this is safe as relevant fields are immutable
			// and inside this method there is a barge-in
			ReportFn: p.sd.SetPrevDropReport,
		}
	})

	return ctx.Jump(p.stepDropStart)
}

func (p *SMJetDropBuilder) stepDropStart(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.prevReport.IsZero() {
		return ctx.Sleep().ThenRepeat()
	}

	ctx.ApplyAdjustment(p.sd.enableAccess())
	return ctx.Jump(p.stepWaitPast)
}

func (p *SMJetDropBuilder) stepWaitPast(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

func (p *SMJetDropBuilder) migratePresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migratePast)
	return ctx.Jump(p.stepFinalize)
}

func (p *SMJetDropBuilder) migratePast(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (p *SMJetDropBuilder) stepFinalize(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO drop finalization
	return ctx.Stop()
}

func (p *SMJetDropBuilder) receivePrevReport(report datareader.PrevDropReport, ctx smachine.BargeInContext) (wakeup bool) {
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
	p.sd.addPrevReport(report)
	return true
}

func (p *SMJetDropBuilder) verifyPrevReport(report datareader.PrevDropReport) bool {
	// TODO verification vs jet tree etc
	return true
}

=======
=======
>>>>>>> Ledger SMs
	switch {
	case p.pulseSlot.State() != conveyor.Present:
		return ctx.Error(throw.E("not a present pulse"))
	case !p.sd.LineRef.IsSelfScope():
		return ctx.Error(throw.E("wrong root"))
	}

	p.sd.limiter = smsync.NewSemaphore(0, fmt.Sprintf("SMLine{%d}.limiter", ctx.SlotLink().SlotID()))

	sdl := ctx.Share(&p.sd, 0)
	if !ctx.Publish(JetDropKey(p.sd.LineRef), sdl) {
		panic(throw.IllegalState())
	}

	return ctx.Jump(p.stepFindLine)
}
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
