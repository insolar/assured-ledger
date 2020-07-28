// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package conveyor

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PulseSlotConfig struct {
	config         smachine.SlotMachineConfig
	eventCallback  func()
	parentRegistry injector.DependencyRegistry
}

func NewPulseSlotMachine(config PulseSlotConfig, pulseManager *PulseDataManager) *PulseSlotMachine {
	psm := &PulseSlotMachine{
		pulseSlot: PulseSlot{pulseManager: pulseManager},
	}

	w := sworker.NewAttachableSimpleSlotWorker()
	psm.innerWorker = w
	psm.innerMachine = smachine.NewSlotMachine(config.config,
		combineCallbacks(w.WakeupWorkerOnEvent, config.eventCallback),
		combineCallbacks(w.WakeupWorkerOnSignal, config.eventCallback),
		config.parentRegistry)

	return psm
}

func combineCallbacks(mainFn, auxFn func()) func() {
	switch {
	case mainFn == nil:
		panic("illegal state")
	case auxFn == nil:
		return mainFn
	default:
		return func() {
			mainFn()
			auxFn()
		}
	}
}

type PulseSlotMachine struct {
	smachine.StateMachineDeclTemplate

	innerMachine *smachine.SlotMachine
	innerWorker  smachine.AttachableSlotWorker
	pulseSlot    PulseSlot // injectable for innerMachine's slots

	finalizeFn func()
	selfLink   smachine.SlotLink
}

func (p *PulseSlotMachine) SlotLink() smachine.SlotLink {
	if p.selfLink.IsZero() {
		panic("illegal state")
	}
	return p.selfLink
}

/* ================ Conveyor control ================== */

func (p *PulseSlotMachine) activate(workerCtx context.Context,
	addFn func(context.Context, smachine.StateMachine, smachine.CreateDefaultValues) (smachine.SlotLink, bool),
) {
	if !p.selfLink.IsZero() {
		panic("illegal state")
	}
	if p.pulseSlot.State() != Antique {
		p.innerMachine.AddDependency(&p.pulseSlot)
	}

	ok := false
	p.selfLink, ok = addFn(workerCtx, p, smachine.CreateDefaultValues{TerminationHandler: p.onTerminate})
	if !ok {
		p.onTerminate(smachine.TerminationData{})
		panic(throw.IllegalState())
	}
}

func (p *PulseSlotMachine) setFuture(pd pulse.Data) {
	if !pd.IsValidExpectedPulsarData() {
		panic("illegal value")
	}

	switch {
	case p.pulseSlot.pulseData == nil:
		p.pulseSlot.pulseData = &futurePulseDataHolder{expected: pd}
	default:
		panic("illegal state")
	}
}

func (p *PulseSlotMachine) setPresent(pr pulse.Range, pulseStart time.Time) {
	switch {
	case p.pulseSlot.pulseData == nil || p.innerMachine.IsEmpty():
		pr.RightBoundData().EnsurePulsarData()
		p.pulseSlot.pulseData = &presentPulseDataHolder{pr: pr, at: pulseStart}
	default:
		p.pulseSlot.pulseData.MakePresent(pr, pulseStart)
	}
}

func (p *PulseSlotMachine) setPast() {
	if p.pulseSlot.pulseData == nil {
		panic("illegal state")
	}
	p.pulseSlot.pulseData.MakePast()
}

func (p *PulseSlotMachine) setAntique() {
	if p.pulseSlot.pulseData != nil {
		panic("illegal state")
	}
	p.pulseSlot.pulseData = &antiqueNoPulseDataHolder{}
}

func (p *PulseSlotMachine) setPulseForUnpublish(m *smachine.SlotMachine, pn pulse.Number) {
	if m == nil {
		panic("illegal value")
	}
	p.finalizeFn = func() {
		m.TryUnsafeUnpublish(pn)
	}
}

/* ================ State Machine ================== */

func (p *PulseSlotMachine) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *PulseSlotMachine) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	if p != sm {
		panic("illegal value")
	}
	return p.stepInit
}

func (p *PulseSlotMachine) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultErrorHandler(p.errorHandler)

	switch p.pulseSlot.State() {
	case Future:
		ctx.SetDefaultMigration(p.migrateFromFuture)
		return ctx.Jump(p.stepFutureLoop)
	case Present:
		ctx.SetDefaultMigration(p.migrateFromPresent)
		return ctx.Jump(p.stepPresentLoop)
	case Past:
		ctx.SetDefaultMigration(p.migratePast)
		return ctx.Jump(p.stepPastLoop)
	case Antique:
		ctx.SetDefaultMigration(p.migrateAntique)
		return ctx.Jump(p.stepPastLoop)
	default:
		panic("illegal state")
	}
}

func (p *PulseSlotMachine) errorHandler(smachine.FailureContext) {}

func (p *PulseSlotMachine) onTerminate(smachine.TerminationData) {
	p.innerMachine.RunToStop(p.innerWorker, synckit.NewNeverSignal())
	if p.finalizeFn != nil {
		p.finalizeFn()
	}
}

