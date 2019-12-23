package common

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
)

func ServiceDataFromContext(ctx context.Context) *payload.ServiceData {
	if ctx == nil {
		log.Error("nil context, can't create correct ServiceData")
		return &payload.ServiceData{}
	}
	return &payload.ServiceData{
		LogTraceID:    inslogger.TraceID(ctx),
		LogLevel:      inslogger.GetLoggerLevel(ctx),
		TraceSpanData: instracer.MustSerialize(ctx),
	}
}
