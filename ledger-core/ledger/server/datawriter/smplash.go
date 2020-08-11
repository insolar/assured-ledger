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

	// shared unbound
	sd *PlashSharedData

	// runtime
	jets      []jet.ExactID
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

	ctx.SetDefaultMigration(p.migratePresent)
	return ctx.Jump(p.stepCreatePlush)
}

func (p *SMPlash) stepCreatePlush(ctx smachine.ExecutionContext) smachine.StateUpdate {


	bd, _ := p.pulseSlot.BeatData()
	pop := bd.Online

	// TODO get jetTree
	tree := jet.NewPrefixTree(true)

	// Empty tree will initiate genesis procedure, so we have to do a split
	tree.Split(0, 0)


	pr := p.sd.pr
	return p.builderSvc.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
		jetAssist, jets := svc.CreatePlash(pr, &tree, pop)

		return func(ctx smachine.AsyncResultContext) {
			if jetAssist == nil {
				panic(throw.IllegalValue())
			}
			p.sd.jetAssist = jetAssist
			p.jets = jets
		}
	}).DelayedStart().Sleep().ThenJump(p.stepCreateJetDrops)
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

func (p *SMPlash) migratePresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	// should NOT happen
	panic(throw.IllegalState())
}

