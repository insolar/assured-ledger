// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
)

type SMVStateRequest struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VStateRequest

	objectStateReport *payload.VStateReport

	failReason payload.VStateUnavailable_ReasonType

	messageSender *messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
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
	return ctx.Jump(s.stepProcess)
}

func (s *SMVStateRequest) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		objectCatalog = object.Catalog{}
	)
	objectSharedState, stateFound := objectCatalog.TryGet(ctx, s.Payload.Callee)
	if !stateFound {
		s.failReason = payload.Missing
		return ctx.Jump(s.stepReturnStateUnavailable)
	}

	var (
		stateNotReady bool
	)

	action := func(state *object.SharedState) {
		if !state.IsReady() {
			stateNotReady = true
			return
		}

		objectState := state.GetState()
		switch objectState {
		case object.Missing:
			s.failReason = payload.Missing
			return
		case object.Inactive:
			s.failReason = payload.Inactive
			return
		case object.HasState:
		// ok case
		default:
			panic(throw.NotImplemented())
		}

		descriptor := state.Descriptor()

		stateRef := descriptor.HeadRef()

		immutableCounts, _ := state.ImmutableExecute.GetCounts()
		mutableCounts, _ := state.MutableExecute.GetCounts()

		s.objectStateReport = &payload.VStateReport{
			AsOf:                  s.Payload.AsOf,
			Callee:                s.Payload.Callee,
			LatestDirtyState:      *stateRef,
			ImmutablePendingCount: int32(immutableCounts),
			MutablePendingCount:   int32(mutableCounts),
		}

		if s.Payload.RequestedContent.Contains(payload.RequestLatestDirtyState) {
			parent := descriptor.Parent()
			proto, _ := descriptor.Prototype()

			memory := descriptor.Memory()

			s.objectStateReport.ProvidedContent = &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference:   *stateRef,
					Parent:      *parent,
					State:       memory,
					Deactivated: state.Deactivated,
				},
			}
			if proto != nil {
				s.objectStateReport.ProvidedContent.LatestDirtyState.Prototype = *proto
			}
		}
	}

	switch objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		ctx.Log().Fatal("failed to get object state: already dead")
	case smachine.Passed:
		// go further
	default:
		panic(throw.NotImplemented())
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

	if s.failReason > 0 {
		return ctx.Jump(s.stepReturnStateUnavailable)
	}

	return ctx.Jump(s.stepSendResult)
}

func (s *SMVStateRequest) stepReturnStateUnavailable(ctx smachine.ExecutionContext) smachine.StateUpdate {
	msg := &payload.VStateUnavailable{
		Reason:   s.failReason,
		Lifeline: s.Payload.Callee,
	}

	target := s.Meta.Sender

	goCtx := ctx.GetContext()
	s.messageSender.PrepareNotify(ctx, func(svc messagesender.Service) {
		err := svc.SendTarget(goCtx, msg, target)
		if err != nil {
			ctx.Log().Warn(throw.W(err, "failed to send state"))
		}
	}).Send()

	return ctx.Stop()
}

func (s *SMVStateRequest) stepSendResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	target := s.Meta.Sender

	goCtx := ctx.GetContext()
	s.messageSender.PrepareNotify(ctx, func(svc messagesender.Service) {
		err := svc.SendTarget(goCtx, s.objectStateReport, target)
		if err != nil {
			ctx.Log().Warn(throw.W(err, "failed to send state"))
		}
	}).Send()

	return ctx.Stop()
}
