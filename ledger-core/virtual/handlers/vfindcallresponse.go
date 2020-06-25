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
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
)

type SMVFindCallResponse struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VFindCallResponse
}

/* -------- Declaration ------------- */

var dSMVFindCallResponseInstance smachine.StateMachineDeclaration = &dSMVFindCallResponse{}

type dSMVFindCallResponse struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVFindCallResponse) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (*dSMVFindCallResponse) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVFindCallResponse)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVFindCallResponse) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVFindCallResponseInstance
}

func (s *SMVFindCallResponse) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVFindCallResponse) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	key := execute.DeduplicationBargeInKey{
		Callee: s.Payload.Callee,
		Outgoing: s.Payload.Outgoing,
	}

	link, bargeInCallback := ctx.GetPublishedGlobalAliasAndBargeIn(key)
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
