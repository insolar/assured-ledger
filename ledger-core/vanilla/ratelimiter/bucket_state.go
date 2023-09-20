package ratelimiter

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

type BucketState struct {
	filledAmount   atomickit.Uint32
	residualAmount atomickit.Uint32 // scaled value!
	bucketConfig
}
type bucketConfig = BucketConfig

type BucketRefillFunc func(need uint32) uint32

func (p *BucketState) TakeQuotaNoWait(max int64, scale uint32, refillFn BucketRefillFunc) int64 {
	if max <= 0 {
		return 0
	}
	for {
		switch n := p.residualAmount.Load(); {
		case int64(n) >= max:
			if p.residualAmount.CompareAndSwap(n, n-uint32(max)) {
				return max
			}
		case n > 0:
			if p.residualAmount.CompareAndSwap(n, 0) {
				return int64(n) + p._takeQuotaNoWait(max-int64(n), scale, refillFn)
			}
		default:
			return p._takeQuotaNoWait(max, scale, refillFn)
		}
	}
}

func (p *BucketState) _takeQuotaNoWait(max int64, scale uint32, refillFn BucketRefillFunc) int64 {
	maxUnscaled := (uint64(max) + uint64(scale) - 1) / uint64(scale)
	if maxUnscaled > math.MaxUint32 {
		maxUnscaled = math.MaxUint32
	}

	allocated := int64(p.takeFilled(uint32(maxUnscaled), refillFn)) * int64(scale)
	if allocated <= max {
		return allocated
	}
	p.addResidual(uint64(allocated - max))
	return max
}

func (p *BucketState) addResidual(residual uint64) {
	for {
		n := p.residualAmount.Load()
		x := residual + uint64(n)
		if x > math.MaxUint32 {
			x = math.MaxUint32
		}
		if p.residualAmount.CompareAndSwap(n, uint32(x)) {
			return
		}
	}
}

func (p *BucketState) TakeQuotaNoScale(max uint32, refillFn BucketRefillFunc) uint32 {
	if max == 0 {
		return 0
	}
	return p.takeFilled(max, refillFn)
}

func (p *BucketState) takeFilled(max uint32, refillFn BucketRefillFunc) uint32 {
	allocate := p.takeQuantizedFilled(quantizeCeiling(max, p.Quantum), refillFn)
	if allocate <= max {
		return allocate
	}
	p.residualAmount.Add(allocate - max)
	return max
}

func (p *BucketState) takeQuantizedFilled(maxAligned uint32, refillFn BucketRefillFunc) uint32 {
	for allocated := uint32(0); ; {
		n := p.filledAmount.Load()

		switch needed := maxAligned - allocated; {
		case n < needed:
			switch allocated += refillFn(needed - n); {
			case allocated < maxAligned:
				//
			case allocated == maxAligned:
				return allocated
			default:
				p.filledAmount.Add(allocated - maxAligned)
				return allocated
			}
		case p.filledAmount.CompareAndSwap(n, n-needed):
			return maxAligned
		default:
			continue
		}

		switch q := quantizeFloor(allocated+n, p.Quantum); {
		case q > allocated:
			if !p.filledAmount.CompareAndSwap(n, n+allocated-q) {
				continue
			}
			return q
		case q == 0:
			if allocated > 0 {
				p.filledAmount.Add(allocated)
			}
			return 0
		case q == allocated:
			return q
		default:
			p.filledAmount.Add(allocated - q)
			return q
		}
	}
}

func (p *BucketState) PeriodsToRefill(periods uint64) uint32 {
	if delta := uint64(p.RefillAmount) * periods; delta < uint64(p.MaxAmount) {
		return uint32(delta)
	}
	return p.MaxAmount
}

func (p *BucketState) ForceRefill(x uint32) {
	if x == 0 {
		return
	}
	for {
		n := p.filledAmount.Load()
		v := n + x
		if v > p.MaxAmount {
			v = p.MaxAmount
		}
		if p.filledAmount.CompareAndSwap(n, v) {
			return
		}
	}
}

func (p *BucketState) config() bucketConfig {
	return bucketConfig{
		RefillAmount: p.RefillAmount,
		Quantum:      p.Quantum,
		MaxAmount:    p.MaxAmount,
	}
}
