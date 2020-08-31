// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMRecentRead{}

type SMRecentRead struct {
	smachine.StateMachineDeclTemplate

}

func (p *SMRecentRead) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMRecentRead) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMRecentRead) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	panic(throw.NotImplemented()) // TODO
}

