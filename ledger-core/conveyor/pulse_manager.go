// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type PulseDataServicePrepareFunc func(smachine.ExecutionContext, func(context.Context, PulseDataService) smachine.AsyncResultFunc) smachine.AsyncCallRequester

type PulseDataManager struct {
	// set at construction, immutable
	pulseDataAdapterFn PulseDataServicePrepareFunc

	cache PulseDataCache

	// set at init, immutable
	maxPastPulseAge uint32
	futureCycles    uint8

	// mutable
	presentAndFuturePulse uint64 // atomic
	earliestCacheBound    uint32 // atomic
	preparingPulseFlag    uint32 // atomic
}

type PulseDataService interface {
	LoadPulseData(context.Context, pulse.Number) (pulse.Data, bool)
}

func CreatePulseDataAdapterFn(ctx context.Context, pds PulseDataService, bufMax, parallelReaders int) PulseDataServicePrepareFunc {
	if pds == nil {
		panic("illegal value")
	}
	n := parallelReaders
	switch {
	case n <= 0 || n == math.MaxInt16:
		n = 64
	case n > math.MaxInt16:
		panic("illegal value")
	}

	executor, callChan := smachine.NewCallChannelExecutor(ctx, bufMax, false, n)
	pulseDataAdapter := smachine.NewExecutionAdapter(smachine.AdapterID(injector.GetDefaultInjectionID(pds)), executor)

	smachine.StartChannelWorkerParallelCalls(ctx, uint16(parallelReaders), callChan, pds)

	return func(ctx smachine.ExecutionContext, fn func(ctx context.Context, svc PulseDataService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
		return pulseDataAdapter.PrepareAsync(ctx, func(ctx context.Context, svc interface{}) smachine.AsyncResultFunc {
			fn(ctx, svc.(PulseDataService))
			return nil
		})
	}
}

func (p *PulseDataManager) init(minCachePulseAge, maxPastPulseAge uint32, maxFutureCycles uint8, pulseDataFn PulseDataServicePrepareFunc) {
	if minCachePulseAge == 0 || minCachePulseAge > pulse.MaxTimePulse {
		panic("illegal value")
	}
	if maxPastPulseAge < minCachePulseAge || maxPastPulseAge > pulse.MaxTimePulse {
		panic("illegal value")
	}
	p.pulseDataAdapterFn = pulseDataFn
	p.maxPastPulseAge = maxPastPulseAge
	p.futureCycles = maxFutureCycles
	p.cache.Init(p, minCachePulseAge, 2) // any pulse data stays in cache for at least 2 pulse cycles
}

const uninitializedFuture = pulse.LocalRelative

func (p *PulseDataManager) GetPresentPulse() (present pulse.Number, nearestFuture pulse.Number) {
	v := atomic.LoadUint64(&p.presentAndFuturePulse)
	if v == 0 {
		return pulse.Unknown, uninitializedFuture
	}
	return p._split(v)
}

func (p *PulseDataManager) setUninitializedFuturePulse(futurePN pulse.Number) bool {
	return atomic.CompareAndSwapUint64(&p.presentAndFuturePulse, 0, uint64(futurePN)<<32)
}

func (*PulseDataManager) _split(v uint64) (present pulse.Number, nearestFuture pulse.Number) {
	return pulse.Number(v), pulse.Number(v >> 32)
}

func (p *PulseDataManager) setPresentPulse(pd pulse.Data) {
	presentPN := pd.PulseNumber
	futurePN := pd.NextPulseNumber()

	if epd, ok := p.cache.Check(presentPN); ok {
		if epd != pd {
			panic("illegal state")
		}
	}

	for {
		prev := atomic.LoadUint64(&p.presentAndFuturePulse)
		if prev != 0 {
			expectedPN := pulse.Number(prev >> 32)
			if pd.PulseNumber < expectedPN {
				panic(fmt.Errorf("illegal pulse data: pn=%v, expected=%v", presentPN, expectedPN))
			}
		}
		if atomic.CompareAndSwapUint64(&p.presentAndFuturePulse, prev, uint64(presentPN)|uint64(futurePN)<<32) {
			if prev == 0 {
				atomic.CompareAndSwapUint32(&p.earliestCacheBound, 0, uint32(presentPN))
			}
			break
		}
	}

	p.cache.EvictAndRotate(presentPN)
}

func (p *PulseDataManager) getEarliestCacheBound() pulse.Number {
	return pulse.Number(atomic.LoadUint32(&p.earliestCacheBound))
}

func (p *PulseDataManager) isPreparingPulse() bool {
	return atomic.LoadUint32(&p.preparingPulseFlag) != 0
}

func (p *PulseDataManager) setPreparingPulse(_ PreparePulseChangeChannel) {
	atomic.StoreUint32(&p.preparingPulseFlag, 1)
}

func (p *PulseDataManager) unsetPreparingPulse() {
	atomic.StoreUint32(&p.preparingPulseFlag, 0)
}

func (p *PulseDataManager) GetPulseData(pn pulse.Number) (pulse.Data, bool) {
	return p.cache.Get(pn)
}

func (p *PulseDataManager) getCachedPulseSlot(pn pulse.Number) (*PulseSlot, bool) {
	return p.cache.getPulseSlot(pn)
}

// for non-recent past HasPulseData() can be incorrect / incomplete
func (p *PulseDataManager) HasPulseData(pn pulse.Number) bool {
	return p.cache.Contains(pn)
}

func (p *PulseDataManager) TouchPulseData(pn pulse.Number) bool {
	return p.cache.Touch(pn)
}

// IsAllowedFutureSpan Returns true when the given PN can be accepted into Future pulse slot, otherwise must be rejected
func (p *PulseDataManager) IsAllowedFutureSpan(futurePN pulse.Number) bool {
	presentPN, expectedPN := p.GetPresentPulse()
	return p.isAllowedFutureSpan(presentPN, expectedPN, futurePN)
}

func (p *PulseDataManager) isAllowedFutureSpan(presentPN, expectedPN pulse.Number, futurePN pulse.Number) bool {
	if futurePN < expectedPN {
		return false
	}
	return p.futureCycles == 0 || futurePN <= (expectedPN+(expectedPN-presentPN)*pulse.Number(p.futureCycles))
}

func (p *PulseDataManager) IsAllowedPastSpan(pastPN pulse.Number) bool {
	presentPN, _ := p.GetPresentPulse()
	return p.isAllowedPastSpan(presentPN, pastPN)
}

func (p *PulseDataManager) isAllowedPastSpan(presentPN pulse.Number, pastPN pulse.Number) bool {
	return pastPN < presentPN && pastPN+pulse.Number(p.maxPastPulseAge) >= presentPN
}

func (p *PulseDataManager) IsRecentPastRange(pastPN pulse.Number) bool {
	presentPN, _ := p.GetPresentPulse()
	return p.isRecentPastRange(presentPN, pastPN)
}

// isRecentPastRange returns true when the given PN is within a mandatory retention interval for the cache. So we don't need to populate it
func (p *PulseDataManager) isRecentPastRange(presentPN pulse.Number, pastPN pulse.Number) bool {
	return pastPN < presentPN &&
		(pastPN+pulse.Number(p.cache.GetMinRange())) >= presentPN &&
		pastPN >= p.getEarliestCacheBound() // this interval can be much narrower for a recently started node
}

func (p *PulseDataManager) PreparePulseDataRequest(ctx smachine.ExecutionContext,
	pn pulse.Number,
	resultFn func(isAvailable bool, pd pulse.Data),
) smachine.AsyncCallRequester {
	switch {
	case resultFn == nil:
		panic("illegal value")
	case p.pulseDataAdapterFn == nil:
		panic("illegal state")
	}
	if pd, ok := p.GetPulseData(pn); ok {
		resultFn(ok, pd)
	}

	return p.pulseDataAdapterFn(ctx, func(ctx context.Context, svc PulseDataService) smachine.AsyncResultFunc {
		pd, ok := svc.LoadPulseData(ctx, pn)

		return func(ctx smachine.AsyncResultContext) {
			if ok && pd.IsValidPulsarData() {
				p.putPulseData(pd)
				resultFn(ok, pd)
			} else {
				resultFn(false, pulse.Data{})
			}
		}
	}).WithFlags(smachine.AutoWakeUp)
}

func (p *PulseDataManager) putPulseData(data pulse.Data) {
	p.cache.Put(data)
}
