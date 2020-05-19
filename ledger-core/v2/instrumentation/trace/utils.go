// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package trace

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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
	if ID(ctx) != "" {
		return context.WithValue(ctx, traceIDKey{}, traceid),
			errors.Errorf("TraceID already set: old: %s new: %s", ID(ctx), traceid)
	}
	return context.WithValue(ctx, traceIDKey{}, traceid), nil
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
