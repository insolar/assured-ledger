// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
)

type TestAPICall struct {
	Payload payload.VCallRequest
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
	case v.Payload.CallType != payload.CTMethod:
	case !builtinTestAPIEchoBriefRef.Equal(v.Payload.Callee):
	default:
		return true, v.Payload.Arguments
	}

	return false, nil
}
