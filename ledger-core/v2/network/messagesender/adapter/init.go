// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapter

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter.MessageSender -o ./ -s _mock.go -g
type MessageSender interface {
	PrepareSync(ctx smachine.ExecutionContext, fn func(svc messagesender.Service)) smachine.SyncCallRequester
	PrepareAsync(ctx smachine.ExecutionContext, fn func(svc messagesender.Service) smachine.AsyncResultFunc) smachine.AsyncCallRequester
	PrepareNotify(ctx smachine.ExecutionContext, fn func(svc messagesender.Service)) smachine.NotifyRequester
}

type ParallelMessageSender struct {
	svc  messagesender.Service
	exec smachine.ExecutionAdapter
}

func (a *ParallelMessageSender) PrepareSync(
	ctx smachine.ExecutionContext,
	fn func(svc messagesender.Service),
) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *ParallelMessageSender) PrepareAsync(
	ctx smachine.ExecutionContext,
	fn func(svc messagesender.Service) smachine.AsyncResultFunc,
) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func (a *ParallelMessageSender) PrepareNotify(
	ctx smachine.ExecutionContext,
	fn func(svc messagesender.Service),
) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(interface{}) {
		fn(a.svc)
	})
}

func CreateMessageSendService(ctx context.Context, messenger messagesender.Service) *ParallelMessageSender {
	// it's copy/past from other realizations
	parallelReaders := 16
	ae, ch := smachine.NewCallChannelExecutor(ctx, -1, false, parallelReaders)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &ParallelMessageSender{
		svc:  messenger,
		exec: smachine.NewExecutionAdapter("MessageSendService", ae),
	}
}
