// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	authentication "github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SMVDelegatedCallRequest struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VDelegatedCallRequest

	objectSharedState object.SharedStateAccessor

	// dependencies
	objectCatalog         object.Catalog
	messageSender         messageSenderAdapter.MessageSender
	authenticationService authentication.Service
	pulseSlot             *conveyor.PulseSlot
}

var dSMVDelegatedCallRequestInstance smachine.StateMachineDeclaration = &dSMVDelegatedCallRequest{}

type dSMVDelegatedCallRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVDelegatedCallRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMVDelegatedCallRequest)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.objectCatalog)
	injector.MustInject(&s.authenticationService)
}

func (*dSMVDelegatedCallRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVDelegatedCallRequest)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVDelegatedCallRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVDelegatedCallRequestInstance
}

func (s *SMVDelegatedCallRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVDelegatedCallRequest) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	objectRef := s.Payload.Callee
	s.objectSharedState = s.objectCatalog.GetOrCreate(ctx, objectRef)

	return ctx.Jump(s.stepWaitObjectReady)
}

func (s *SMVDelegatedCallRequest) stepWaitObjectReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		semaphoreReadyToWork smachine.SyncLink
	)

	action := func(state *object.SharedState) {
		semaphoreReadyToWork = state.ReadyToWork
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.Passed:
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	default:
		panic(throw.Impossible())
	}

	if ctx.AcquireForThisStep(semaphoreReadyToWork).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepProcessRequest)
}

type delegationResult int

const (
	delegationOk delegationResult = iota
	delegationOldRequest
	delegationFullTable
)

func (s *SMVDelegatedCallRequest) stepProcessRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		resultCheck = delegationOk
	)

	action := func(state *object.SharedState) {
		var (
			oldestPulse  pulse.Number
			pendingList  *object.RequestList
			pendingCount uint8
		)

		callTolerance := s.Payload.CallFlags.GetInterference()

		switch callTolerance {
		case contract.CallTolerable:
			pendingList = state.PendingTable.GetList(contract.CallTolerable)
			oldestPulse = state.OrderedPendingEarliestPulse
			pendingCount = state.ActiveOrderedPendingCount
		case contract.CallIntolerable:
			pendingList = state.PendingTable.GetList(contract.CallIntolerable)
			oldestPulse = state.UnorderedPendingEarliestPulse
			pendingCount = state.ActiveUnorderedPendingCount
		default:
			panic(throw.Unsupported())
		}

		if oldestPulse == pulse.Unknown || s.Payload.CallOutgoing.GetLocal().GetPulseNumber() < oldestPulse {
			resultCheck = delegationOldRequest
			return
		}

		// pendingList already full
		if pendingList.Count() == int(pendingCount) && !pendingList.Exist(s.Payload.CallOutgoing) {
			fmt.Println(pendingList.Count(), int(pendingCount), pendingList.Exist(s.Payload.CallOutgoing))
			resultCheck = delegationFullTable
			return
		}

		if !pendingList.Add(s.Payload.CallOutgoing) {
			ctx.Log().Trace(struct {
				*log.Msg  `txt:"request already in pending list"`
				Reference string
			}{Reference: s.Payload.CallOutgoing.String()})
			// request was already in the list we will delegate token, maybe it is repeated call
			return
		}

		if pendingList.Count() == int(pendingCount) {
			state.SetPendingListFilled(ctx, callTolerance)
		}
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.Passed:
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	default:
		panic(throw.Impossible())
	}

	switch resultCheck {
	case delegationOldRequest:
		return ctx.Error(throw.E("There is no such object", struct{ Reference string }{Reference: s.Payload.CallIncoming.String()}))
	case delegationFullTable:
		return ctx.Error(throw.E("Pending table already full", struct {
			Reference string
		}{Reference: s.Payload.CallIncoming.String()}))
		// Ok case
	case delegationOk:
	default:
		panic(throw.IllegalState())
	}

	return ctx.Jump(s.stepBuildResponse)
}

func (s *SMVDelegatedCallRequest) stepBuildResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	target := s.Meta.Sender

	token := s.authenticationService.GetCallDelegationToken(s.Payload.CallOutgoing, s.Meta.Sender, s.pulseSlot.PulseData().PulseNumber, s.Payload.Callee)

	response := payload.VDelegatedCallResponse{
		Callee:         s.Payload.Callee,
		CallIncoming:   s.Payload.CallIncoming,
		DelegationSpec: token,
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendTarget(goCtx, &response, target)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error(struct {
					*log.Msg `txt:"failed to send VDelegatedCallResponse"`
					Object   string
					Request  string
					target   string
				}{Object: s.Payload.Callee.String(), Request: s.Payload.String(), target: target.String()}, err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Stop()
}
