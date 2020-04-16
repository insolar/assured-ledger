// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package request

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
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

func (*dSMVCallResult) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
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
	return ctx.Stop()
}
