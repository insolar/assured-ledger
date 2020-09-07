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
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type GenesisKey string
const GenesisKeyValue GenesisKey = "genesis"
const DefaultGenesisSplitDepth = 2

var _ smachine.StateMachine = &SMGenesis{}

type SMGenesis struct {
	smachine.StateMachineDeclTemplate

	// provided by creator
	jetAssist buildersvc.PlashAssistant
	jetGenesis jet.LegID

	// injected
	pulseSlot  *conveyor.PulseSlot
	builderSvc buildersvc.WriteAdapter
	treeSvc    treesvc.Service

	// runtime
	lastPN     pulse.Number
}

func (p *SMGenesis) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMGenesis) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMGenesis) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.builderSvc)
	injector.MustInject(&p.treeSvc)
}

func (p *SMGenesis) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	if !ctx.PublishGlobalAlias(GenesisKeyValue) { // ensure only one
		panic(throw.IllegalState())
	}

	p.lastPN = p.pulseSlot.PulseNumber()

	// Genesis procedure can run through multiple pulses
	ctx.SetDefaultMigration(p.migrateTracker)

	return ctx.Jump(p.stepPrepare)
}

func (p *SMGenesis) stepPrepare(ctx smachine.ExecutionContext) smachine.StateUpdate {

	// init and run genesis service here
	//
	// ...

	// FinishGenesis must be immediately before the stop
	p.treeSvc.FinishGenesis(DefaultGenesisSplitDepth, p.lastPN)
	return ctx.Stop()
}

func (p *SMGenesis) migrateTracker(ctx smachine.MigrationContext) smachine.StateUpdate {
	// access to CurrentPulseNumber() from migrate handler is guaranteed from racing
	p.lastPN = p.pulseSlot.CurrentPulseNumber()

	return ctx.Stay()
}

