// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_test_api // nolint:golint

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_sender"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type TestAPICallSM struct {
	requestPayload  payload.VCallRequest
	responsePayload payload.VCallResult

	response chan payload.VCallResult

	// injected arguments
	PulseSlot *conveyor.PulseSlot
	sender    *s_sender.SenderServiceAdapter
}

type TestAPICallSMDeclaration struct {
	smachine.StateMachineDeclTemplate
}

var APICaller, _ = insolar.NewObjectReferenceFromString("insolar:0AAABAnRB0CKuqXTeTfQNTolmyixqQGMJz5sVvW81Dng")

func (TestAPICallSMDeclaration) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*TestAPICallSM)
	return s.Init
}

func (*TestAPICallSMDeclaration) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*TestAPICallSM)

	injector.MustInject(&s.sender)
}

var testAPICallSMDeclarationInstance smachine.StateMachineDeclaration = &TestAPICallSMDeclaration{}

func (s *TestAPICallSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return testAPICallSMDeclarationInstance
}

func (s *TestAPICallSM) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepSendRequest)
}

func (s *TestAPICallSM) stepSendRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	goCtx := ctx.GetContext()

	msg, err := payload.NewMessage(&s.requestPayload)
	if err != nil {
		panic("couldn't serialize messagePayload: " + err.Error())
	}

	var obj insolar.Reference
	switch s.requestPayload.CallType {
	case payload.CTMethod:
		obj = s.requestPayload.Callee
	case payload.CTConstructor:
		s.requestPayload.Caller = *APICaller
		s.requestPayload.CallOutgoing = gen.IDWithPulse(s.PulseSlot.PulseData().PulseNumber)
		obj = reference.NewGlobalSelf(s.requestPayload.CallOutgoing)
	default:
		panic(throw.IllegalValue())
	}

	bgin := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {

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
	ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bgin)

	s.sender.PrepareNotify(ctx, func(svc s_sender.SenderService) {
		svc.SendRole(goCtx, msg, insolar.DynamicRoleVirtualExecutor, obj)
	}).Send()

	return ctx.Sleep().ThenJump(s.stepProcessResult)
}

func (s *TestAPICallSM) stepProcessResult(ctx smachine.ExecutionContext) smachine.StateUpdate {

	reChan := s.response
	go func() {
		reChan <- s.responsePayload
		close(s.response)
	}()

	return ctx.Stop()
}
