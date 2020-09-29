// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewMulti(cfgProvider ConfigurationProvider, appFn AppFactoryFunc, multiFn MultiNodeConfigFunc,
	extraComponents ...interface{},
) (*Server, MultiController) {
	if multiFn == nil {
		panic(throw.IllegalValue())
	}

	srv := newServer(&AppInitializer{
		appFn:        appFn,
		extra:        extraComponents,
		confProvider: cfgProvider,
	})

	srv.multi = &multiLifecycle{
		multiFn:   multiFn,
		apps: map[string]*appEntry{},
		loggerFn: func(baseCtx context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context {
			ctx, _ := inslogger.InitNodeLogger(baseCtx, cfg, nodeRef, nodeRole)
			return ctx
		},
	}

	return srv, srv.multi
}
