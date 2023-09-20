package foundation

import (
	"github.com/insolar/gls"

	"github.com/insolar/assured-ledger/ledger-core/runner/call"
)

const glsCallContextKey = "callCtx"

// GetLogicalContext returns current calling context.
func GetLogicalContext() *call.LogicContext {
	ctx := gls.Get(glsCallContextKey)
	if ctx == nil {
		panic("object has no context")
	}

	if ctx, ok := ctx.(*call.LogicContext); ok {
		return ctx
	}

	panic("wrong type of context")
}

// SetLogicalContext saves current calling context
func SetLogicalContext(ctx *call.LogicContext) {
	gls.Set(glsCallContextKey, ctx)
}

// ClearContext clears underlying gls context
func ClearContext() {
	gls.Cleanup()
}
