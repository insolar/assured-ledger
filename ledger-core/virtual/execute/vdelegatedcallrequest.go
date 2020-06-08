// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

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

type SMVDelegatedCallRequest struct {
	// input arguments
	Meta           *payload.Meta
	RequestPayload payload.VDelegatedCallRequest

	response *payload.VDelegatedCallResponse

	// dependencies
	pulseSlot     *conveyor.PulseSlot
	messageSender messageSenderAdapter.MessageSender
}

/* -------- Declaration ------------- */

var dSMVDelegatedCallResultInstance smachine.StateMachineDeclaration = &dSMVDelegatedCallRequest{}

type dSMVDelegatedCallRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVDelegatedCallRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMVDelegatedCallRequest)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
}

func (*dSMVDelegatedCallRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return nil
}

/* -------- Instance ------------- */

func (s *SMVDelegatedCallRequest) GetSubroutineInitState(_ smachine.SubroutineStartContext) smachine.InitFunc {
	return s.Init
}

func (s *SMVDelegatedCallRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVDelegatedCallResultInstance
}

func (s *SMVDelegatedCallRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepRegisterBargeIn)
}

func (s *SMVDelegatedCallRequest) stepRegisterBargeIn(ctx smachine.ExecutionContext) smachine.StateUpdate {
	bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*payload.VDelegatedCallResponse)
		if !ok || res == nil {
			panic(throw.IllegalValue())
		}
		s.response = res

		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			return ctx.WakeUp()
		}
	})

	callOutgoing := gen.UniqueIDWithPulse(s.pulseSlot.PulseData().PulseNumber)
	outgoingRef := reference.NewRecordOf(s.RequestPayload.Callee, callOutgoing)
	if !ctx.PublishGlobalAliasAndBargeIn(outgoingRef, bargeInCallback) {
		return ctx.Error(errors.New("failed to publish bargeInCallback"))
	}
	s.RequestPayload.RefIn = outgoingRef
	return ctx.Jump(s.stepSendRequest)
}

func (s *SMVDelegatedCallRequest) stepSendRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &s.RequestPayload, node.DynamicRoleVirtualExecutor, s.RequestPayload.Callee, s.pulseSlot.PulseData().PulseNumber)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
				return
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Sleep().ThenJump(s.stepProcessResult)
}

func (s *SMVDelegatedCallRequest) stepProcessResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.response == nil {
		ctx.Sleep().ThenRepeat()
	}

	return ctx.Stop()
}
