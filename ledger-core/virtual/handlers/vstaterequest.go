// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SMVStateRequest struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VStateRequest

	objectStateReport *payload.VStateReport
	stateAccessor     object.SharedStateAccessor

	// dependencies
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
	objectCatalog object.Catalog
}

/* -------- Declaration ------------- */

var dSMVStateRequestInstance smachine.StateMachineDeclaration = &dSMVStateRequest{}

type dSMVStateRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVStateRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMVStateRequest)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.objectCatalog)
}

func (*dSMVStateRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVStateRequest)
	return s.Init
}

/* -------- Instance ------------- */

func (*SMVStateRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SMVStateRequest)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
}

func (s *SMVStateRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVStateRequestInstance
}

func (s *SMVStateRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepCheckCatalog)
}

func (s *SMVStateRequest) stepCheckCatalog(ctx smachine.ExecutionContext) smachine.StateUpdate {
	objectSharedState, stateFound := s.objectCatalog.TryGet(ctx, s.Payload.Callee)
	if !stateFound {
		return ctx.Jump(s.stepBuildMissing)
	}

	s.stateAccessor = objectSharedState
	return ctx.Jump(s.stepBuildStateReport)
}

func (s *SMVStateRequest) stepBuildMissing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.objectStateReport = &payload.VStateReport{
		Status: payload.Missing,
		AsOf:   s.Payload.AsOf,
		Callee: s.Payload.Callee,
	}
	return ctx.Jump(s.stepSendResult)
}

func (s *SMVStateRequest) stepBuildStateReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		stateNotReady bool
	)

	action := func(state *object.SharedState) {
		if state.GetState() == object.Unknown {
			stateNotReady = true
			return
		}

		report := state.BuildStateReport()
		report.AsOf = s.Payload.AsOf

		s.objectStateReport = &report

		if s.Payload.RequestedContent.Contains(payload.RequestLatestDirtyState) {
			report.ProvidedContent.LatestDirtyState = state.BuildLatestDirtyState()
		}
	}

	switch s.stateAccessor.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.stateAccessor.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	if stateNotReady {
		ctx.Log().Trace(struct {
			*log.Msg  `txt:"State not ready for object"`
			Reference reference.Global
		}{
			Reference: s.Payload.Callee,
		})
		panic(throw.IllegalState())
	}

	return ctx.Jump(s.stepSendResult)
}

func (s *SMVStateRequest) stepSendResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	target := s.Meta.Sender

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendTarget(goCtx, s.objectStateReport, target)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Stop()
}
