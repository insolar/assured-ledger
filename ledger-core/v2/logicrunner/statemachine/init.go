// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/sm_request"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/sm_test_api"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

func DefaultHandlersFactory(_ pulse.Number, input conveyor.InputEvent) smachine.CreateFunc {
	switch inputConverted := input.(type) {
	case *common.DispatcherMessage:
		return sm_request.HandlerFactoryMeta(inputConverted)
	case *common.TestAPICall:
		return sm_test_api.Handler(inputConverted)
	// case *sm_execute_request.SMEventSendOutgoing:
	// 	return sm_execute_request.HandlerFactoryOutgoingSender(inputConverted)
	default:
		panic(fmt.Sprintf("unknown event type, got %T", input))
	}
}
