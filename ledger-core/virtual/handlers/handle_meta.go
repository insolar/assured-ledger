// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"context"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/statemachine"
)

type FactoryMeta struct {
	LogContextOverride context.Context
	AuthService        authentication.Service
}

type skippedMessage struct {
	messageTypeID uint64
	messageType   reflect.Type
	incomingPulse pulse.Number
	targetPulse   pulse.Number
}

func (f FactoryMeta) Process(ctx context.Context, msg *statemachine.DispatcherMessage, pr pulse.Range) (pulse.Number, smachine.CreateFunc, error) {
	payloadMeta := msg.PayloadMeta
	messageMeta := msg.MessageMeta

	traceID := messageMeta.Get(defaults.TraceID)
	if traceID == "" {
		panic(throw.FailHere("TraceID is empty"))
	}

	logCtx := f.LogContextOverride
	if logCtx == nil {
		logCtx = ctx
	}
	logCtx, logger := inslogger.WithTraceField(logCtx, traceID)

	payloadBytes := payloadMeta.Payload
	payloadTypeID, payloadObj, err := rms.Unmarshal(payloadBytes)
	if err != nil {
		logger.Warnm(throw.WithSeverity(throw.W(err, "invalid msg"), throw.ViolationSeverity))
		return pulse.Unknown, nil, nil
	}

	payloadType := rms.GetRegistry().Get(payloadTypeID)

	logger.Infom(struct {
		Message         string
		PayloadTypeID   uint64
		PayloadTypeName string
	}{"processing message", payloadTypeID, payloadType.String()})

	targetPulse := pr.RightBoundData().PulseNumber
	if targetPulse != payloadMeta.Pulse {
		panic(throw.Impossible())
	}

	// don't check sender for future pulses in R0
	if !pr.RightBoundData().IsExpectedPulse() {
		mustReject, err := f.AuthService.IsMessageFromVirtualLegitimate(logCtx, payloadObj, payloadMeta.Sender, pr)
		if err != nil {
			logger.Warn(throw.W(err, "illegitimate msg", skippedMessage{
				messageTypeID: payloadTypeID,
				messageType:   payloadType,
				incomingPulse: payloadMeta.Pulse,
				targetPulse:   targetPulse,
			}))

			return pulse.Unknown, nil, nil
		}

		if mustReject {
			logger.Warn(throw.W(err, "rejected msg", skippedMessage{
				messageTypeID: payloadTypeID,
				messageType:   payloadType,
				incomingPulse: payloadMeta.Pulse,
				targetPulse:   targetPulse,
			}))
			// when this flag is set, then the relevant SM has to stop asap and send negative answer
			return pulse.Unknown, nil, throw.NotImplemented()
		}
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
		case *payload.VFindCallRequest:
			return obj.LookAt, &SMVFindCallRequest{Meta: payloadMeta, Payload: obj}
		case *payload.VFindCallResponse:
			return targetPulse, &SMVFindCallResponse{Meta: payloadMeta, Payload: obj}
		default:
			logger.Warnm(struct {
				Msg             string
				PayloadTypeID   uint64
				PayloadTypeName string
			}{"no handler for message type", payloadTypeID, payloadType.String()})
			return 0, nil
		}
	}(); sm != nil {
		return pn, func(constructorCtx smachine.ConstructionContext) smachine.StateMachine {
			constructorCtx.SetContext(logCtx)
			constructorCtx.SetTracerID(traceID)
			return sm
		}, nil
	}

	return pulse.Unknown, nil, nil
}
