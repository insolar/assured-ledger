package smachines

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
)

type HashingAdapter struct {
	exec smachine.ExecutionAdapter
}

func (a *HashingAdapter) PrepareAsync(
	ctx smachine.ExecutionContext,
	fn func() smachine.AsyncResultFunc,
) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn()
	})
}

func NewHashingAdapter() *HashingAdapter {
	ctx := context.Background()
	exec, ch := smachine.NewCallChannelExecutor(ctx, -1, false, 8)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &HashingAdapter{
		exec: smachine.NewExecutionAdapter("HashingAdapter", exec),
	}
}

type SyncAdapter struct {
	exec smachine.ExecutionAdapter
}

func (a *SyncAdapter) PrepareAsync(
	ctx smachine.ExecutionContext,
	fn func() smachine.AsyncResultFunc,
) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn()
	})
}

func NewSyncAdapter() *SyncAdapter {
	ctx := context.Background()
	exec, ch := smachine.NewCallChannelExecutor(ctx, -1, false, 32)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &SyncAdapter{
		exec: smachine.NewExecutionAdapter("SyncAdapter", exec),
	}
}
