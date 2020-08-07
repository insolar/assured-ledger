// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtual

import (
	"context"
	"runtime"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/virtual"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
)

func AppFactory(ctx context.Context, cfg configuration.Configuration, comps insapp.AppComponents) (insapp.AppComponent, error) {
	runnerService := runner.NewService()
	virtualDispatcher := virtual.NewDispatcher()

	virtualDispatcher.Runner = runnerService
	virtualDispatcher.MessageSender = comps.MessageSender
	virtualDispatcher.Affinity = comps.AffinityHelper
	virtualDispatcher.AuthenticationService = authentication.NewService(ctx, comps.AffinityHelper)

	// TODO: rewrite this after PLAT-432
	if n := runtime.NumCPU() - 2; n > 4 {
		virtualDispatcher.MaxRunners = n
	} else {
		virtualDispatcher.MaxRunners = 4
	}

	testAPI := testwalletapi.NewTestWalletServer(cfg.TestWalletAPI, virtualDispatcher, comps.BeatHistory)

	// ComponentManager can only work with by-pointer objects
	return &wrapper{runnerService, virtualDispatcher, testAPI }, nil
}

