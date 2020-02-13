// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package foundation

import (
	"github.com/tylerb/gls"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

const glsCallContextKey = "callCtx"

// GetLogicalContext returns current calling context.
func GetLogicalContext() *insolar.LogicCallContext {
	ctx := gls.Get(glsCallContextKey)
	if ctx == nil {
		panic("object has no context")
	}

	if ctx, ok := ctx.(*insolar.LogicCallContext); ok {
		return ctx
	}

	panic("wrong type of context")
}

// SetLogicalContext saves current calling context
func SetLogicalContext(ctx *insolar.LogicCallContext) {
	gls.Set(glsCallContextKey, ctx)
}

// ClearContext clears underlying gls context
func ClearContext() {
	gls.Cleanup()
}
