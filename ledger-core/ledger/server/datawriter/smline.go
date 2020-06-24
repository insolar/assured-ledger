// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
<<<<<<< HEAD
<<<<<<< HEAD
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

var _ smachine.StateMachine = &SMLine{}

type SMLine struct {
	smachine.StateMachineDeclTemplate

	// injected
	pulseSlot *conveyor.PulseSlot
	cataloger DropCataloger
	streamer  StreamDropCatalog

	// input & shared
<<<<<<< HEAD
<<<<<<< HEAD
	sd     LineSharedData
	onFind smachine.BargeInWithParam
=======
=======
>>>>>>> Ledger SMs
	sd LineSharedData

	// runtime
	sdl DropDataLink
	ssd *StreamSharedData
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
}

func (p *SMLine) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMLine) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMLine) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.cataloger)
	injector.MustInject(&p.streamer)
}

func (p *SMLine) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	switch {
	case p.pulseSlot.State() != conveyor.Present:
		return ctx.Error(throw.E("not a present pulse"))
	case !p.sd.lineRef.IsSelfScope():
<<<<<<< HEAD
<<<<<<< HEAD
		return ctx.Error(throw.E("wrong root - must be self"))
	case !p.sd.lineRef.IsObjectReference():
		return ctx.Error(throw.E("wrong root - must be object"))
	}

	if !RegisterLine(ctx, &p.sd) {
		panic(throw.IllegalState())
	}

	ctx.SetDefaultMigration(p.migratePresentNotReady)
	ctx.SetDefaultErrorHandler(p.handleError)
=======
=======
>>>>>>> Ledger SMs
		return ctx.Error(throw.E("wrong root"))
	}

	p.sd.limiter = smsync.NewSemaphore(0, fmt.Sprintf("SMLine{%d}.limiter", ctx.SlotLink().SlotID()))

	sdl := ctx.Share(&p.sd, 0)
	if !ctx.Publish(LineKey(p.sd.lineRef), sdl) {
		panic(throw.IllegalState())
	}

	// ctx.SetDefaultMigration(p.migrate)
	// ctx.SetDefaultErrorHandler(p.handleError)
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
	return ctx.Jump(p.stepFindDrop)
}

func (p *SMLine) stepFindDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
<<<<<<< HEAD
<<<<<<< HEAD
	ssd := p.streamer.GetOrCreate(ctx, p.pulseSlot.PulseNumber())
	if ssd == nil {
		panic(throw.IllegalState())
	}
	readySync := ssd.GetReadySync()

	if ctx.AcquireForThisStep(readySync) {
		p.sd.jetDropID = ssd.GetJetDrop(p.sd.lineRef)
		return ctx.Jump(p.stepDropIsCreated)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.AcquireForThisStep(readySync) {
			p.sd.jetDropID = ssd.GetJetDrop(p.sd.lineRef)
			return ctx.Jump(p.stepDropIsCreated)
		}
		return ctx.Sleep().ThenRepeat()
	})
}

