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

type StateMachineExecutorResults struct {
	// input arguments
	Meta *payload.Meta
}

var declExecutorResults smachine.StateMachineDeclaration = &declarationExecutorResults{}

type declarationExecutorResults struct {
	smachine.StateMachineDeclTemplate
}

/* -------- Declaration ------------- */

func (declarationExecutorResults) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachineExecutorResults)
	return s.Init
}

func (declarationExecutorResults) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	_ = sm.(*StateMachineExecutorResults)
}

/* -------- Instance ------------- */

func (s *StateMachineExecutorResults) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declExecutorResults
}

func (s *StateMachineExecutorResults) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
