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
	builderSvc *buildersvc.Adapter
	cataloger  DropCataloger

	// shared unbound
	sd *PlashSharedData

	// runtime
	jets      []jet.PrefixedID
}

func (p *SMPlash) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMPlash) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMPlash) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
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
	return ctx.Jump(p.stepCreateStreamDrop)
}

// func (p *SMPlash) getPrevDropPulseNumber() pulse.Number {
// 	pr, _ := p.pulseSlot.PulseRange()
//
// 	if !pr.IsArticulated() {
// 		if pnd := pr.LeftPrevDelta(); pnd != 0 {
// 			return pr.LeftBoundNumber().Prev(pnd)
// 		}
// 		return pulse.Unknown
// 	}
//
// 	prevPN := pulse.Unknown
// 	captureNext := true
// 	pr.EnumData(func(data pulse.Data) bool {
// 		switch {
// 		case data.PulseEpoch.IsArticulation():
// 			captureNext = true
// 		case !captureNext:
// 		case data.IsFirstPulse():
// 		default:
// 			captureNext = false
// 			prevPN = data.PrevPulseNumber()
// 		}
// 		return false
// 	})
// 	return prevPN
// }

func (p *SMPlash) stepCreateStreamDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {

	// TODO get jetTree, online population

	pr := p.sd.pr
	return p.builderSvc.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
		jetAssist, jets := svc.CreatePlash(pr)

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
		legID := jetID.AsLeg(pn)
		updater := p.sd.jetAssist.GetJetDropAssistant(jetID.ID())
		p.cataloger.Create(ctx, legID, updater)
	}

	ctx.ApplyAdjustment(p.sd.enableAccess())
	return ctx.Stop()
}

func (p *SMPlash) migratePresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	// should NOT happen
	panic(throw.IllegalState())
}

