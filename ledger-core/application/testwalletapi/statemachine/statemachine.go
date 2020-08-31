// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package statemachine

import (
	"context"
	"errors"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
)

const BuiltinTestAPIEcho = "insolar:0AAABApiTestEcho____"
const BuiltinTestAPIBriefEcho = "insolar:0AAABrief.0AAABApiTestEcho____"

var builtinTestAPIEchoRef, _ = reference.GlobalFromString(BuiltinTestAPIEcho)
var builtinTestAPIEchoBriefRef, _ = reference.GlobalFromString(BuiltinTestAPIBriefEcho)

var APICaller, _ = reference.GlobalObjectFromString("insolar:0AAABAnRB0CKuqXTeTfQNTolmyixqQGMJz5sVvW81Dng")

const MaxRepeats = 3

type SMTestAPICall struct {
	requestPayload  payload.VCallRequest
	responsePayload []byte

	object           reference.Global
	messageSentTimes atomickit.Uint32

	// injected arguments
	pulseSlot     *conveyor.PulseSlot
	messageSender messageSenderAdapter.MessageSender
	memoryCache   memoryCacheAdapter.MemoryCache
}

/* -------- Declaration ------------- */

var testAPICallSMDeclarationInstance smachine.StateMachineDeclaration = &dSMTestAPICall{}

type dSMTestAPICall struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMTestAPICall) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMTestAPICall)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.memoryCache)
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
	return ctx.Jump(s.stepSend)
}

func (s *SMTestAPICall) stepSend(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.requestPayload.Caller = APICaller
	outLocal := gen.UniqueLocalRefWithPulse(s.pulseSlot.CurrentPulseNumber())
	s.requestPayload.CallOutgoing = reference.NewRecordOf(APICaller, outLocal)

	switch s.requestPayload.CallType {
	case payload.CallTypeMethod:
		s.object = s.requestPayload.Callee
	case payload.CallTypeConstructor:
		s.object = reference.NewSelf(outLocal)
	default:
		panic(throw.IllegalValue())
	}

	bargeIn := s.newBargeIn(ctx)

	if !ctx.PublishGlobalAliasAndBargeIn(s.requestPayload.CallOutgoing, bargeIn) {
		return ctx.Error(errors.New("failed to publish bargeInCallback"))
	}

	s.sendRequest(ctx)

	ctx.SetDefaultMigration(s.migrateResend)

	return ctx.Jump(s.stepProcessResult)
}

func (s *SMTestAPICall) stepResend(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.sendRequest(ctx)
	return ctx.Jump(s.stepProcessResult)
}

func (s *SMTestAPICall) migrateResend(ctx smachine.MigrationContext) smachine.StateUpdate {
	if s.messageSentTimes.Load() >= MaxRepeats {
		res, err := foundation.MarshalMethodErrorResult(throw.New("timeout: exceeded resend limit"))
		if err != nil {
			panic(throw.W(err, "couldn't marshal error"))
		}
		s.responsePayload = res

		return ctx.Jump(s.stepProcessResult)
	}
	return ctx.Jump(s.stepResend)
}

func (s *SMTestAPICall) stepProcessResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.responsePayload == nil {
		return ctx.Sleep().ThenRepeat()
	}

	ctx.SetDefaultTerminationResult(s.responsePayload)
	return ctx.Stop()
}

func (s *SMTestAPICall) newBargeIn(ctx smachine.ExecutionContext) smachine.BargeInWithParam {
	return ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*payload.VCallResult)
		if !ok || res == nil {
			panic(throw.IllegalValue())
		}

		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			s.responsePayload = res.ReturnArguments

			return ctx.WakeUp()
		}
	})
}

func (s *SMTestAPICall) sendRequest(ctx smachine.ExecutionContext) {
	payloadData := s.requestPayload

	if s.messageSentTimes.Load() > 0 {
		payloadData.CallRequestFlags.WithRepeatedCall(payload.RepeatedCall)
	}

	if s.object.GetBase().Equal(builtinTestAPIEchoRef.GetBase()) {
		s.sendEchoRequest(ctx)
		return
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &payloadData, affinity.DynamicRoleVirtualExecutor, s.object, s.pulseSlot.CurrentPulseNumber())
		s.messageSentTimes.Add(1)

		if err == nil {
			return nil
		}
		return func(ctx smachine.AsyncResultContext) {
			ctx.Log().Error("failed to send message", err)
		}
	}).WithoutAutoWakeUp().Start()
}

func (s *SMTestAPICall) sendEchoRequest(ctx smachine.ExecutionContext) {
	if !s.object.Equal(builtinTestAPIEchoRef) {
		s.responsePayload = s.requestPayload.Arguments
		return
	}

	_, bargeIn := ctx.GetPublishedGlobalAliasAndBargeIn(s.requestPayload.CallOutgoing)

	result := &payload.VCallResult{
		ReturnArguments: s.requestPayload.Arguments,
	}

	sendFn := func() bool {
		return bargeIn.CallWithParam(result)
	}

	s.messageSender.PrepareAsync(ctx, func(context.Context, messagesender.Service) smachine.AsyncResultFunc {
		ok := sendFn()
		s.messageSentTimes.Add(1)

		if ok {
			return nil
		}
		return func(ctx smachine.AsyncResultContext) {
			ctx.Log().Warn("barge-in was not found for BuiltinTestAPIEcho")
		}
	}).WithoutAutoWakeUp().Start()
}
