package smachine

import (
	"context"
	"math"
)

func StartChannelWorker(ctx context.Context, ch <-chan AdapterCall, runArg interface{}) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case t, ok := <-ch:
				if !ok {
					return
				}
				if err := t.RunAndSendResult(runArg); err != nil {
					t.ReportError(err)
				}
			}
		}
	}()
}

func StartChannelWorkerParallelCalls(ctx context.Context, parallelLimit uint16, ch <-chan AdapterCall, runArg interface{}) {
	switch parallelLimit {
	case 1:
		StartChannelWorker(ctx, ch, runArg)
		return
	case 0, math.MaxInt16:
		startChannelWorkerUnlimParallel(ctx, ch, runArg)
	default:
		for i := parallelLimit; i > 0; i-- {
			StartChannelWorker(ctx, ch, runArg)
		}
	}
}

func startChannelWorkerUnlimParallel(ctx context.Context, ch <-chan AdapterCall, runArg interface{}) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case t, ok := <-ch:
				if !ok {
					return
				}
				go func() {
					if err := t.RunAndSendResult(runArg); err != nil {
						t.ReportError(err)
					}
				}()
			}
		}
	}()
}
