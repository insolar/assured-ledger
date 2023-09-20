package ratelimiter

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

/****************************************************/

type PeriodRefiller struct {
	manager     *PeriodManager
	periodCount atomickit.Uint64
}

func (p *PeriodRefiller) TakeQuota(max int64, state *BucketState, refillFn BucketRefillFunc) int64 {
	currentPeriod := p.manager.currentPeriod.Load()
	for {
		if q := p.TakeQuotaNoWait(max, state, refillFn); q > 0 {
			return q
		}
		p.manager.waitNextPeriod(currentPeriod)
	}
}

func (p *PeriodRefiller) TakeQuotaNoWait(max int64, state *BucketState, refillFn BucketRefillFunc) int64 {
	return state.TakeQuotaNoWait(max, p.manager.amountScale, refillFn)
}

func (p *PeriodRefiller) GetRefillCount() uint64 {
	for {
		currentPeriod := p.manager.currentPeriod.Load()
		lastPeriod := p.periodCount.Load()
		if currentPeriod == lastPeriod {
			return 0
		}
		if p.periodCount.CompareAndSwap(lastPeriod, currentPeriod) {
			periods := currentPeriod - lastPeriod
			if periods > math.MaxUint32 {
				// to avoid overflows of following calculations
				return math.MaxUint32
			}
			return periods
		}
	}
}
