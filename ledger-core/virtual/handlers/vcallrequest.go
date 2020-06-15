// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
)

type SMVCallRequest struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCallRequest

	pulseSlot *conveyor.PulseSlot
}

/* -------- Declaration ------------- */

var dSMVCallRequestInstance smachine.StateMachineDeclaration = &dSMVCallRequest{}

type dSMVCallRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVCallRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMVCallRequest)

	injector.MustInject(&s.pulseSlot)
}

func (*dSMVCallRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVCallRequest)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVCallRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVCallRequestInstance
}

func (s *SMVCallRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepExecute)
}

func (s *SMVCallRequest) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &execute.SMExecute{Meta: s.Meta, Payload: s.Payload}
	})
}
