// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package statemachine

import (
	"context"
	"errors"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var APICaller, _ = reference.GlobalObjectFromString("insolar:0AAABAnRB0CKuqXTeTfQNTolmyixqQGMJz5sVvW81Dng")

type SMTestAPICall struct {
	requestPayload  payload.VCallRequest
	responsePayload payload.VCallResult

	messageAlreadySent bool

	// injected arguments
	pulseSlot     *conveyor.PulseSlot
	messageSender messageSenderAdapter.MessageSender
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

func (s *SMTestAPICall) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepSendRequest)
}

func (s *SMTestAPICall) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(s.migrationDefault)

	s.requestPayload.Caller = APICaller
	s.requestPayload.CallOutgoing = gen.UniqueIDWithPulse(s.pulseSlot.PulseData().PulseNumber)

	return ctx.Jump(s.stepRegisterBargeIn)
}

func (s *SMTestAPICall) stepRegisterBargeIn(ctx smachine.ExecutionContext) smachine.StateUpdate {
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

	outgoingRef := reference.NewRecordOf(s.requestPayload.Caller, s.requestPayload.CallOutgoing)
	if !ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bargeInCallback) {
		return ctx.Error(errors.New("failed to publish bargeInCallback"))
	}
	return ctx.Jump(s.stepSendRequest)
}

func (s *SMTestAPICall) stepSendRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var obj reference.Global
	switch s.requestPayload.CallType {
	case payload.CTMethod:
		obj = s.requestPayload.Callee
	case payload.CTConstructor:
		obj = reference.NewSelf(s.requestPayload.CallOutgoing)
	default:
		panic(throw.IllegalValue())
	}

	payloadData := s.requestPayload
	if s.messageAlreadySent {
		payloadData.CallRequestFlags.WithRepeatedCall(payload.RepeatedCall)
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &payloadData, node.DynamicRoleVirtualExecutor, obj, s.pulseSlot.CurrentPulseNumber())
		s.messageAlreadySent = true
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
				return
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Sleep().ThenJump(s.stepProcessResult)
}

func (s *SMTestAPICall) stepProcessResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.SetDefaultTerminationResult(s.responsePayload)

	return ctx.Stop()
}
