// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/lmn"
)

type SMLRegisterResponse struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.LRegisterResponse
}

/* -------- Declaration ------------- */

var dSMLRegisterResponseInstance smachine.StateMachineDeclaration = &dSMLRegisterResponse{}

type dSMLRegisterResponse struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMLRegisterResponse) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
}

func (*dSMLRegisterResponse) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMLRegisterResponse)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMLRegisterResponse) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMLRegisterResponseInstance
}

func (s *SMLRegisterResponse) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMLRegisterResponse) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	key := lmn.NewResultAwaitKey(s.Payload.AnticipatedRef, s.Payload.Flags)

	switch link, bargeInCallback := ctx.GetPublishedGlobalAliasAndBargeIn(key); {
	case link.IsZero():
		return ctx.Error(throw.E("no one is waiting"))
	case bargeInCallback == nil:
		return ctx.Error(throw.Impossible())
	case !bargeInCallback.CallWithParam(s.Payload):
		return ctx.Error(throw.E("no one is waiting anymore"))
	}

	return ctx.Stop()
}
