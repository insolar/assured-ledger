// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
)

type SMVDelegatedCallResponse struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VDelegatedCallResponse
}

var dSMVDelegatedCallResponseInstance smachine.StateMachineDeclaration = &dSMVDelegatedCallResponse{}

type dSMVDelegatedCallResponse struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVDelegatedCallResponse) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ injector.DependencyInjector) {
}

func (*dSMVDelegatedCallResponse) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVDelegatedCallResponse)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVDelegatedCallResponse) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVDelegatedCallResponseInstance
}

func (s *SMVDelegatedCallResponse) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVDelegatedCallResponse) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	key := execute.DelegationTokenAwaitKey{
		Outgoing: s.Payload.ResponseDelegationSpec.Outgoing.GetValue(),
	}
	slotLink, bargeInHolder := ctx.GetPublishedGlobalAliasAndBargeIn(key)
	if slotLink.IsZero() {
		return ctx.Error(errors.New("bargeIn was not published"))
	}
	if !bargeInHolder.CallWithParam(s.Payload) {
		return ctx.Error(errors.New("fail to call BargeIn"))
	}
	return ctx.Stop()
}
