// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var APICaller, _ = insolar.NewObjectReferenceFromString("insolar:0AAABAnRB0CKuqXTeTfQNTolmyixqQGMJz5sVvW81Dng")

type SMTestAPICall struct {
	requestPayload  payload.VCallRequest
	responsePayload payload.VCallResult

	response chan payload.VCallResult

	// injected arguments
	pulseSlot     *conveyor.PulseSlot
	messageSender *messageSenderAdapter.MessageSender
}

/* -------- Declaration ------------- */

var testAPICallSMDeclarationInstance smachine.StateMachineDeclaration = &dSMTestAPICall{}

type dSMTestAPICall struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMTestAPICall) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMTestAPICall)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
}

func (dSMTestAPICall) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMTestAPICall)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMTestAPICall) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return testAPICallSMDeclarationInstance
}

func (s *SMTestAPICall) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepSendRequest)
}

func (s *SMTestAPICall) stepSendRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	goCtx := ctx.GetContext()

	pulseNumber := s.pulseSlot.PulseData().PulseNumber

	var obj insolar.Reference
	switch s.requestPayload.CallType {
	case payload.CTMethod:
		obj = s.requestPayload.Callee

	case payload.CTConstructor:
		s.requestPayload.Caller = *APICaller
		s.requestPayload.CallOutgoing = gen.IDWithPulse(pulseNumber)
		obj = reference.NewGlobalSelf(s.requestPayload.CallOutgoing)

	default:
		panic(throw.IllegalValue())
	}

	bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*payload.VCallResult)
		if !ok || res == nil {
			panic(throw.IllegalValue())
		}
		s.responsePayload = *res

		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			return ctx.WakeUp()
		}
	})

	outgoingRef := reference.NewGlobal(*s.requestPayload.Caller.GetLocal(), s.requestPayload.CallOutgoing)

	if !ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bargeInCallback) {
		return ctx.Error(errors.New("failed to publish bargeInCallback"))
	}

	s.messageSender.PrepareNotify(ctx, func(svc messagesender.Service) {
		_ = svc.SendRole(goCtx, &s.requestPayload, insolar.DynamicRoleVirtualExecutor, obj, pulseNumber)
	}).Send()

	return ctx.Sleep().ThenJump(s.stepProcessResult)
}

func (s *SMTestAPICall) stepProcessResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	reChan := s.response

	go func() {
		reChan <- s.responsePayload
		close(s.response)
	}()

	return ctx.Stop()
}
