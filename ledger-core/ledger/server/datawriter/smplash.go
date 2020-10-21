// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/treesvc"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMPlash{}

type SMPlash struct {
	smachine.StateMachineDeclTemplate

	// injected
	pulseSlot  *conveyor.PulseSlot
	builderSvc buildersvc.WriteAdapter
	cataloger  DropCataloger
	treeSvc    treesvc.Service

	// shared unbound
	sd *PlashSharedData

	// runtime
	jets         []jet.ExactID
	treePrev     jet.Tree
	treeCur      jet.Tree
	pulseChanger conveyor.PulseChanger
	plashWasClosed bool
}

func (p *SMPlash) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMPlash) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMPlash) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.builderSvc)
	injector.MustInject(&p.cataloger)
	injector.MustInject(&p.treeSvc)
}

func (p *SMPlash) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	if p.pulseSlot.State() != conveyor.Present {
		return ctx.Error(throw.E("not a present pulse"))
	}

	pr, _ := p.pulseSlot.PulseRange()
	p.sd = RegisterPlash(ctx, pr)
	if p.sd == nil {
		panic(throw.IllegalState())
	}

	ctx.SetDefaultMigration(p.migrateNotReady)
	return ctx.Jump(p.stepGetTree)
}

func (p *SMPlash) migrateNotReady(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Error(throw.IllegalState())
}

func (p *SMPlash) stepGetTree(ctx smachine.ExecutionContext) smachine.StateUpdate {
	pn := p.pulseSlot.PulseNumber()
	prev, curr, ok := p.treeSvc.GetTrees(pn) // TODO this is a simple implementation that is only valid for one-LMN network
	if !ok {
		// Trees' pulse can only be switched during migration
		panic(throw.Impossible())
	}

	p.treePrev, p.treeCur = &prev, &curr

	switch {
	case !curr.IsEmpty():
		return ctx.Jump(p.stepCreatePlush)
	case p.treeSvc.TryLockGenesis(pn):
		ctx.SetDefaultMigration(nil)
		return ctx.Jump(p.stepCreateGenesisPlush)
	default:
		// regular SMs are NOT allowed while genesis is running
		return ctx.Stop()
	}
}

func (p *SMPlash) makePlashConfig(ctx smachine.ExecutionContext) buildersvc.BasicPlashConfig {
	bd, _ := p.pulseSlot.BeatData()

	bi := ctx.NewBargeInWithParam(func(e interface{}) smachine.BargeInCallbackFunc {
		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			if e == nil {
				p.plashWasClosed = true
				return ctx.WakeUp()
			}
			return ctx.Error(e.(error))
		}
	})

	return buildersvc.BasicPlashConfig{
		PulseRange: p.sd.pr,
		Population: bd.Online,
		CallbackFn: func(err error) {
			bi.CallWithParam(err)
		},
	}
}

func (p *SMPlash) stepCreatePlush(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.treeCur.IsEmpty() {
		panic(throw.IllegalState())
	}

	plashCfg := p.makePlashConfig(ctx)

	return p.builderSvc.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
		jetAssist, pulseChanger, jets := svc.CreatePlash(plashCfg, p.treePrev, p.treeCur)

		return func(ctx smachine.AsyncResultContext) {
			if jetAssist == nil {
				panic(throw.IllegalValue())
			}
			p.sd.jetAssist = jetAssist
			p.pulseChanger = pulseChanger
			p.jets = jets
		}
	}).DelayedStart().Sleep().ThenJump(p.stepCreateJetDrops)
}

func (p *SMPlash) stepCreateJetDrops(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sd.jetAssist == nil {
		return ctx.Sleep().ThenRepeat()
	}

	pn := p.pulseSlot.PulseNumber()
	prevPN := p.pulseSlot.PrevOperationPulseNumber()
	if prevPN.IsUnknown() {
		// Ledger must have information about the immediately previous pulse
		ctx.Log().Warn("previous pulse is unavailable")
		return ctx.Stop()
	}

	for _, jetID := range p.jets {
		prevJet, pln := p.treePrev.GetPrefix(jetID.ID().AsPrefix())

		op := JetStraight
		switch bl := jetID.BitLen(); {
		case pln == bl:
		case pln > bl:
			op = JetMerge
		case p.treePrev.IsEmpty():
			op = JetGenesisSplit
		default:
			op = JetSplit
		}

		p.cataloger.Create(ctx, DropInfo{
			ID:         jetID.AsDrop(pn),
			PrevID:     prevJet.AsID().AsDrop(prevPN),
			LastOp:     op,
			AssistData: p.sd,
		}, p.sd.onDropStop)
	}

	if p.pulseChanger != nil && !p.pulseSlot.SetPulseChanger(p.pulseChanger) {
		return ctx.Error(throw.IllegalState())
	}

	ctx.ApplyAdjustment(p.sd.enableAccess())
	return ctx.Jump(p.stepWaitPast)
}

