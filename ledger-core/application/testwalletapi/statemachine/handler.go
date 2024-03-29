package statemachine

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type TestAPICall struct {
	Payload rms.VCallRequest
}

func (v TestAPICall) AsSMCreate() smachine.CreateFunc {
	return func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &SMTestAPICall{
			requestPayload: v.Payload,
		}
	}
}

func (v TestAPICall) CanBeBrief() (bool, []byte) {
	switch {
	case v.Payload.CallType != rms.CallTypeMethod:
	case !builtinTestAPIEchoBriefRef.Equal(v.Payload.Callee.GetValue()):
	default:
		return true, v.Payload.Arguments.GetBytes()
	}

	return false, nil
}
