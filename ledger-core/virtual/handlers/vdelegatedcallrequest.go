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
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
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

func (*dSMVDelegatedCallRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
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
	if s.pulseSlot.State() != conveyor.Present {
		ctx.Log().Trace("stop processing VDelegatedCallRequest since we are not in present pulse")
		return ctx.Stop()
	}

	ctx.SetDefaultMigration(s.migrationDefault)
	return ctx.Jump(s.stepProcess)
}

func (s *SMVDelegatedCallRequest) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.Log().Trace("stop processing VDelegatedCallRequest since pulse was changed")
	return ctx.Stop()
}

func (s *SMVDelegatedCallRequest) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.objectSharedState = s.objectCatalog.GetOrCreate(ctx, s.Payload.Callee.GetValue())

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
			oldestPulse                  pulse.Number
			pendingList                  *callregistry.PendingList
			previousExecutorPendingCount int
		)

		callTolerance := s.Payload.CallFlags.GetInterference()

		switch callTolerance {
		case isolation.CallTolerable:
			pendingList = state.PendingTable.GetList(isolation.CallTolerable)
			oldestPulse = state.OrderedPendingEarliestPulse
			previousExecutorPendingCount = int(state.PreviousExecutorOrderedPendingCount)
		case isolation.CallIntolerable:
			pendingList = state.PendingTable.GetList(isolation.CallIntolerable)
			oldestPulse = state.UnorderedPendingEarliestPulse
			previousExecutorPendingCount = int(state.PreviousExecutorUnorderedPendingCount)
		default:
			panic(throw.Unsupported())
		}

		if oldestPulse == pulse.Unknown || s.Payload.CallOutgoing.GetPulseOfLocal() < oldestPulse {
			resultCheck = delegationOldRequest
			return
		}

		// pendingList already full
		callOutgoing := s.Payload.CallOutgoing.GetValue()
		if pendingList.Count() == previousExecutorPendingCount && !pendingList.Exist(callOutgoing) {
			resultCheck = delegationFullTable
			return
		}

		if !pendingList.Add(callOutgoing) {
			ctx.Log().Trace(struct {
				*log.Msg  `txt:"request already in pending list"`
				Reference reference.Global
			}{Reference: callOutgoing})
			// request was already in the list we will delegate token, maybe it is repeated call
			return
		}
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.Passed:
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	default:
		panic(throw.Impossible())
	}

	// TODO: sure about that? callIncoming here, but in action callOutgoing is used
	callIncoming := s.Payload.CallIncoming.GetValue()
	switch resultCheck {
	case delegationOldRequest:
		return ctx.Error(throw.E("There is no such object", struct {
			Reference reference.Global
		}{Reference: callIncoming}))
	case delegationFullTable:
		return ctx.Error(throw.E("Pending table already full", struct {
			Reference reference.Global
		}{Reference: callIncoming}))
		// Ok case
	case delegationOk:
	default:
		panic(throw.IllegalState())
	}

	return ctx.Jump(s.stepBuildResponse)
}

func (s *SMVDelegatedCallRequest) stepBuildResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		delegateTo   = s.Meta.Sender
		callee       = s.Payload.Callee.GetValue()
		pn           = s.pulseSlot.PulseNumber()
		callOutgoing = s.Payload.CallOutgoing.GetValue()
	)

	token := s.authenticationService.GetCallDelegationToken(callOutgoing, delegateTo, pn, callee)

	response := payload.VDelegatedCallResponse{
		Callee:                 s.Payload.Callee,
		CallIncoming:           s.Payload.CallIncoming,
		ResponseDelegationSpec: token,
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendTarget(goCtx, &response, delegateTo)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error(struct {
					*log.Msg   `txt:"failed to send VDelegatedCallResponse"`
					Object     reference.Global
					DelegateTo reference.Global
					Request    string
				}{
					Object:     callee,
					DelegateTo: delegateTo,
					Request:    s.Payload.String(),
				}, err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Stop()
}
