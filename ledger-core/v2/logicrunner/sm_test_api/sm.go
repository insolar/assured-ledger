// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_test_api

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_sender"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type TestApiCallSM struct {
	requestPayload  payload.VCallRequest
	responsePayload payload.VCallResult

	response chan payload.VCallResult

	sender *s_sender.SenderServiceAdapter
}

type TestApiCallSMDeclaration struct {
	smachine.StateMachineDeclTemplate
}

var ApiCaller, _ = insolar.NewObjectReferenceFromString("insolar:0AAABAnRB0CKuqXTeTfQNTolmyixqQGMJz5sVvW81Dng")

func (TestApiCallSMDeclaration) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*TestApiCallSM)
	return s.Init
}

func (*TestApiCallSMDeclaration) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*TestApiCallSM)

	injector.MustInject(&s.sender)
}

var testApiCallSMDeclarationInstance smachine.StateMachineDeclaration = &TestApiCallSMDeclaration{}

func (s *TestApiCallSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return testApiCallSMDeclarationInstance
}

func (s *TestApiCallSM) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepSendRequest)
}

func (s *TestApiCallSM) stepSendRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
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
		s.requestPayload.Caller = *ApiCaller
		s.requestPayload.CallOutgoing = gen.ID()
		obj = *insolar.NewGlobalReference(s.requestPayload.CallOutgoing, s.requestPayload.CallOutgoing)
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

	outgoingRef := *insolar.NewGlobalReference(*s.requestPayload.Caller.GetLocal(), s.requestPayload.CallOutgoing)
	ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bgin)

	s.sender.PrepareNotify(ctx, func(svc s_sender.SenderService) {
		svc.SendRole(goCtx, msg, insolar.DynamicRoleVirtualExecutor, obj)
	}).Send()

	return ctx.Sleep().ThenJump(s.stepProcessResult)
}

func (s *TestApiCallSM) stepProcessResult(ctx smachine.ExecutionContext) smachine.StateUpdate {

	reChan := s.response
	go func() {
		reChan <- s.responsePayload
		close(s.response)
	}()

	return ctx.Stop()
}
