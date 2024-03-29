package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datafinder"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMLine{}

type SMLine struct {
	smachine.StateMachineDeclTemplate

	// injected
	pulseSlot *conveyor.PulseSlot
	cataloger DropCataloger
	plasher   PlashCataloger

	// input & shared
	sd     LineSharedData
	onFind smachine.BargeInWithParam
}

func (p *SMLine) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMLine) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMLine) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.cataloger)
	injector.MustInject(&p.plasher)
	injector.MustInject(&p.sd.adapter)
}

func (p *SMLine) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	switch {
	case p.pulseSlot.State() != conveyor.Present:
		return ctx.Error(throw.E("not a present pulse"))
	case !p.sd.lineRef.IsSelfScope():
		return ctx.Error(throw.E("wrong root - must be self"))
	case !p.sd.lineRef.IsObjectReference():
		return ctx.Error(throw.E("wrong root - must be object"))
	}

	if !RegisterLine(ctx, &p.sd) {
		panic(throw.IllegalState())
	}

	ctx.SetDefaultMigration(p.migratePresentNotReady)
	ctx.SetDefaultErrorHandler(p.handleError)
	return ctx.Jump(p.stepFindDrop)
}

func (p *SMLine) stepFindDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ssd := p.plasher.GetOrCreate(ctx, p.pulseSlot.PulseNumber())
	if ssd == nil {
		panic(throw.IllegalState())
	}
	readySync := ssd.GetReadySync()

	// NB! can't use AcquireForThisStep here as ctx.Sleep().ThenJump() will cancel it
	if ctx.Acquire(readySync) {
		ctx.ReleaseAll()
		p.sd.jetDropID = ssd.GetDrop(p.sd.lineRef)
		return ctx.Jump(p.stepDropIsCreated)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if !ctx.Acquire(readySync) {
			return ctx.Sleep().ThenRepeat()
		}
		ctx.ReleaseAll()
		p.sd.jetDropID = ssd.GetDrop(p.sd.lineRef)
		return ctx.Jump(p.stepDropIsCreated)
	})
}

func (p *SMLine) stepDropIsCreated(ctx smachine.ExecutionContext) smachine.StateUpdate {
	sdl := p.cataloger.Get(ctx, p.sd.jetDropID)
	if sdl.IsZero() {
		panic(throw.IllegalState())
	}

	var readySync smachine.SyncLink
	sdl.MustAccess(func(sd *DropSharedData) {
		readySync = sd.GetReadySync()
	})

	if ctx.Acquire(readySync) {
		ctx.ReleaseAll()
		sdl.MustAccess(func(sd *DropSharedData) {
			p.sd.onDropReady(sd)
		})
		return ctx.Jump(p.stepDropIsReady)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if !ctx.Acquire(readySync) {
			return ctx.Sleep().ThenRepeat()
		}
		ctx.ReleaseAll()
		sdl.MustAccess(func(sd *DropSharedData) {
			p.sd.onDropReady(sd)
		})
		return ctx.Jump(p.stepDropIsReady)
	})
}

func (p *SMLine) stepDropIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	pn := p.pulseSlot.PulseNumber()
	refPN := p.sd.lineRef.GetBase().GetPulseNumber()

	switch {
	case pn == refPN:
		// This is creation
		return ctx.Jump(p.stepLineIsReady)
	case pn < refPN:
		// It was before - find a recap
		sm := &datafinder.SMFindRecap{ RootRef: p.sd.lineRef }
		return ctx.CallSubroutine(sm, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
			if sm.RecapRec == nil {
				// TODO Unknown object
				panic(throw.NotImplemented())
			}
			// TODO p.sd.addRecap(sm.RecapRef, sm.RecapRec)
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
			return &datafinder.SMFindRecord{
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

	// make sure that the drop is blocked until all lines will issue a summary
	// but we don't need to wait here
	ctx.Acquire(p.sd.dropFinalizeSync)

	ctx.SetDefaultMigration(nil)
	return ctx.JumpExt(smachine.SlotStep{
		Transition: p.stepSummarize,
		Flags:      smachine.StepPriority,
	})
}

func (p *SMLine) stepSummarize(ctx smachine.ExecutionContext) smachine.StateUpdate {
	summary := p.sd.data.CreateSummary()
	if summary.IsZero() {
		ctx.ReleaseAll()
	} else {
		jetDropID := p.sd.jetDropID

		p.sd.adapter.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
			svc.AppendToDropSummary(jetDropID, summary)

			return func(ctx smachine.AsyncResultContext) {
				ctx.ReleaseAll()
			}
		}).WithoutAutoWakeUp().Start()
	}

	return ctx.JumpExt(smachine.SlotStep{
		Transition: p.stepWeakWaitIndefinitely,
		Flags:      smachine.StepWeak,
	})
}

func (p *SMLine) stepWeakWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// This step is marked as Weak and SM will be stopped by SlotMachine when only weak SM's remain
	return ctx.Sleep().ThenRepeat()
}

