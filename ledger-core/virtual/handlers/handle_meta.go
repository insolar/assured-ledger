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
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
)

type FactoryMeta struct {
	LogContextOverride context.Context
	AuthService        authentication.Service
}

type skippedMessage struct {
	messageType   reflect.Type
	incomingPulse pulse.Number
	targetPulse   pulse.Number
}

func (f FactoryMeta) Process(ctx context.Context, msg insconveyor.DispatchedMessage, pr pulse.Range) (pulse.Number, smachine.CreateFunc, error) {
	payloadMeta := &msg.PayloadMeta
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

	var (
		payloadObj  = payloadMeta.Payload.Get()
		payloadType = reflect.Indirect(reflect.ValueOf(payloadObj)).Type()
		sender      = payloadMeta.Sender.GetValue()
	)

	logger.Infom(struct {
		Message        string
		PayloadType    reflect.Type
		Source, Target reference.Holder
	}{
		"processing message",
		payloadType,
		sender,
		payloadMeta.Receiver.GetValue(),
	})

	targetPulse := pr.RightBoundData().PulseNumber
	if targetPulse != payloadMeta.Pulse {
		panic(throw.Impossible())
	}

	// don't check sender for future pulses in R0
	switch payloadObj.(type) {
	case *rms.LRegisterResponse:
		// TODO: do nothing for now, message from ledger
	case *rms.LRegisterRequest:
		panic(throw.Impossible())
	default:
		if pr.RightBoundData().IsExpectedPulse() {
			break
		}

		mustReject, err := f.AuthService.CheckMessageFromAuthorizedVirtual(logCtx, payloadObj, sender, pr)
		if err != nil {
			logger.Warn(throw.W(err, "illegitimate msg", skippedMessage{
				messageType:   payloadType,
				incomingPulse: payloadMeta.Pulse,
				targetPulse:   targetPulse,
			}))

			return pulse.Unknown, nil, nil
		}

		if mustReject {
			logger.Warn(throw.W(err, "rejected msg", skippedMessage{
				messageType:   payloadType,
				incomingPulse: payloadMeta.Pulse,
				targetPulse:   targetPulse,
			}))
			// when this flag is set, then the relevant SM has to stop asap and send negative answer
			return pulse.Unknown, nil, throw.NotImplemented()
		}
	}

	// validate message field invariants
	if p, ok := payloadObj.(rms.Validatable); ok {
		if err := p.Validate(targetPulse); err != nil {
			logger.Warn(throw.W(err, "invalid msg", skippedMessage{
				messageType:   payloadType,
				incomingPulse: payloadMeta.Pulse,
				targetPulse:   targetPulse,
			}))
			return pulse.Unknown, nil, nil
		}
	}

	if pn, sm := func() (pulse.Number, smachine.StateMachine) {
		switch obj := payloadObj.(type) {
		case *rms.VCallRequest:
			return targetPulse, &SMVCallRequest{Meta: payloadMeta, Payload: obj}
		case *rms.VCallResult:
			return targetPulse, &SMVCallResult{Meta: payloadMeta, Payload: obj}
		case *rms.VStateRequest:
			return obj.AsOf, &SMVStateRequest{Meta: payloadMeta, Payload: obj}
		case *rms.VStateReport:
			return targetPulse, &SMVStateReport{Meta: payloadMeta, Payload: obj}
		case *rms.VDelegatedRequestFinished:
			return targetPulse, &SMVDelegatedRequestFinished{Meta: payloadMeta, Payload: obj}
		case *rms.VDelegatedCallRequest:
			return targetPulse, &SMVDelegatedCallRequest{Meta: payloadMeta, Payload: obj}
		case *rms.VDelegatedCallResponse:
			return targetPulse, &SMVDelegatedCallResponse{Meta: payloadMeta, Payload: obj}
		case *rms.VFindCallRequest:
			return obj.LookAt, &SMVFindCallRequest{Meta: payloadMeta, Payload: obj}
		case *rms.VFindCallResponse:
			return targetPulse, &SMVFindCallResponse{Meta: payloadMeta, Payload: obj}
		case *rms.VObjectTranscriptReport:
			return targetPulse, &SMVObjectTranscriptReport{Meta: payloadMeta, Payload: obj}
		case *rms.VCachedMemoryRequest:
			return targetPulse, &SMVCachedMemoryRequest{Meta: payloadMeta, Payload: obj}
		case *rms.VObjectValidationReport:
			return obj.In, &SMVObjectValidationReport{Meta: payloadMeta, Payload: obj}
		case *rms.LRegisterResponse:
			return targetPulse, &SMLRegisterResponse{Meta: payloadMeta, Payload: obj}
		default:
			logger.Warnm(struct {
				Msg             string
				PayloadTypeName string
			}{"no handler for message type", payloadType.String()})
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