func (p *PulseSlotMachine) _runInnerMigrate(ctx smachine.MigrationContext) {
	// TODO PLAT-23 ensure that p.innerWorker is stopped or detached
	p.innerMachine.MigrateNested(ctx)
	p.pulseSlot.postMigrate(p.innerMachine.AsHolder())
}

/* ------------- Future handlers --------------- */

func (p *PulseSlotMachine) stepFutureLoop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.pulseSlot.pulseManager.isPreparingPulse() {
		return ctx.WaitAny().ThenRepeat()
	}
	p.innerMachine.ScanNested(ctx, smachine.ScanDefault, 0, p.innerWorker)
	return ctx.Poll().ThenRepeat()
}

func (p *PulseSlotMachine) migrateFromFuture(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migrateFromPresent)
	p._runInnerMigrate(ctx)
	return ctx.Jump(p.stepPresentLoop)
}

/* ------------- Present handlers --------------- */

const presentSlotCycleBoost = 1

func (p *PulseSlotMachine) stepPresentLoop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	repeatNow, nextPollTime := p.innerMachine.ScanNested(ctx, smachine.ScanDefault, 0, p.innerWorker)

	switch {
	case repeatNow:
		return ctx.Repeat(presentSlotCycleBoost)
	case !nextPollTime.IsZero():
		return ctx.WaitAnyUntil(nextPollTime).ThenRepeat()
	}
	return ctx.WaitAny().ThenRepeat()
}

// Conveyor direct barge-in
func (p *PulseSlotMachine) preparePulseChange(ctx smachine.BargeInContext, outFn PreparePulseChangeFunc) smachine.StateUpdate {
	// =================
	// HERE - initiate state calculations
	// =================

	if outFn != nil {
		// TODO temporary hack
		nshBytes := longbits.Bits512{}
		_, _ = rand.Read(nshBytes[:])
		outFn(beat.AckData{UpstreamState: api.UpstreamState{NodeState: cryptkit.NewDigest(nshBytes, "random")}})
	}

	if !isSlotInitialized(ctx) {
		// direct barge-in has arrived BEFORE completion of init step
		// in this case we won't touch the slot
		return ctx.Stay()
	}

	return ctx.JumpExt(smachine.SlotStep{Transition: p.stepPreparingChange, Flags: smachine.StepPriority})
}

func (p *PulseSlotMachine) stepPreparingChange(ctx smachine.ExecutionContext) smachine.StateUpdate {
	repeatNow, nextPollTime := p.innerMachine.ScanNested(ctx, smachine.ScanPriorityOnly, 0, p.innerWorker)

	switch {
	case repeatNow:
		return ctx.Repeat(presentSlotCycleBoost)
	case !nextPollTime.IsZero():
		return ctx.WaitAnyUntil(nextPollTime).ThenRepeat()
	case p.innerMachine.HasPriorityWork(): // this is a concurrency-unsafe method
		return ctx.Yield().ThenRepeat()
	}
	return ctx.WaitAny().ThenRepeat()
}

// WARNING! this check doesn't cover the case of <replace_init>, but it is not a case here
func isSlotInitialized(ctx smachine.BasicContext) bool {
	sl, _ := ctx.SlotLink().GetStepLink()
	return sl.StepNo() > 1
}

// Conveyor direct barge-in
func (p *PulseSlotMachine) cancelPulseChange(ctx smachine.BargeInContext) smachine.StateUpdate {

	if !isSlotInitialized(ctx) {
		// direct barge-in has arrived BEFORE completion of init step
		// in this case we won't touch the slot
		return ctx.Stay()
	}

	return ctx.Jump(p.stepPresentLoop)
}

func (p *PulseSlotMachine) migrateFromPresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migratePast)
	p._runInnerMigrate(ctx)
	return ctx.Jump(p.stepPastLoop)
}

/* ------------- Past handlers --------------- */

func (p *PulseSlotMachine) stepPastLoop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.pulseSlot.pulseManager.isPreparingPulse() {
		return ctx.WaitAny().ThenRepeat()
	}

	repeatNow, nextPollTime := p.innerMachine.ScanNested(ctx, smachine.ScanDefault, 0, p.innerWorker)

	switch {
	case repeatNow:
		return ctx.Yield().ThenRepeat()
	case !nextPollTime.IsZero():
		return ctx.WaitAnyUntil(nextPollTime).ThenRepeat()
	}
	return ctx.WaitAny().ThenRepeat()
}

func (p *PulseSlotMachine) migratePast(ctx smachine.MigrationContext) smachine.StateUpdate {
	p._runInnerMigrate(ctx)

	if p.innerMachine.IsEmpty() {
		ctx.UnpublishAll()
		return ctx.Stop()
	}
	return ctx.Stay()
}

func (p *PulseSlotMachine) migrateAntique(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SkipMultipleMigrations()
	p._runInnerMigrate(ctx)
	return ctx.Stay()
}
