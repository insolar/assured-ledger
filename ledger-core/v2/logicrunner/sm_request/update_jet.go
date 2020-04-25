// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_request // nolint:golint

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_jet_storage"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_sender"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type StateMachineUpdateJet struct {
	// input arguments
	Meta *payload.Meta

	sender     *s_sender.SenderServiceAdapter
	jetStorage *s_jet_storage.JetStorageServiceAdapter

	externalError error
}

var declUpdateJet smachine.StateMachineDeclaration = &declarationUpdateJet{}

type declarationUpdateJet struct {
	smachine.StateMachineDeclTemplate
}

/* -------- Declaration ------------- */

func (declarationUpdateJet) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachineUpdateJet)
	return s.Init
}

func (declarationUpdateJet) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*StateMachineUpdateJet)

	injector.MustInject(&s.sender)
	injector.MustInject(&s.jetStorage)
}

/* -------- Instance ------------- */

func (s *StateMachineUpdateJet) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declUpdateJet
}

func (s *StateMachineUpdateJet) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepUpdateJet)
}

func (s *StateMachineUpdateJet) stepUpdateJet(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}
