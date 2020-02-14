// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package s_jet_storage

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
)

// TODO[bigbes]: port it to state machine
type JetStorageService interface {
	jet.Storage
}

type JetStorageServiceAdapter struct {
	svc  JetStorageService
	exec smachine.ExecutionAdapter
}

func (a *JetStorageServiceAdapter) PrepareSync(ctx smachine.ExecutionContext, fn func(svc JetStorageService)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *JetStorageServiceAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc JetStorageService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func (a *JetStorageServiceAdapter) PrepareNotify(ctx smachine.ExecutionContext, fn func(svc JetStorageService)) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(interface{}) { fn(a.svc) })
}

type jetStorageService struct {
	jet.Storage
	Accessor pulse.Accessor
}

func CreateJetStorageService(JetStorage jet.Storage) *JetStorageServiceAdapter {
	ctx := context.Background()
	ae, ch := smachine.NewCallChannelExecutor(ctx, -1, false, 16)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &JetStorageServiceAdapter{
		svc: jetStorageService{
			Storage: JetStorage,
		},
		exec: smachine.NewExecutionAdapter("JetStorage", ae),
	}
}
