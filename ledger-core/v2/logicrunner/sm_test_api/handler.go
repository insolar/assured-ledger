// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_test_api

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
)

func Handler(call *common.TestApiCall) smachine.CreateFunc {
	return func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &TestApiCallSM{
			requestPayload: call.Payload,
			response:       call.Response,
		}
	}
}
