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

type FactoryMeta struct {
	AuthService authentication.Service
}

func (f FactoryMeta) Process(msg *statemachine.DispatcherMessage, pr pulse.Range) (pulse.Number, smachine.CreateFunc, error) {
	payloadMeta := msg.PayloadMeta
	messageMeta := msg.MessageMeta

	traceID := messageMeta.Get(defaults.TraceID)
	if traceID == "" {
		panic("TraceID is empty")
	}

	goCtx, _ := inslogger.WithTraceField(context.Background(), traceID)
	goCtx, logger := inslogger.WithField(goCtx, "component", "sm")

	payloadBytes := payloadMeta.Payload
	payloadTypeID, payloadObj, err := rms.Unmarshal(payloadBytes)
	if err != nil {
		logger.Warn(throw.WithSeverity(throw.W(err, "invalid msg"), throw.ViolationSeverity))
		return pulse.Unknown, nil, nil
	}

	payloadType := rms.GetRegistry().Get(payloadTypeID)

	logger.Info(logProcessing{
		messageType: fmt.Sprintf("id=%d, type=%s", payloadTypeID, payloadType.String()),
	})

	targetPulse := pr.RightBoundData().PulseNumber
	if targetPulse != payloadMeta.Pulse {
		panic(throw.Impossible())
	}

	mustReject, err := f.AuthService.IsMessageFromVirtualLegitimate(goCtx, payloadObj, payloadMeta.Sender, pr)
	if err != nil {
		logger.Warn(throw.W(err, "illegitimate msg", struct {
			messageTypeID uint64
			messageType   reflect.Type
			incomingPulse pulse.Number
			targetPulse   pulse.Number
		}{
			messageTypeID: payloadTypeID,
			messageType:   payloadType,
			incomingPulse: payloadMeta.Pulse,
			targetPulse:   targetPulse,
		}))

		return pulse.Unknown, nil, nil
	}

	if mustReject {
		// when this flag is set, then the relevant SM has to stop asap and send negative answer
		return pulse.Unknown, nil, throw.NotImplemented()
	}

	if pn, sm := func() (pulse.Number, smachine.StateMachine) {
		switch obj := payloadObj.(type) {
		case *payload.VCallRequest:
			return targetPulse, &SMVCallRequest{Meta: payloadMeta, Payload: obj}
		case *payload.VCallResult:
			return targetPulse, &SMVCallResult{Meta: payloadMeta, Payload: obj}
		case *payload.VStateRequest:
			return obj.AsOf, &SMVStateRequest{Meta: payloadMeta, Payload: obj}
		case *payload.VStateReport:
			return targetPulse, &SMVStateReport{Meta: payloadMeta, Payload: obj}
		case *payload.VDelegatedRequestFinished:
			return targetPulse, &SMVDelegatedRequestFinished{Meta: payloadMeta, Payload: obj}
		case *payload.VDelegatedCallRequest:
			return targetPulse, &SMVDelegatedCallRequest{Meta: payloadMeta, Payload: obj}
		case *payload.VDelegatedCallResponse:
			return targetPulse, &SMVDelegatedCallResponse{Meta: payloadMeta, Payload: obj}
		default:
			logger.Warn(errNoHandler{messageTypeID: payloadTypeID, messageType: payloadType})
			return 0, nil
		}
	}(); sm != nil {
		return pn, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			ctx.SetContext(goCtx)
			ctx.SetTracerID(traceID)
			return sm
		}, nil
	}

	return pulse.Unknown, nil, nil
}
