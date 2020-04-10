// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package s_messagesend

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesend"
)

type MessageSendService interface {
	messagesend.Service
}

type MessageSendServiceAdapter struct {
	svc  MessageSendService
	exec smachine.ExecutionAdapter
}

func (a *MessageSendServiceAdapter) PrepareSync(
	ctx smachine.ExecutionContext,
	fn func(svc MessageSendService),
) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *MessageSendServiceAdapter) PrepareAsync(
	ctx smachine.ExecutionContext,
	fn func(svc MessageSendService) smachine.AsyncResultFunc,
) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func (a *MessageSendServiceAdapter) PrepareNotify(
	ctx smachine.ExecutionContext,
	fn func(svc MessageSendService),
) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(interface{}) {
		fn(a.svc)
	})
}

type messengerService struct {
	messagesend.Service
}

func CreateMessageSendService(ctx context.Context, messenger messagesend.Service) *MessageSendServiceAdapter {
	// it's copy/past from other realizations
	parallelReaders := 16
	ae, ch := smachine.NewCallChannelExecutor(ctx, -1, false, parallelReaders)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &MessageSendServiceAdapter{
		svc: messengerService{
			Service: messenger,
		},
		exec: smachine.NewExecutionAdapter("MessageSendService", ae),
	}
}
