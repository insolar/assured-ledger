// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"context"

	"github.com/google/uuid"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/rms"
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

func FactoryMeta(message *statemachine.DispatcherMessage) (pulse.Number, smachine.CreateFunc) {
	payloadMeta := message.PayloadMeta
	messageMeta := message.MessageMeta

	traceID := messageMeta.Get(defaults.TraceID)
	if traceID == "" { // TODO[bigbes]: dirty hack, if we have no traceID - replace it with surrogate one
		traceID = uuid.New().String()
	}

	payloadBytes := payloadMeta.Payload
	payloadType, payloadObj, err := rms.Unmarshal(payloadBytes)
	if err != nil {
		panic(err)
	}

	goCtx, _ := inslogger.WithTraceField(context.Background(), traceID)
	goCtx, logger := inslogger.WithField(goCtx, "component", "sm")

	logger.Info(logProcessing{messageType: string(payloadType)})

	switch obj := payloadObj.(type) {
	case *payload.VCallRequest:
		return payloadMeta.Pulse, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVCallRequest{Meta: payloadMeta, Payload: obj}
		}

	case *payload.VCallResult:
		return payloadMeta.Pulse, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVCallResult{Meta: payloadMeta, Payload: obj}
		}
	case *payload.VStateRequest:
		return obj.AsOf, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVStateRequest{Meta: payloadMeta, Payload: obj}
		}
	case *payload.VStateReport:
		return payloadMeta.Pulse, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVStateReport{Meta: payloadMeta, Payload: obj}
		}
	case *payload.VStateUnavailable:
		return payloadMeta.Pulse, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVStateUnavailable{Meta: payloadMeta, Payload: obj}
		}
	case *payload.VDelegatedRequestFinished:
		return payloadMeta.Pulse, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVDelegatedRequestFinished{Meta: payloadMeta, Payload: obj}
		}
	default:
		panic(errNoHandler{MessageType: payload.Type(payloadType)})
	}
}
