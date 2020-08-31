// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapter

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type CallFunc func(ctx context.Context, svc messagesender.Service)
type AsyncCallFunc func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter.MessageSender -o ./ -s _mock.go -g
type MessageSender interface {
	PrepareSync(ctx smachine.ExecutionContext, fn CallFunc) smachine.SyncCallRequester
	PrepareAsync(ctx smachine.ExecutionContext, fn AsyncCallFunc) smachine.AsyncCallRequester
	PrepareNotify(ctx smachine.ExecutionContext, fn CallFunc) smachine.NotifyRequester
}

type ParallelMessageSender struct {
	svc  messagesender.Service
	exec smachine.ExecutionAdapter
}

func (a *ParallelMessageSender) PrepareSync(ctx smachine.ExecutionContext, fn CallFunc) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(ctx context.Context, _ interface{}) smachine.AsyncResultFunc {
		fn(ctx, a.svc)
		return nil
	})
}

func (a *ParallelMessageSender) PrepareAsync(ctx smachine.ExecutionContext, fn AsyncCallFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(ctx context.Context, _ interface{}) smachine.AsyncResultFunc {
		return fn(ctx, a.svc)
	})
}

func (a *ParallelMessageSender) PrepareNotify(ctx smachine.ExecutionContext, fn CallFunc) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(ctx context.Context, _ interface{}) {
		fn(ctx, a.svc)
	})
}

func CreateMessageSendService(ctx context.Context, svc messagesender.Service) *ParallelMessageSender {
	if svc == nil {
		panic(throw.IllegalValue())
	}

	// it's copy/past from other realizations
	parallelReaders := 16
	ae, ch := smadapter.NewCallChannelExecutor(ctx, -1, false, parallelReaders)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &ParallelMessageSender{
		svc:  svc,
		exec: smachine.NewExecutionAdapter("MessageSendService", ae),
	}
}
