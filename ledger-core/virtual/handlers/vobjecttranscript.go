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
)

type SMVObjectTranscript struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VObjectTranscript
}

/* -------- Declaration ------------- */

var dSMVObjectTranscriptInstance smachine.StateMachineDeclaration = &dSMVObjectTranscript{}

type dSMVObjectTranscript struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVObjectTranscript) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	_ = sm.(*SMVObjectTranscript)
}

func (*dSMVObjectTranscript) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVObjectTranscript)
	return s.Init
}

/* -------- Instance ------------- */

func (*SMVObjectTranscript) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	_ = sm.(*SMVObjectTranscript)
}

func (s *SMVObjectTranscript) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVObjectTranscriptInstance
}

func (s *SMVObjectTranscript) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVObjectTranscript) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}
