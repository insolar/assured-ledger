package adapter

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
)

type CallFunc func(ctx context.Context, svc memorycache.Service)
type AsyncCallFunc func(ctx context.Context, svc memorycache.Service) smachine.AsyncResultFunc

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter.MemoryCache -o ./ -s _mock.go -g
type MemoryCache interface {
	PrepareSync(ctx smachine.ExecutionContext, fn CallFunc) smachine.SyncCallRequester
	PrepareAsync(ctx smachine.ExecutionContext, fn AsyncCallFunc) smachine.AsyncCallRequester
	PrepareNotify(ctx smachine.ExecutionContext, fn CallFunc) smachine.NotifyRequester
}

type DefaultMemoryCacheAdapter struct {
	svc  memorycache.Service
	exec smachine.ExecutionAdapter
}

func (a *DefaultMemoryCacheAdapter) PrepareSync(ctx smachine.ExecutionContext, fn CallFunc) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(ctx context.Context, _ interface{}) smachine.AsyncResultFunc {
		fn(ctx, a.svc)
		return nil
	})
}

func (a *DefaultMemoryCacheAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn AsyncCallFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(ctx context.Context, _ interface{}) smachine.AsyncResultFunc {
		return fn(ctx, a.svc)
	})
}

func (a *DefaultMemoryCacheAdapter) PrepareNotify(ctx smachine.ExecutionContext, fn CallFunc) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(ctx context.Context, _ interface{}) {
		fn(ctx, a.svc)
	})
}

func CreateMemoryCacheAdapter(ctx context.Context, svc memorycache.Service) *DefaultMemoryCacheAdapter {
	if svc == nil {
		panic(throw.IllegalValue())
	}

	// it's copy/past from other realizations
	parallelReaders := 16
	ae, ch := smadapter.NewCallChannelExecutor(ctx, -1, false, parallelReaders)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &DefaultMemoryCacheAdapter{
		svc:  svc,
		exec: smachine.NewExecutionAdapter("MemoryCacheService", ae),
	}
}
