// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package s_messenger

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/facade"
)

type MessengerService interface {
	facade.Messenger
}

type MessengerServiceAdapter struct {
	svc  MessengerService
	exec smachine.ExecutionAdapter
}

func (a *MessengerServiceAdapter) PrepareSync(
	ctx smachine.ExecutionContext,
	fn func(svc MessengerService),
) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *MessengerServiceAdapter) PrepareAsync(
	ctx smachine.ExecutionContext,
	fn func(svc MessengerService) smachine.AsyncResultFunc,
) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func (a *MessengerServiceAdapter) PrepareNotify(
	ctx smachine.ExecutionContext,
	fn func(svc MessengerService),
) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(interface{}) {
		fn(a.svc)
	})
}

type messengerService struct {
	facade.Messenger
}

func CreateMessengerService(ctx context.Context, messenger facade.Messenger) *MessengerServiceAdapter {
	// it's copy/past from other realizations
	parallelReaders := 16
	ae, ch := smachine.NewCallChannelExecutor(ctx, -1, false, parallelReaders)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &MessengerServiceAdapter{
		svc: messengerService{
			Messenger: messenger,
		},
		exec: smachine.NewExecutionAdapter("MessengerService", ae),
	}
}
