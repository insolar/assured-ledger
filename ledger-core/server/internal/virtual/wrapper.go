// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtual

import (
	"context"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/virtual"
)

var _ component.Initer = wrapper{}
var _ component.Starter = wrapper{}
var _ component.Stopper = wrapper{}

type wrapper struct {
	runnerService     *runner.DefaultService
	virtualDispatcher *virtual.Dispatcher
	apiServer         *testwalletapi.TestWalletServer
}

func (v wrapper) Init(ctx context.Context) error {
	if err := v.runnerService.Init(); err != nil {
		return err
	}
	if err := v.virtualDispatcher.Init(ctx); err != nil {
		return err
	}
	// if err := v.apiServer.Init(ctx); err != nil {
	// 	return err
	// }
	return nil
}

func (v wrapper) Start(ctx context.Context) error {
	if err := v.virtualDispatcher.Start(ctx); err != nil {
		return err
	}
	if err := v.apiServer.Start(ctx); err != nil {
		return err
	}
	inslogger.FromContext(ctx).Info("All components has started")
	return nil
}

func (v wrapper) Stop(ctx context.Context) error {
	if err := v.virtualDispatcher.Stop(ctx); err != nil {
		return err
	}
	if err := v.apiServer.Stop(ctx); err != nil {
		return err
	}
	return nil
}

func (v wrapper) GetBeatDispatcher() beat.Dispatcher {
	return v.virtualDispatcher.FlowDispatcher
}
