// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
//go:generate sm-uml-gen -f $GOFILE
package handlers

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

type SMVObjectValidationReport struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VObjectValidationReport

	// dependencies
	pulseSlot     *conveyor.PulseSlot
	messageSender messageSenderAdapter.MessageSender
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
	msg := payload.VCachedMemoryRequest{
		Object:  s.Payload.Object,
		StateID: s.Payload.Validated.GetLocal(),
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, affinity.DynamicRoleVirtualValidator, s.Payload.Object, s.pulseSlot.CurrentPulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()
	return ctx.Stop()
}
