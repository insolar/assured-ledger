// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_request // nolint:golint

import (
	"time"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/reply"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_artifact"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_sender"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/sm_execute_request"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/sm_object"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type StateMachineCallMethod struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.CallMethod

	// injected arguments
	catalogObj     *sm_object.LocalObjectCatalog
	artifactClient *s_artifact.ArtifactClientServiceAdapter
	sender         *s_sender.SenderServiceAdapter
	pulseSlot      *conveyor.PulseSlot

	externalError error // error that is returned from ledger

	requestInfo *common.ParsedRequestInfo
}

/* -------- Declaration ------------- */

var declCallMethod smachine.StateMachineDeclaration = &declarationCallMethod{}

type declarationCallMethod struct {
	smachine.StateMachineDeclTemplate
}

func (*declarationCallMethod) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*StateMachineCallMethod)

	injector.MustInject(&s.catalogObj)
	injector.MustInject(&s.artifactClient)
	injector.MustInject(&s.sender)
	injector.MustInject(&s.pulseSlot)
}

func (*declarationCallMethod) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachineCallMethod)
	return s.Init
}

/* -------- Instance ------------- */

func (s *StateMachineCallMethod) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declCallMethod
}

func (s *StateMachineCallMethod) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepRegisterIncoming)
}

func (s *StateMachineCallMethod) stepRegisterIncoming(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		goCtx       = ctx.GetContext()
		asyncLogger = ctx.LogAsync()

		incoming = s.Payload.Request
	)

	var externalError error

	return s.artifactClient.PrepareAsync(ctx, func(svc s_artifact.ArtifactClientService) smachine.AsyncResultFunc {
		defer common.LogAsyncTime(asyncLogger, time.Now(), "RegisterIncoming")

		info, err := svc.RegisterIncomingRequest(goCtx, incoming)

		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				externalError = err
				return
			}

			s.requestInfo, externalError = common.NewParsedRequestInfo(incoming, info)
			if externalError != nil {
				//
			} else if _, ok := s.requestInfo.Request.(*record.IncomingRequest); !ok {
				externalError = errors.Errorf("unexpected request type: %T", s.requestInfo.Request)
			}
		}
	}).DelayedStart().Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if externalError != nil {
			return ctx.Error(externalError)
		}
		return ctx.Jump(s.stepSendRequestID)
	})
}

func (s *StateMachineCallMethod) stepSendRequestID(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.externalError != nil {
		return ctx.Jump(s.stepError)
	}

	var (
		messageMeta = s.Meta
		response    = &reply.RegisterRequest{Request: s.requestInfo.RequestReference}
		goCtx       = ctx.GetContext()
	)

	s.sender.PrepareNotify(ctx, func(svc s_sender.SenderService) {
		msg := bus.ReplyAsMessage(goCtx, response)
		svc.Reply(goCtx, *messageMeta, msg)
	}).Send()

	if s.pulseSlot.State() == conveyor.Antique {
		// pulse has changed, send message
		return ctx.Jump(s.stepSendRegisteredCall)
	}
	return ctx.Jump(s.stepExecute)
}

func (s *StateMachineCallMethod) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(nil)

	var (
		requestInfo = s.requestInfo
		meta        = s.Meta
		incoming    = requestInfo.Request.(*record.IncomingRequest)
		traceID     = inslogger.TraceID(ctx.GetContext())
	)

	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		ctx.SetTracerID(traceID)

		return &sm_execute_request.ExecuteIncomingRequest{
			ExecuteIncomingCommon: &sm_execute_request.ExecuteIncomingCommon{
				SharedRequestState: sm_execute_request.SharedRequestState{
					RequestInfo: requestInfo,
					Nonce:       incoming.Nonce,
				},
				MessageMeta: meta,
			},
		}
	})
}

func (s *StateMachineCallMethod) stepDone(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (s *StateMachineCallMethod) stepError(ctx smachine.ExecutionContext) smachine.StateUpdate {
	err := s.externalError
	messageMeta := s.Meta
	goCtx := ctx.GetContext()

	s.sender.PrepareNotify(ctx, func(svc s_sender.SenderService) {
		bus.ReplyError(goCtx, svc, *messageMeta, err)
	}).Send()

	return ctx.Error(s.externalError)
}

/* -------- Migration ------------- */

// nolint:unused
func (s *StateMachineCallMethod) migrationPulseChanged(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(nil)

	return ctx.Jump(s.stepPulseChanged)
}

// nolint:unused
func (s *StateMachineCallMethod) stepPulseChanged(ctx smachine.ExecutionContext) smachine.StateUpdate {
	messageMeta := s.Meta
	response := &reply.Error{ErrType: reply.FlowCancelled}
	goCtx := ctx.GetContext()

	s.sender.PrepareNotify(ctx, func(svc s_sender.SenderService) {
		msg := bus.ReplyAsMessage(goCtx, response)
		svc.Reply(goCtx, *messageMeta, msg)
	}).Send()

	return ctx.Jump(s.stepDone)
}

// nolint:unused
func (s *StateMachineCallMethod) migrationSendRegisteredCall(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(nil)

	return ctx.Jump(s.stepSendRegisteredCall)
}

func (s *StateMachineCallMethod) stepSendRegisteredCall(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		request                = s.requestInfo.Request.(*record.IncomingRequest)
		requestReference       = s.requestInfo.RequestReference
		requestObjectReference = s.requestInfo.RequestObjectReference
	)

	pl := &payload.AdditionalCallFromPreviousExecutor{
		ObjectReference: requestObjectReference,
		RequestRef:      requestReference,
		Request:         request,

		// TODO[bigbes]: what should be here (??)
		ServiceData: common.ServiceDataFromContext(ctx.GetContext()),
	}

	msg, err := payload.NewMessage(pl)
	if err != nil {
		panic("couldn't serialize message: " + err.Error())
	}

	goCtx := ctx.GetContext()
	return s.sender.PrepareNotify(ctx, func(svc s_sender.SenderService) {
		_, done := svc.SendRole(goCtx, msg, insolar.DynamicRoleVirtualExecutor, requestObjectReference)
		done()
	}).DelayedSend().ThenJump(s.stepDone)
}
