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
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
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
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMStreamDropBuilder{}

type SMStreamDropBuilder struct {
	smachine.StateMachineDeclTemplate

	// injected
	pulseSlot  *conveyor.PulseSlot
	builderSvc *buildersvc.Adapter
	cataloger  DropCataloger

<<<<<<< HEAD
<<<<<<< HEAD
	// shared unbound
	sd *StreamSharedData

	// runtime
	jets      []jet.PrefixedID
=======
=======
>>>>>>> Ledger SMs
	// input & shared
	sd   *StreamSharedData

	// runtime
	jets []buildersvc.JetID
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
}

func (p *SMStreamDropBuilder) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMStreamDropBuilder) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMStreamDropBuilder) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.builderSvc)
	injector.MustInject(&p.cataloger)
}

func (p *SMStreamDropBuilder) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	if p.pulseSlot.State() != conveyor.Present {
		return ctx.Error(throw.E("not a present pulse"))
	}

<<<<<<< HEAD
<<<<<<< HEAD
	pr, _ := p.pulseSlot.PulseRange()
	p.sd = RegisterStreamDrop(ctx, pr)
	if p.sd == nil {
		panic(throw.IllegalState())
	}

	ctx.SetDefaultMigration(p.migratePresent)
	return ctx.Jump(p.stepCreateStreamDrop)
}

// func (p *SMStreamDropBuilder) getPrevDropPulseNumber() pulse.Number {
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

func (p *SMStreamDropBuilder) stepCreateStreamDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {

	// TODO get jetTree, online population

	pr := p.sd.pr
	return p.builderSvc.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
		jetAssist, jets := svc.CreateStreamDrop(pr)

		return func(ctx smachine.AsyncResultContext) {
			if jetAssist == nil {
				panic(throw.IllegalValue())
			}
			p.sd.jetAssist = jetAssist
			p.jets = jets
		}
	}).DelayedStart().Sleep().ThenJump(p.stepCreateJetDrops)
}

func (p *SMStreamDropBuilder) stepCreateJetDrops(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sd.jetAssist == nil {
=======
=======
>>>>>>> Ledger SMs
	sd := &StreamSharedData{
		ready: smsync.NewConditionalBool(false, fmt.Sprintf("SMStreamDropBuilder{%d}.ready", ctx.SlotLink().SlotID())),
	}
	sd.pr, _ = p.pulseSlot.PulseRange()

	if sd.pr == nil {
		panic(throw.IllegalState())
	}

	sdl := ctx.Share(sd, smachine.ShareDataUnbound)
	if !ctx.Publish(StreamDropKey(p.pulseSlot.PulseNumber()), sdl) {
		panic(throw.IllegalState())
	}
	p.sd = sd

	ctx.SetDefaultMigration(p.migrate)
	return ctx.Jump(p.stepCreateStreamDrop)
}

func (p *SMStreamDropBuilder) stepCreateStreamDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {

	// TODO get jetTree, pulse data, online population

	pr := p.sd.pr
	return p.builderSvc.PrepareAsync(ctx, func(svc buildersvc.Service) smachine.AsyncResultFunc {
		jets := svc.CreateStreamDrop(pr)
		if jets == nil {
			jets = []buildersvc.JetID{}
		}
		return func(ctx smachine.AsyncResultContext) {
			p.jets = jets
		}
	}).DelayedStart().Sleep().ThenJump(p.stepCreateJets)
}

func (p *SMStreamDropBuilder) stepCreateJets(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.jets == nil {
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
		return ctx.Sleep().ThenRepeat()
	}

	pn := p.pulseSlot.PulseNumber()
	for _, jetID := range p.jets {
<<<<<<< HEAD
<<<<<<< HEAD
		legID := jetID.AsLeg(pn)
		updater := p.sd.jetAssist.CreateJetDropAssistant(jetID.ID())
		p.cataloger.Create(ctx, legID, updater)
	}

	ctx.ApplyAdjustment(p.sd.enableAccess())
	return ctx.Stop()
}

func (p *SMStreamDropBuilder) migratePresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	// should NOT happen
	panic(throw.IllegalState())
=======
=======
>>>>>>> Ledger SMs
		p.cataloger.GetOrCreate(ctx, NewJetDropID(pn, jetID))
	}

	p.sd.enableAccess()
	ctx.ApplyAdjustment(p.sd.ready.NewValue(true))

	return ctx.Stop()
}

func (p *SMStreamDropBuilder) migrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
<<<<<<< HEAD
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
}

