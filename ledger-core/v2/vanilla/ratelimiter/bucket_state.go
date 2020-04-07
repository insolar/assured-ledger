// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ratelimiter

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
)

type BucketState struct {
	filledAmount   atomickit.Uint32
	residualAmount atomickit.Uint32
	bucketConfig
}
type bucketConfig = BucketConfig

type BucketRefillFunc func(need uint32) uint32

func (p *BucketState) TakeQuotaNoWait(max int64, scale uint32, refillFn BucketRefillFunc) int64 {
	maxAllocated := max / int64(scale)
	switch {
	case maxAllocated == 0:
		maxAllocated = 1
	case maxAllocated > math.MaxUint32:
		maxAllocated = math.MaxUint32
	}
	return int64(p.TakeQuotaNoScale(uint32(maxAllocated), refillFn)) * int64(scale)
}

func (p *BucketState) TakeQuotaNoScale(maxAllocated uint32, refillFn BucketRefillFunc) uint32 {
	if maxAllocated <= 0 {
		return 0
	}
	for {
		switch n := p.residualAmount.Load(); {
		case n >= maxAllocated:
			if p.residualAmount.CompareAndSwap(n, n-maxAllocated) {
				return maxAllocated
			}
		case n > 0:
			if p.residualAmount.CompareAndSwap(n, 0) {
				return n + p.takeFilled(maxAllocated-n, refillFn)
			}
		default:
			return p.takeFilled(maxAllocated, refillFn)
		}
	}
}

func (p *BucketState) quantizeCeiling(v uint32) uint32 {
	if v < p.Quantum {
		return p.Quantum
	}
	a := (uint64(v) + uint64(p.Quantum) - 1) / uint64(p.Quantum)
	a *= uint64(p.Quantum)
	if a > math.MaxUint32 {
		return uint32(a - uint64(p.Quantum))
	}
	return uint32(a)
}

func (p *BucketState) quantizeFloor(v uint32) uint32 {
	if v < p.Quantum {
		return 0
	}
	a := uint64(v) / uint64(p.Quantum)
	return uint32(a * uint64(p.Quantum))
}

func (p *BucketState) takeFilled(maxAllocated uint32, refillFn BucketRefillFunc) uint32 {
	allocate := p.takeQuantizedFilled(p.quantizeCeiling(maxAllocated), refillFn)
	if allocate <= maxAllocated {
		return allocate
	}
	p.residualAmount.Add(allocate - maxAllocated)
	return maxAllocated
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

		switch q := p.quantizeFloor(allocated + n); {
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
