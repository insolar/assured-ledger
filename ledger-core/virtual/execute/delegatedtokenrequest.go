// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"context"
	"errors"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type DelegationTokenAwaitKey struct {
	Outgoing reference.Global
}

type SMDelegatedTokenRequest struct {
	// input arguments
	Meta           *payload.Meta
	RequestPayload payload.VDelegatedCallRequest

	response *payload.VDelegatedCallResponse

	// dependencies
	pulseSlot     *conveyor.PulseSlot
	messageSender messageSenderAdapter.MessageSender
}

/* -------- Declaration ------------- */

var dSMDelegatedTokenRequestInstance smachine.StateMachineDeclaration = &dSMDelegatedTokenRequest{}

type dSMDelegatedTokenRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMDelegatedTokenRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMDelegatedTokenRequest)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
}

func (*dSMDelegatedTokenRequest) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return nil
}

/* -------- Instance ------------- */

func (s *SMDelegatedTokenRequest) GetSubroutineInitState(ctx smachine.SubroutineStartContext) smachine.InitFunc {
	ctx.SetSubroutineCleanupMode(smachine.SubroutineCleanupAliasesAndShares)

	return s.Init
}

func (s *SMDelegatedTokenRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMDelegatedTokenRequestInstance
}

func (s *SMDelegatedTokenRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(s.migration)
	return ctx.Jump(s.stepRegisterBargeIn)
}

func (s *SMDelegatedTokenRequest) stepRegisterBargeIn(ctx smachine.ExecutionContext) smachine.StateUpdate {
	bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*payload.VDelegatedCallResponse)
		if !ok || res == nil {
			panic(throw.IllegalValue())
		}

		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			s.response = res

			return ctx.WakeUp()
		}
	})

	key := DelegationTokenAwaitKey{s.RequestPayload.CallOutgoing.GetValue()}

	if !ctx.PublishGlobalAliasAndBargeIn(key, bargeInCallback) {
		return ctx.Error(errors.New("failed to publish bargeIn callback"))
	}
	return ctx.Jump(s.stepSendRequest)
}

func (s *SMDelegatedTokenRequest) stepSendRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		pl = s.RequestPayload
		pn = s.pulseSlot.CurrentPulseNumber()
	)

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &pl, affinity.DynamicRoleVirtualExecutor, pl.Callee.GetValue(), pn)

		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
				return
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Sleep().ThenJump(s.stepProcessResult)
}

func (s *SMDelegatedTokenRequest) stepProcessResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.response == nil {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Stop()
}

func (s *SMDelegatedTokenRequest) migration(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}
