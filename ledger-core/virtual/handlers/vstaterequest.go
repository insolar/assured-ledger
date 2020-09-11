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
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object/preservedstatereport"
)

type SMVStateRequest struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VStateRequest

	objectStateReport *rms.VStateReport
	reportAccessor    preservedstatereport.SharedReportAccessor

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

func (*dSMVStateRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
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

func (s *SMVStateRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVStateRequestInstance
}

func (s *SMVStateRequest) migrateFutureMessage(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(func(ctx smachine.MigrationContext) smachine.StateUpdate {
		return ctx.Stop()
	})
	return ctx.Jump(s.stepCheckCatalog)
}

func (s *SMVStateRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if s.pulseSlot.State() == conveyor.Present {
		ctx.SetDefaultMigration(s.migrateFutureMessage)
		return ctx.Jump(s.stepWait)
	}

	return ctx.Jump(s.stepCheckCatalog)
}

func (s *SMVStateRequest) stepWait(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

func (s *SMVStateRequest) stepCheckCatalog(ctx smachine.ExecutionContext) smachine.StateUpdate {
	reportSharedState, stateFound := preservedstatereport.GetSharedStateReport(ctx, s.Payload.Object.GetValue())

	if !stateFound {
		return ctx.Jump(s.stepBuildMissing)
	}

	s.reportAccessor = reportSharedState
	return ctx.Jump(s.stepBuildStateReport)
}

func (s *SMVStateRequest) stepBuildMissing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.objectStateReport = &rms.VStateReport{
		Status: rms.StateStatusMissing,
		AsOf:   s.Payload.AsOf,
		Object: s.Payload.Object,
	}
	return ctx.Jump(s.stepSendResult)
}

func (s *SMVStateRequest) stepBuildStateReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		response rms.VStateReport
		content  *rms.VStateReport_ProvidedContentBody
	)

	action := func(report rms.VStateReport) {
		response = report
		content = report.ProvidedContent
	}

	switch s.reportAccessor.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.reportAccessor.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	response.ProvidedContent = nil
	if s.Payload.RequestedContent != 0 && content != nil {
		response.ProvidedContent = &rms.VStateReport_ProvidedContentBody{}
		if s.Payload.RequestedContent.Contains(rms.RequestLatestDirtyState) {
			response.ProvidedContent.LatestDirtyState = content.LatestDirtyState
		}

		if s.Payload.RequestedContent.Contains(rms.RequestLatestValidatedState) {
			response.ProvidedContent.LatestValidatedState = content.LatestValidatedState
		}
	}

	response.AsOf = s.Payload.AsOf

	s.objectStateReport = &response

	return ctx.Jump(s.stepSendResult)
}

func (s *SMVStateRequest) stepSendResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	target := s.Meta.Sender.GetValue()

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
