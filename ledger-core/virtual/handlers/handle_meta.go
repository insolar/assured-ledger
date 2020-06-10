// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/statemachine"
)

type logProcessing struct {
	*log.Msg `txt:"processing message"`

	messageType string
}

type errNoHandler struct {
	*log.Msg `txt:"no handler for message type"`

	messageTypeID uint64
	messageType   reflect.Type
}

func FactoryMeta(message *statemachine.DispatcherMessage, authService authentication.Service) (pulse.Number, smachine.CreateFunc) {
	payloadMeta := message.PayloadMeta
	messageMeta := message.MessageMeta

	traceID := messageMeta.Get(defaults.TraceID)
	if traceID == "" {
		panic("TraceID is empty")
	}

	payloadBytes := payloadMeta.Payload
	payloadTypeID, payloadObj, err := rms.Unmarshal(payloadBytes)
	if err != nil {
		panic(err)
	}

	payloadType := rms.GetRegistry().Get(payloadTypeID)

	goCtx, _ := inslogger.WithTraceField(context.Background(), traceID)
	goCtx, logger := inslogger.WithField(goCtx, "component", "sm")

	logger.Info(logProcessing{
		messageType: fmt.Sprintf("id=%d, type=%s", payloadTypeID, payloadType.String()),
	})

	err = authService.IsMessageFromVirtualLegitimate(goCtx, payloadObj, payloadMeta.Sender)
	if err != nil {
		logger.Warn(struct {
			*log.Msg      `txt:"illegitimate message"`
			messageTypeID uint64
			messageType   reflect.Type
			error         string
		}{messageTypeID: payloadTypeID, messageType: payloadType, error: err.Error()})
		return pulse.Unknown, nil
	}

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
	case *payload.VDelegatedRequestFinished:
		return payloadMeta.Pulse, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVDelegatedRequestFinished{Meta: payloadMeta, Payload: obj}
		}
	case *payload.VDelegatedCallRequest:
		return payloadMeta.Pulse, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVDelegatedCallRequest{Meta: payloadMeta, Payload: obj}
		}
	case *payload.VDelegatedCallResponse:
		return payloadMeta.Pulse, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return &SMVDelegatedCallResponse{Meta: payloadMeta, Payload: obj}
		}
	default:
		panic(errNoHandler{
			messageTypeID: payloadTypeID,
			messageType:   payloadType,
		})
	}
}
