package checker

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

func waitCounterIndefinitely(ctx context.Context, counter *atomickit.Int, count int) synckit.SignalChannel {
	if counter.Load() >= count {
		return synckit.ClosedChannel()
	}

	ch := make(synckit.ClosableSignalChannel)
	go func() {
		defer close(ch)

		for {
			if counter.Load() >= count {
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Millisecond):
			}
		}
	}()

	return ch
}
