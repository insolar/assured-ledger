// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_request // nolint:golint

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
)

type logProcessing struct {
	*log.Msg `txt:"processing message"`

	messageType string
}

func HandlerFactoryMeta(message *common.DispatcherMessage) smachine.CreateFunc {
	payloadMeta := message.PayloadMeta
	messageMeta := message.MessageMeta
	traceID := messageMeta.Get(meta.TraceID)

	payloadBytes := payloadMeta.Payload
	payloadType, err := payload.UnmarshalType(payloadBytes)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal payload type: %s", err.Error()))
	}

	goCtx, _ := inslogger.WithTraceField(context.Background(), traceID)
	goCtx, logger := inslogger.WithField(goCtx, "component", "sm")

	logger.Error(logProcessing{messageType: payloadType.String()})

	panic(fmt.Sprintf(" no handler for message type %s", payloadType.String()))
}
