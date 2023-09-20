package trace

import (
	"context"
	"encoding/binary"
	"fmt"

	uuid "github.com/satori/go.uuid"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type traceIDKey struct{}

// ID returns traceid provided by WithTraceField and ContextWithTrace helpers.
func ID(ctx context.Context) string {
	val := ctx.Value(traceIDKey{})
	if val == nil {
		return ""
	}
	return val.(string)
}

func SetID(ctx context.Context, traceid string) (context.Context, error) {
	switch id := ID(ctx); {
	case id == "":
		return context.WithValue(ctx, traceIDKey{}, traceid), nil
	case id == traceid:
		return ctx, nil
	default:
		return context.WithValue(ctx, traceIDKey{}, traceid),
			errors.Errorf("TraceID already set: old: %s new: %s", ID(ctx), traceid)
	}
}

// RandID returns random traceID in uuid format.
func RandID() string {
	traceID, err := uuid.NewV4()
	if err != nil {
		return "createRandomTraceIDFailed:" + err.Error()
	}
	// We use custom serialization to be able to pass this trace to jaeger TraceID
	hi, low := binary.LittleEndian.Uint64(traceID[:8]), binary.LittleEndian.Uint64(traceID[8:])
	return fmt.Sprintf("%016x%016x", hi, low)
}
