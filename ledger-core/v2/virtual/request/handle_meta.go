// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package request

import (
	"context"

	"github.com/google/uuid"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

type logProcessing struct {
	*log.Msg `txt:"processing message"`

	messageType string
}

type errNoHandler struct {
	*log.Msg `txt:"no handler for message type"`

	MessageType payload.Type
}

func HandlerFactoryMeta(message *statemachine.DispatcherMessage) smachine.CreateFunc {
	payloadMeta := message.PayloadMeta
	messageMeta := message.MessageMeta

	traceID := messageMeta.Get(meta.TraceID)
	if traceID == "" { // TODO[bigbes]: dirty hack, if we have no traceID - replace it with surrogate one
		traceID = uuid.New().String()
	}

	payloadBytes := payloadMeta.Payload
	payloadType, err := payload.UnmarshalType(payloadBytes)
	if err != nil {
		panic(err)
	}

	goCtx, _ := inslogger.WithTraceField(context.Background(), traceID)
	goCtx, logger := inslogger.WithField(goCtx, "component", "sm")

	logger.Error(logProcessing{messageType: payloadType.String()})

	switch payloadType {
	case payload.TypeVCallRequest:
		pl := payload.VCallRequest{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(err)
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVCallRequest{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeVCallResult:
		pl := payload.VCallResult{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(err)
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVCallResult{Meta: payloadMeta, Payload: &pl}
		}
	case payload.TypeVStateRequest:
		pl := payload.VStateRequest{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(err)
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVStateRequest{Meta: payloadMeta, Payload: &pl}
		}
	case payload.TypeVStateReport:
		pl := payload.VStateReport{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(err)
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVStateReport{Meta: payloadMeta, Payload: &pl}
		}

	case payload.TypeVStateUnavailable:
		pl := payload.VStateUnavailable{}
		if err := pl.Unmarshal(payloadBytes); err != nil {
			panic(err)
		}
		return func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVStateUnavailable{Meta: payloadMeta, Payload: &pl}
		}
	default:
		panic(errNoHandler{MessageType: payloadType})
	}
}