func (p *SMPlash) stepWaitPast(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migratePresentToPast)
	return ctx.Sleep().ThenRepeat()
}

func (p *SMPlash) migratePresentToPast(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migrateWaitClosedPlash)
	return ctx.Jump(p.stepWaitClosedPlash)
}

func (p *SMPlash) stepCreateGenesisPlush(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !p.treeCur.IsEmpty() {
		panic(throw.IllegalState())
	}

	plashCfg := p.makePlashConfig(ctx)

	return p.builderSvc.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
		jetAssist, pulseChanger, jetGenesis := svc.CreateGenesis(plashCfg)

		return func(ctx smachine.AsyncResultContext) {
			if jetAssist == nil  {
				panic(throw.IllegalValue())
			}
			p.sd.jetAssist = jetAssist
			p.pulseChanger = pulseChanger
			if jetGenesis != 0 {
				p.jets = []jet.ExactID{jetGenesis}
			}
		}
	}).DelayedStart().Sleep().ThenJump(p.stepGenesis)
}

func (p *SMPlash) stepGenesis(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sd.jetAssist == nil {
		return ctx.Sleep().ThenRepeat()
	}

	// NB! Genesis does NOT follow pulse changes

	switch len(p.jets) {
	case 1:
		jetGenesis := p.jets[0].AsLeg(p.pulseSlot.PulseNumber())

		dropInfo := DropInfo{
			ID:         jetGenesis.AsDrop(),
			PrevID:     jetGenesis.AsDrop(),
			LastOp:     JetGenesis,
			AssistData: p.sd,
		}
		cataloger := p.cataloger

		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &SMGenesis{
				// LegID remembers the pulse when genesis was started
				jetGenesis: jetGenesis,
				pulseChanger: p.pulseChanger,

				createDropFn: func(ctx smachine.ExecutionContext) {
					cataloger.Create(ctx, dropInfo, dropInfo.AssistData.onDropStop)
				},
			}
		})
	case 0:
		// this node can't run genesis
	default:
		panic(throw.Impossible())
	}

	// NB! Regular SMs can NOT be allowed to run during genesis-related pulse(s)
	// ctx.ApplyAdjustment(p.sd.enableAccess())

	// NB! Genesis can take multiple pulses - do NOT stop waiting
	ctx.SetDefaultMigration(nil)
	return ctx.Jump(p.stepWaitClosedPlash)
}

const plashClosingPortion = 0.25 // 1/4th of pulse

func (p *SMPlash) stepWaitClosedPlash(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !p.plashWasClosed {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(p.stepPlashClosed)
}

func (p *SMPlash) migrateWaitClosedPlash(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(nil)
	if !p.plashWasClosed {
		return ctx.Jump(p.stepWaitClosedPlashWithDeadline)
	}
	return ctx.Jump(p.stepPlashClosed)
}

func (p *SMPlash) stepWaitClosedPlashWithDeadline(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !p.plashWasClosed {
		return ctx.WaitAnyUntil(p.pulseSlot.PulseRelativeDeadline(plashClosingPortion)).ThenRepeatOrJump(p.stepPlashCloseTimeout)
	}
	return ctx.Jump(p.stepPlashClosed)
}

func (p *SMPlash) stepPlashClosed(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// do something else

	// stopping SMPlash should also force all weak SMLine(s) to be terminated
	return ctx.Stop()
}

func (p *SMPlash) stepPlashCloseTimeout(smachine.ExecutionContext) smachine.StateUpdate {
	panic(throw.NotImplemented()) // TODO force plash closing
}
