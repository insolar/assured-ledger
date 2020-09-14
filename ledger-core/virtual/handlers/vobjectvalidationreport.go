// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
//go:generate sm-uml-gen -f $GOFILE
package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/statemachine"
)

type SMVObjectValidationReport struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VObjectValidationReport

	// dependencies
	pulseSlot     *conveyor.PulseSlot
	messageSender messageSenderAdapter.MessageSender

	objDesc descriptor.Object
}

/* -------- Declaration ------------- */

var dSMVObjectValidationReportInstance smachine.StateMachineDeclaration = &dSMVObjectValidationReport{}

type dSMVObjectValidationReport struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVObjectValidationReport) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVObjectValidationReport)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
}

func (*dSMVObjectValidationReport) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVObjectValidationReport)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVObjectValidationReport) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVObjectValidationReportInstance
}

func (s *SMVObjectValidationReport) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVObjectValidationReport) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.Payload.Object.IsEmpty() || s.Payload.Validated.IsEmpty() {
		panic(throw.IllegalState())
	}

	return ctx.Jump(s.stepGetMemory)
}

func (s *SMVObjectValidationReport) stepGetMemory(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subSM := &statemachine.SMGetCachedMemory{
		Object: s.Payload.Object, State: s.Payload.Validated.GetLocal(),
	}
	return ctx.CallSubroutine(subSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		if subSM.Result == nil {
			panic(throw.IllegalState())
		}
		s.objDesc = subSM.Result
		return ctx.Jump(s.stepIncomingRequest)
	})
}

func (s *SMVObjectValidationReport) stepIncomingRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {

	return ctx.Stop()
}
