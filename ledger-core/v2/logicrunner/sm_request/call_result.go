// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_request

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
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

	outgoingRef := *insolar.NewGlobalReference(*s.Payload.Caller.GetLocal(), s.Payload.CallOutgoing)

	link, bgin := ctx.GetPublishedGlobalAliasAndBargeIn(outgoingRef)
	if link.IsZero() {
		// TODO: log
		return ctx.Stop()
	}
	if bgin == nil {
		// TODO: log
		return ctx.Stop()
	}
	if !bgin.IsValid() {
		// TODO: log
		return ctx.Stop()
	}

	done := bgin.CallWithParam(s.Payload)
	if !done {
		// TODO: log
		return ctx.Stop()
	}

	return ctx.Stop()
}
