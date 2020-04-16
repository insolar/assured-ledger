// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_request // nolint:golint

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type StateMachineCallResult struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCallResult
}

var declCallResult smachine.StateMachineDeclaration = &declarationCallResult{}

type declarationCallResult struct {
	smachine.StateMachineDeclTemplate
}

func (declarationCallResult) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachineCallResult)
	return s.Init
}

func (s *StateMachineCallResult) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declCallResult
}

func (s *StateMachineCallResult) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *StateMachineCallResult) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.Payload.CallType != payload.CTMethod && s.Payload.CallType != payload.CTConstructor {
		panic(throw.IllegalValue())
	}

	outgoingRef := reference.NewGlobal(*s.Payload.Caller.GetLocal(), s.Payload.CallOutgoing)
	link, bgin := ctx.GetPublishedGlobalAliasAndBargeIn(outgoingRef)
	if link.IsZero() {
		return ctx.Error(throw.E("no one is waiting", struct{ smachine.SlotLink }{link}))
	}
	if bgin == nil {
		return ctx.Error(throw.E("impossible situation ", struct{ smachine.BargeInHolder }{bgin}))
	}

	done := bgin.CallWithParam(s.Payload)
	if !done {
		return ctx.Error(throw.E("no one is waiting anymore", struct{ *payload.VCallResult }{s.Payload}))
	}

	return ctx.Stop()
}
