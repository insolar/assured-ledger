// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtual

import (
	"context"
	"runtime"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/virtual"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
)

func AppFactory(ctx context.Context, cfg configuration.Configuration, comps insapp.AppComponents) (insapp.AppComponent, error) {
	runnerService := runner.NewService()
	if err := runnerService.Init(); err != nil {
		return nil, err
	}

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

	return appComp{ runnerService, virtualDispatcher, testAPI }, nil
}

var _ component.Initer = appComp{}
var _ component.Starter = appComp{}
var _ component.Stopper = appComp{}

type appComp struct {
	runnerService     *runner.DefaultService
	virtualDispatcher *virtual.Dispatcher
	apiServer         *testwalletapi.TestWalletServer
}

func (v appComp) Init(ctx context.Context) error {
	// if err := v.runnerService.Init(); err != nil {
	// 	return err
	// }
	if err := v.virtualDispatcher.Init(ctx); err != nil {
		return err
	}
	// if err := v.apiServer.Init(ctx); err != nil {
	// 	return err
	// }
	return nil
}

func (v appComp) Start(ctx context.Context) error {
	if err := v.virtualDispatcher.Start(ctx); err != nil {
		return err
	}
	if err := v.apiServer.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (v appComp) Stop(ctx context.Context) error {
	if err := v.virtualDispatcher.Stop(ctx); err != nil {
		return err
	}
	if err := v.apiServer.Stop(ctx); err != nil {
		return err
	}
	return nil
}

func (v appComp) GetMessageHandler() message.NoPublishHandlerFunc {
	return v.virtualDispatcher.FlowDispatcher.Process
}

func (v appComp) GetBeatDispatcher() beat.Dispatcher {
	return v.virtualDispatcher.FlowDispatcher
}
