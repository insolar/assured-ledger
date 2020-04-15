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

type StateMachineAbandonedRequests struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.AbandonedRequestsNotification
}

var declAbandonedRequests smachine.StateMachineDeclaration = &declarationAbandonedRequests{}

type declarationAbandonedRequests struct {
	smachine.StateMachineDeclTemplate
}

func (declarationAbandonedRequests) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachineAbandonedRequests)
	return s.Init
}

func (declarationAbandonedRequests) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	_ = sm.(*StateMachineAbandonedRequests)
}

/* -------- Instance ------------- */

func (s *StateMachineAbandonedRequests) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declAbandonedRequests
}

func (s *StateMachineAbandonedRequests) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}

// func (s *StateMachineAbandonedRequests) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
// 	return ctx.Stop()
// }
