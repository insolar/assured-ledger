// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executor

import (
	"context"
	"time"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

type BadgerGCRunner interface {
	// RunValueGC run badger values garbage collection
	RunValueGC(ctx context.Context)
}

type BadgerGCRunInfo struct {
	runner BadgerGCRunner
	// runFrequency is period of running gc (in number of pulses)
	runFrequency uint

	callCounter uint
	tryLock     chan struct{}
}

func NewBadgerGCRunInfo(runner BadgerGCRunner, runFrequency uint) *BadgerGCRunInfo {
	tryLock := make(chan struct{}, 1)
	tryLock <- struct{}{}
	return &BadgerGCRunInfo{
		runner:       runner,
		runFrequency: runFrequency,
		tryLock:      tryLock,
	}
}

func (b *BadgerGCRunInfo) RunGCIfNeeded(ctx context.Context) (doneWaiter <-chan struct{}) {
	done := make(chan struct{}, 1)
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		select {
		case v := <-b.tryLock:
			b.callCounter++
			if (b.runFrequency > 0) && (b.callCounter >= b.runFrequency) && (b.callCounter%b.runFrequency == 0) {
				startedAt := time.Now().Second()
				b.runner.RunValueGC(ctx)
				stats.Record(ctx, statBadgerValueGCTime.M(int64(time.Now().Second()-startedAt)))
			} else {
				inslogger.FromContext(ctx).Info("values GC is not called")
			}
			b.tryLock <- v
		default:
			inslogger.FromContext(ctx).Info("values GC in progress. Skip It")
		}
	}()

	return done
}
