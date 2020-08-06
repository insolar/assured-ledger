// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"
	"runtime"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
)

// jaeger is a default insolar tracer preset.
func jaeger(ctx context.Context, cfg configuration.JaegerConfig, traceID, nodeRef, nodeRole string) func() {

	runtime.KeepAlive(traceID) // linter

	inslogger.FromContext(ctx).Infof(
		"Tracing enabled. Agent endpoint: '%s', collector endpoint: '%s'\n",
		cfg.AgentEndpoint,
		cfg.CollectorEndpoint,
	)
	flush := instracer.ShouldRegisterJaeger(ctx, nodeRole, nodeRef,
		cfg.AgentEndpoint,
		cfg.CollectorEndpoint,
		cfg.ProbabilityRate,
	)
	return flush
}
