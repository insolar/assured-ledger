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
	builderSvc buildersvc.Adapter
	cataloger  DropCataloger
	treeSvc    treesvc.Service

	// shared unbound
	sd *PlashSharedData

	// runtime
	jets      []jet.ExactID
	treePrev  jet.Tree
	treeCur   jet.Tree
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

	ctx.SetDefaultMigration(func(ctx smachine.MigrationContext) smachine.StateUpdate {
		panic(throw.IllegalState())
	})
	return ctx.Jump(p.stepGetTree)
}

func (p *SMPlash) stepGetTree(ctx smachine.ExecutionContext) smachine.StateUpdate {
	prev, curr := p.treeSvc.GetTrees() // TODO this is a simple implementation that is only valid for one-LMN network
	p.treePrev, p.treeCur = &prev, &curr

	return ctx.Jump(p.stepCreatePlush)
}

func (p *SMPlash) stepCreatePlush(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.treeCur == nil {
		return ctx.Sleep().ThenRepeat()
	}

	bd, _ := p.pulseSlot.BeatData()
	pop := bd.Online

	pr := p.sd.pr

	switch {
	case p.treeCur.IsEmpty():
		if p.treePrev != nil && !p.treePrev.IsEmpty() {
			panic(throw.IllegalState())
		}

		return p.builderSvc.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
			jetAssist, jetGenesis := svc.CreateGenesis(pr, pop)

			return func(ctx smachine.AsyncResultContext) {
				if jetAssist == nil {
					panic(throw.IllegalValue())
				}
				p.sd.jetAssist = jetAssist
				if jetGenesis != 0 {
					p.jets = []jet.ExactID{jetGenesis}
				}
			}
		}).DelayedStart().Sleep().ThenJump(p.stepGenesis)

	case p.treePrev == nil:
		panic(throw.IllegalState())
	default:

		return p.builderSvc.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
			jetAssist, jets := svc.CreatePlash(pr, p.treePrev, p.treeCur, pop)

			return func(ctx smachine.AsyncResultContext) {
				if jetAssist == nil {
					panic(throw.IllegalValue())
				}
				p.sd.jetAssist = jetAssist
				p.jets = jets
			}
		}).DelayedStart().Sleep().ThenJump(p.stepCreateJetDrops)
	}
}

func (p *SMPlash) stepCreateJetDrops(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sd.jetAssist == nil {
		return ctx.Sleep().ThenRepeat()
	}

	pn := p.pulseSlot.PulseNumber()
	for _, jetID := range p.jets {
		p.cataloger.Create(ctx, jetID, pn)
	}

	ctx.ApplyAdjustment(p.sd.enableAccess())
	return ctx.Stop()
}

func (p *SMPlash) stepGenesis(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sd.jetAssist == nil {
		return ctx.Sleep().ThenRepeat()
	}

	switch len(p.jets) {
	case 1:
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &SMGenesis{
				jetAssist: p.sd.jetAssist,
				// LegID remembers the pulse when genesis was started
				jetGenesis: p.jets[0].AsLeg(p.pulseSlot.PulseNumber()),
			}
		})
	case 0:
		// this node can't run genesis
	default:
		panic(throw.Impossible())
	}

	// NB! Regular SMs can NOT be allowed to run during genesis-related pulse(s)
	// ctx.ApplyAdjustment(p.sd.enableAccess())
	return ctx.Stop()
}

