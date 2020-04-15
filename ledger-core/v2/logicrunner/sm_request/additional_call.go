// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_request // nolint:golint

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type StateMachineAdditionalCall struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.AdditionalCallFromPreviousExecutor
}

var declAdditionalCall smachine.StateMachineDeclaration = &declarationAdditionalCall{}

type declarationAdditionalCall struct {
	smachine.StateMachineDeclTemplate
}

func (declarationAdditionalCall) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachineAdditionalCall)
	return s.Init
}

func (declarationAdditionalCall) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	_ = sm.(*StateMachineAdditionalCall)
}

/* -------- Instance ------------- */

func (s *StateMachineAdditionalCall) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declAdditionalCall
}

func (s *StateMachineAdditionalCall) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
