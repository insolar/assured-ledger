// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datafinder

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMFindDrop{}

type SMFindDrop struct {
	smachine.StateMachineDeclTemplate

//	Assistant buildersvc.JetDropAssistant
	ReportFn  func(report PrevDropReport) // to avoid circular dependency
}

func (p *SMFindDrop) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMFindDrop) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMFindDrop) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	panic(throw.NotImplemented()) // TODO
}

