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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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

func FactoryMeta(message *statemachine.DispatcherMessage, authService authentication.Service) (pulse.Number, smachine.CreateFunc, error) {
	payloadMeta := message.PayloadMeta
	messageMeta := message.MessageMeta

	traceID := messageMeta.Get(defaults.TraceID)
	if traceID == "" {
		panic("TraceID is empty")
	}

	goCtx, _ := inslogger.WithTraceField(context.Background(), traceID)
	goCtx, logger := inslogger.WithField(goCtx, "component", "sm")

	payloadBytes := payloadMeta.Payload
	payloadTypeID, payloadObj, err := rms.Unmarshal(payloadBytes)
	if err != nil {
		logger.Warn(throw.WithSeverity(throw.W(err, "invalid message"), throw.ViolationSeverity))
		return pulse.Unknown, nil, nil
	}

	payloadType := rms.GetRegistry().Get(payloadTypeID)

	logger.Info(logProcessing{
		messageType: fmt.Sprintf("id=%d, type=%s", payloadTypeID, payloadType.String()),
	})

	err = authService.IsMessageFromVirtualLegitimate(goCtx, payloadObj, payloadMeta.Sender)
	if err != nil {
		logger.Warn(throw.W(err, "illegitimate message", struct {
			messageTypeID uint64
			messageType   reflect.Type
		}{messageTypeID: payloadTypeID, messageType: payloadType}))

		return pulse.Unknown, nil, nil
	}

	if pn, sm := func () (pulse.Number, smachine.StateMachine) {
		switch obj := payloadObj.(type) {
		case *payload.VCallRequest:
			return payloadMeta.Pulse, &SMVCallRequest{Meta: payloadMeta, Payload: obj}
		case *payload.VCallResult:
			return payloadMeta.Pulse, &SMVCallResult{Meta: payloadMeta, Payload: obj}
		case *payload.VStateRequest:
			return obj.AsOf, &SMVStateRequest{Meta: payloadMeta, Payload: obj}
		case *payload.VStateReport:
			return payloadMeta.Pulse, &SMVStateReport{Meta: payloadMeta, Payload: obj}
		case *payload.VDelegatedRequestFinished:
			return payloadMeta.Pulse, &SMVDelegatedRequestFinished{Meta: payloadMeta, Payload: obj}
		case *payload.VDelegatedCallRequest:
			return payloadMeta.Pulse, &SMVDelegatedCallRequest{Meta: payloadMeta, Payload: obj}
		case *payload.VDelegatedCallResponse:
			return payloadMeta.Pulse, &SMVDelegatedCallResponse{Meta: payloadMeta, Payload: obj}
		default:
			return 0, nil
		}
	}(); sm != nil {
		return pn, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return sm
		}, nil
	}

	logger.Warn(errNoHandler{messageTypeID: payloadTypeID, messageType: payloadType})

	return pulse.Unknown, nil, nil
}
