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
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type GenesisKey string
const GenesisKeyValue GenesisKey = "genesis"

var _ smachine.StateMachine = &SMGenesis{}

type SMGenesis struct {
	smachine.StateMachineDeclTemplate

	// provided by creator
	jetAssist buildersvc.PlashAssistant
	jetGenesis jet.ExactID

	// injected
	pulseSlot  *conveyor.PulseSlot
	builderSvc buildersvc.Adapter

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
}

func (p *SMGenesis) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	if !ctx.PublishGlobalAlias(GenesisKeyValue) {
		panic(throw.IllegalState())
	}

	p.lastPN = p.pulseSlot.PulseNumber()

	// Genesis procedure can run through multiple pulses
	ctx.SetDefaultMigration(p.migrateTracker)

	return ctx.Jump(p.stepPrepare)
}

func (p *SMGenesis) stepPrepare(ctx smachine.ExecutionContext) smachine.StateUpdate {

	// init and run genesis service here
	postGenesisTreeDepth := uint8(1)

	tree := jet.NewPrefixTree(false)
	tree.MakePerfect(postGenesisTreeDepth)

	panic("implement me")
}

func (p *SMGenesis) migrateTracker(ctx smachine.MigrationContext) smachine.StateUpdate {
	// access to CurrentPulseNumber() from migrate handler is guaranteed from racing
	p.lastPN = p.pulseSlot.CurrentPulseNumber()

	return ctx.Stay()
}

