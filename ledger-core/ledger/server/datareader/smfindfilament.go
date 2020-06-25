// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
<<<<<<< HEAD
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMFindRecord{}

type SMFindRecord struct {
	smachine.StateMachineDeclTemplate

	Unresolved lineage.UnresolvedDependency
	FindCallback    smachine.BargeInWithParam
}

func (p *SMFindRecord) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMFindRecord) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMFindRecord) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
=======
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMFindFilament{}

type SMFindFilament struct {
	smachine.StateMachineDeclTemplate
	FilamentRootRef reference.Global
	FindCallback    smachine.BargeInWithParam
}

func (p *SMFindFilament) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMFindFilament) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMFindFilament) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
>>>>>>> SMs
	panic(throw.NotImplemented()) // TODO
}

