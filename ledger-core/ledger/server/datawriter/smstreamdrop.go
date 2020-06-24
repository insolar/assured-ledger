// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
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

	// input & shared
	sd   *StreamSharedData

	// runtime
	jets []buildersvc.JetID
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
		return ctx.Sleep().ThenRepeat()
	}

	pn := p.pulseSlot.PulseNumber()
	for _, jetID := range p.jets {
		p.cataloger.GetOrCreate(ctx, NewJetDropID(pn, jetID))
	}

	p.sd.enableAccess()
	ctx.ApplyAdjustment(p.sd.ready.NewValue(true))

	return ctx.Stop()
}

func (p *SMStreamDropBuilder) migrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

