package virtual

import (
	"context"
	"runtime"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/virtual"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
)

func AppFactory(ctx context.Context, cfg configuration.Configuration, comps insapp.AppComponents) (insapp.AppComponent, error) {
	runnerService := runner.NewService()
	memoryCache := memorycache.NewDefaultService()
	virtualDispatcher := virtual.NewDispatcher()

	virtualDispatcher.Runner = runnerService
	virtualDispatcher.MemoryCache = memoryCache
	virtualDispatcher.MessageSender = comps.MessageSender
	virtualDispatcher.Affinity = comps.AffinityHelper
	virtualDispatcher.AuthenticationService = authentication.NewService(ctx, comps.AffinityHelper)
	virtualDispatcher.PlatformScheme = comps.CryptoScheme

	if cfg.Virtual.MaxRunners > 0 {
		virtualDispatcher.MaxRunners = cfg.Virtual.MaxRunners
	} else if n := runtime.NumCPU() - 2; n > 4 {
		virtualDispatcher.MaxRunners = n
	} else {
		virtualDispatcher.MaxRunners = 4
	}

	testAPI := testwalletapi.NewTestWalletServer(inslogger.FromContext(ctx), cfg.TestWalletAPI, virtualDispatcher, comps.BeatHistory)

	// ComponentManager can only work with by-pointer objects
	return &wrapper{runnerService, virtualDispatcher, testAPI}, nil
}
