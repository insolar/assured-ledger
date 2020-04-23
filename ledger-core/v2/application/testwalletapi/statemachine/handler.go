// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
)

type TestAPICall struct {
	Payload  payload.VCallRequest
	Response chan payload.VCallResult
}

func Handler(call *TestAPICall) smachine.CreateFunc {
	return func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &SMTestAPICall{
			requestPayload: call.Payload,
			response:       call.Response,
		}
	}
}
