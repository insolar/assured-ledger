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
	sd LineSharedData

	// runtime
	sdl DropDataLink
	ssd *StreamSharedData
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
		return ctx.Error(throw.E("wrong root"))
	}

	p.sd.limiter = smsync.NewSemaphore(0, fmt.Sprintf("SMLine{%d}.limiter", ctx.SlotLink().SlotID()))

	sdl := ctx.Share(&p.sd, 0)
	if !ctx.Publish(LineKey(p.sd.lineRef), sdl) {
		panic(throw.IllegalState())
	}

	// ctx.SetDefaultMigration(p.migrate)
	// ctx.SetDefaultErrorHandler(p.handleError)
	return ctx.Jump(p.stepFindDrop)
}

func (p *SMLine) stepFindDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	sdl := p.streamer.GetOrCreate(ctx, p.pulseSlot.PulseNumber())
	if sdl.IsZero() {
		panic(throw.IllegalState())
	}

	p.ssd = sdl.GetSharedData(ctx)

	readySync := p.ssd.GetReadySync()

	if ctx.AcquireForThisStep(readySync) {
		return ctx.Jump(p.stepStreamIsReady)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.AcquireForThisStep(readySync) {
			return ctx.Jump(p.stepStreamIsReady)
		}
		return ctx.Sleep().ThenRepeat()
	})
}

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

}

