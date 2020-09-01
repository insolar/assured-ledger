// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requests

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMRead{}

type SMRead struct {
	smachine.StateMachineDeclTemplate

	// input
	request *rms.LReadRequest

	// injected
	pulseSlot  *conveyor.PulseSlot
	cataloger  datawriter.LineCataloger

	// runtime
	sdl          datawriter.LineDataLink
}

func (p *SMRead) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMRead) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.cataloger)
}

func (p *SMRead) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMRead) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	switch ps := p.pulseSlot.State(); {
	case p.request == nil:
		return ctx.Error(throw.E("missing request"))
	case p.request.TargetRef.IsEmpty():
		return ctx.Error(throw.E("missing target"))
	case ps == conveyor.Antique:
		panic(throw.NotImplemented()) // TODO alternative reader
	case ps == conveyor.Present:
		ctx.SetDefaultMigration(p.migratePresent)
	default:
		ctx.SetDefaultMigration(p.migratePast)
	}

	ctx.SetDefaultErrorHandler(p.handleError)

	if tpn := p.request.TargetRef.Get().GetLocal().Pulse(); tpn != p.pulseSlot.PulseNumber() {
		return ctx.Error(throw.E("wrong target pulse", struct { TargetPN, SlotPN pulse.Number }{ tpn, p.pulseSlot.PulseNumber() }))
	}

	return ctx.Jump(p.stepFindLine)
}

func (p *SMRead) stepFindLine(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sdl.IsZero() {
		normTargetRef := reference.NormCopy(p.request.TargetRef.Get())
		lineRef := reference.NewSelf(normTargetRef.GetBase())

		// TODO use Get after merge
		p.sdl = p.cataloger.GetOrCreate(ctx, lineRef)
		if p.sdl.IsZero() {
			panic(throw.IllegalState())
		}
	}

	var limiter smachine.SyncLink
	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		limiter = sd.GetActiveSync()
		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	if ctx.Acquire(limiter) {
		return ctx.Jump(p.stepLineIsReady)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.Acquire(limiter) {
			return ctx.Jump(p.stepLineIsReady)
		}
		return ctx.Sleep().ThenRepeat()
	})

}

func (p *SMRead) stepLineIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.ReleaseAll()

	isValid := false
	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		if !sd.IsValid() {
			return
		}
		isValid = true

		sd.TrimStages()
		// switch ok, idx, fut, rec := sd.FindWithTracker(); {
		//
		// }
		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	if !isValid {
		return ctx.Jump(p.stepSendFinalResponse)
	}

	return ctx.Error(throw.NotImplemented())
}

func (p *SMRead) stepSendFinalResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Error(throw.NotImplemented())
}

func (p *SMRead) migratePresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Error(throw.NotImplemented())
}

func (p *SMRead) migratePast(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Error(throw.NotImplemented())
}

func (p *SMRead) handleError(ctx smachine.FailureContext) {

}