func (p *SMLine) stepDropIsCreated(ctx smachine.ExecutionContext) smachine.StateUpdate {
	sdl := p.cataloger.Get(ctx, p.sd.jetDropID)
=======
	sdl := p.streamer.GetOrCreate(ctx, p.pulseSlot.PulseNumber())
>>>>>>> Ledger SMs
=======
	sdl := p.streamer.GetOrCreate(ctx, p.pulseSlot.PulseNumber())
>>>>>>> Ledger SMs
	if sdl.IsZero() {
		panic(throw.IllegalState())
	}

<<<<<<< HEAD
<<<<<<< HEAD
	var readySync smachine.SyncLink
	sdl.MustAccess(func(sd *DropSharedData) {
		readySync = sd.GetReadySync()
	})

	if ctx.AcquireForThisStep(readySync) {
		sdl.MustAccess(func(sd *DropSharedData) {
			p.sd.dropUpdater = sd.GetDropAssistant()
		})
		return ctx.Jump(p.stepDropIsReady)
=======
=======
>>>>>>> Ledger SMs
	p.ssd = sdl.GetSharedData(ctx)

	readySync := p.ssd.GetReadySync()

	if ctx.AcquireForThisStep(readySync) {
		return ctx.Jump(p.stepStreamIsReady)
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.AcquireForThisStep(readySync) {
<<<<<<< HEAD
<<<<<<< HEAD
			sdl.MustAccess(func(sd *DropSharedData) {
				p.sd.dropUpdater = sd.GetDropAssistant()
			})
			return ctx.Jump(p.stepDropIsReady)
=======
			return ctx.Jump(p.stepStreamIsReady)
>>>>>>> Ledger SMs
=======
			return ctx.Jump(p.stepStreamIsReady)
>>>>>>> Ledger SMs
		}
		return ctx.Sleep().ThenRepeat()
	})
}

<<<<<<< HEAD
<<<<<<< HEAD
func (p *SMLine) stepDropIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	pn := p.pulseSlot.PulseNumber()
	refPN := p.sd.lineRef.GetBase().GetPulseNumber()

	switch {
	case pn == refPN:
		// This is creation
		return ctx.Jump(p.stepLineIsReady)
	case pn < refPN:
		// It was before - find a recap
		sm := &datareader.SMFindRecap{ RootRef: p.sd.lineRef }
		return ctx.CallSubroutine(sm, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
			if sm.RecapRec == nil {
				// TODO Unknown object
				panic(throw.NotImplemented())
			}
			p.sd.addRecap(sm.RecapRef, sm.RecapRec)
			return ctx.Jump(p.stepLineIsReady)
		})
	default:
		panic(throw.Impossible())
	}
}

func (p *SMLine) stepLineIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	p.sd.valid = true
	p.onFind = ctx.NewBargeInWithParam(func(v interface{}) smachine.BargeInCallbackFunc {
		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			return ctx.WakeUp()
		}
	})
	ctx.ApplyAdjustment(p.sd.enableAccess())
	ctx.SetDefaultMigration(p.migratePresent)
	return ctx.Sleep().ThenJump(p.stepWaitForContextUpdates)
}

func (p *SMLine) stepWaitForContextUpdates(ctx smachine.ExecutionContext) smachine.StateUpdate {

	for _, ur := range p.sd.getUnresolved() {
		unresolved := ur
		ctx.NewChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &datareader.SMFindRecord{
				Unresolved: unresolved,
				FindCallback:    p.onFind,
			}
		})
	}

	return ctx.Sleep().ThenRepeat()
}

func (p *SMLine) handleError(ctx smachine.FailureContext) {
	sd := &LineSharedData{
		lineRef: p.sd.lineRef,
		limiter: p.sd.limiter,
	}
	sdl := ctx.Share(sd, smachine.ShareDataUnbound)
	if !ctx.PublishReplacement(LineKey(sd.lineRef), sdl) {
		panic(throw.Impossible())
	}
	ctx.ApplyAdjustment(sd.enableAccess())
}

func (p *SMLine) migratePresentNotReady(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Error(throw.E("failed to initialize"))
}

func (p *SMLine) migratePresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	p.sd.disableAccess()

	ctx.SetDefaultMigration(nil)
	return ctx.Jump(p.stepFinalize)
}

func (p *SMLine) stepFinalize(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO some finalization for lines?
	return ctx.Stop()
=======
=======
>>>>>>> Ledger SMs
func (p *SMLine) stepStreamIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	jetDropID := p.ssd.GetJetDrop(p.sd.lineRef)

	p.sdl = p.cataloger.GetOrCreate(ctx, jetDropID)
	if p.sdl.IsZero() {
		panic(throw.IllegalState())
	}
}


func (p *SMLine) getDropID() JetDropID {

}

func (p *SMLine) stepDropIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {

<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
}

