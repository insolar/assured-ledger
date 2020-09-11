// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
)

type SMVFindCallRequest struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VFindCallRequest

	// dependencies
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot

	syncLinkAccessor callsummary.SyncAccessor

	callResult *rms.VCallResult
	status     rms.VFindCallResponse_CallState
}

/* -------- Declaration ------------- */

var dSMVFindCallRequestInstance smachine.StateMachineDeclaration = &dSMVFindCallRequest{}

type dSMVFindCallRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVFindCallRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVFindCallRequest)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
}

func (*dSMVFindCallRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVFindCallRequest)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVFindCallRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVFindCallRequestInstance
}

func (s *SMVFindCallRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if s.pulseSlot.State() == conveyor.Present {
		return ctx.JumpExt(smachine.SlotStep{
			Transition: s.stepWait,
			Migration:  s.migrateFutureMessage,
		})
	}

	return ctx.Jump(s.stepProcessRequest)
}

func (s *SMVFindCallRequest) stepWait(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

func (s *SMVFindCallRequest) migrateFutureMessage(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcessRequest)
}

func (s *SMVFindCallRequest) stepProcessRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	accessor, isCondPublished := callsummary.GetSummarySMSyncAccessor(ctx, s.Payload.Callee.GetValue())
	if isCondPublished {
		// if cond is still published we need to wait for object to register itself in call summary
		s.syncLinkAccessor = accessor
		return ctx.Jump(s.stepWaitCallResult)
	}
	return ctx.Jump(s.stepGetRequestData)
}

func (s *SMVFindCallRequest) stepWaitCallResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var cond smachine.SyncLink

	switch s.syncLinkAccessor.Prepare(func(link *smachine.SyncLink) {
		cond = *link
	}).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.syncLinkAccessor.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	if ctx.Acquire(cond).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepGetRequestData)
}

func (s *SMVFindCallRequest) stepGetRequestData(ctx smachine.ExecutionContext) smachine.StateUpdate {
	summarySharedStateAccessor, ok := callsummary.GetSummarySMSharedAccessor(ctx, s.pulseSlot.PulseData().PulseNumber)
	// no result
	if !ok {
		return ctx.Jump(s.stepNotFoundResponse)
	}

	var (
		foundResult bool
	)
	action := func(sharedCallSummary *callsummary.SharedCallSummary) {
		requests, ok := sharedCallSummary.Requests.GetObjectCallResults(s.Payload.Callee.GetValue())
		if !ok {
			return
		}

		objectCallResult, ok := requests.CallResults[s.Payload.Outgoing.GetValue()]
		if !ok {
			return
		}

		foundResult = true

		s.callResult = objectCallResult.Result
	}

	switch summarySharedStateAccessor.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(summarySharedStateAccessor.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	if !foundResult {
		return ctx.Jump(s.stepNotFoundResponse)
	}

	s.status = rms.CallStateFound

	return ctx.Jump(s.stepSendResponse)
}

func (s *SMVFindCallRequest) stepNotFoundResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.status = rms.CallStateMissing
	if s.Payload.Outgoing.GetPulseOfLocal() < s.Payload.LookAt {
		s.status = rms.CallStateUnknown
	}
	return ctx.Jump(s.stepSendResponse)
}

func (s *SMVFindCallRequest) stepSendResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	target := s.Meta.Sender

	msg := rms.VFindCallResponse{
		LookedAt:   s.Payload.LookAt,
		Callee:     s.Payload.Callee,
		Outgoing:   s.Payload.Outgoing,
		Status:     s.status,
		CallResult: s.callResult,
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendTarget(goCtx, &msg, target)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Stop()
}
