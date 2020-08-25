// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SMVCallResult struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCallResult
}

/* -------- Declaration ------------- */

var dSMVCallResultInstance smachine.StateMachineDeclaration = &dSMVCallResult{}

type dSMVCallResult struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVCallResult) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ injector.DependencyInjector) {
}

func (*dSMVCallResult) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVCallResult)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVCallResult) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVCallResultInstance
}

func (s *SMVCallResult) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVCallResult) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.Payload.CallType != payload.CallTypeMethod && s.Payload.CallType != payload.CallTypeConstructor {
		panic(throw.IllegalValue())
	}

	link, bargeInCallback := ctx.GetPublishedGlobalAliasAndBargeIn(s.Payload.CallOutgoing)
	if link.IsZero() {
		return ctx.Error(throw.E("no one is waiting"))
	}
	if bargeInCallback == nil {
		return ctx.Error(throw.Impossible())
	}

	done := bargeInCallback.CallWithParam(s.Payload)
	if !done {
		return ctx.Error(throw.E("no one is waiting anymore"))
	}

	return ctx.Stop()
}
