// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_request // nolint:golint

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
)

type logProcessing struct {
	*log.Msg `txt:"processing message"`

	messageType string
}

func HandlerFactoryMeta(message *common.DispatcherMessage) smachine.CreateFunc {
	payloadMeta := message.PayloadMeta
	messageMeta := message.MessageMeta
	traceID := messageMeta.Get(meta.TraceID)

	payloadBytes := payloadMeta.Payload
	payloadType, err := payload.UnmarshalType(payloadBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal payload type: %s", err.Error()))
	}

	goCtx, _ := inslogger.WithTraceField(context.Background(), traceID)
	goCtx, logger := inslogger.WithField(goCtx, "component", "sm")

	logger.Error(logProcessing{messageType: payloadType.String()})

	switch payloadType {
	case payload.TypeCallMethod:
		pl := payload.CallMethod{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.CallMethod: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachineCallMethod{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeSagaCallAcceptNotification:
		pl := payload.SagaCallAcceptNotification{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.SagaCallAcceptNotification: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachineSagaAccept{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeUpdateJet:
		pl := payload.UpdateJet{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.UpdateJet: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachineUpdateJet{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypePendingFinished:
		pl := payload.PendingFinished{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.PendingFinished: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachinePendingFinished{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeExecutorResults:
		pl := payload.ExecutorResults{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.ExecutorResults: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachineExecutorResults{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeStillExecuting:
		pl := payload.StillExecuting{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.StillExecuting: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachineStillExecuting{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeAdditionalCallFromPreviousExecutor:
		pl := payload.AdditionalCallFromPreviousExecutor{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.AdditionalCallFromPreviousExecutor: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachineAdditionalCall{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeAbandonedRequestsNotification:
		pl := payload.AbandonedRequestsNotification{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.AbandonedRequestsNotification: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachineAbandonedRequests{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeVCallResult:
		pl := payload.VCallResult{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(fmt.Sprintf("failed to unmarshal payload.VCallResult: %s", err.Error()))
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &StateMachineCallResult{Meta: payloadMeta, Payload: &pl}
		}

	default:
		panic(fmt.Sprintf(" no handler for message type %s", payloadType.String()))
	}
}
