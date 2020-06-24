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

var _ smachine.StateMachine = &SMJetDropBuilder{}

type SMJetDropBuilder struct {
	smachine.StateMachineDeclTemplate

	pulseSlot *conveyor.PulseSlot

	sd DropSharedData
}

func (p *SMJetDropBuilder) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMJetDropBuilder) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMJetDropBuilder) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
}

func (p *SMJetDropBuilder) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	switch {
	case p.pulseSlot.State() != conveyor.Present:
		return ctx.Error(throw.E("not a present pulse"))
	case !p.sd.LineRef.IsSelfScope():
		return ctx.Error(throw.E("wrong root"))
	}

	p.sd.limiter = smsync.NewSemaphore(0, fmt.Sprintf("SMLine{%d}.limiter", ctx.SlotLink().SlotID()))

	sdl := ctx.Share(&p.sd, 0)
	if !ctx.Publish(JetDropKey(p.sd.LineRef), sdl) {
		panic(throw.IllegalState())
	}

	return ctx.Jump(p.stepFindLine)
}
