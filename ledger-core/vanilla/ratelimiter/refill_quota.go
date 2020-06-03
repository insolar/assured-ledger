// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ratelimiter

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RefillQuota struct {
	refillQuota atomickit.Uint32 // used by child bucket to avoid exhaustion of parent
}

func (p *RefillQuota) Refill(needed uint32, periods uint64, state *BucketState, parentRefillFn func(uint32) uint32) uint32 {

	q := p.getParentQuota(needed)
	if q < needed && periods > 0 {
		allowed := state.PeriodsToRefill(periods)
		if lacks := needed - q; lacks >= allowed {
			q += allowed
		} else {
			q = needed
			p.addToRefillQuota(allowed-lacks, state.MaxAmount)
		}
	}

	allocated := parentRefillFn(q)
	if q > allocated {
		panic(throw.Impossible())
	}
	return allocated

}

func (p *RefillQuota) getParentQuota(needed uint32) uint32 {
	for {
		switch q := p.refillQuota.Load(); {
		case q > needed:
			if !p.refillQuota.CompareAndSwap(q, q-needed) {
				continue
			}
			return needed
		case q > 0:
			if !p.refillQuota.CompareAndSwap(q, 0) {
				continue
			}
			return q
		}
		return 0
	}
}

func (p *RefillQuota) addToRefillQuota(x, max uint32) {
	for {
		v := p.refillQuota.Load()
		v2 := v + x
		if v2 > max || v2 < v /* overflow */ {
			v2 = max
		}
		if p.refillQuota.CompareAndSwap(v, v2) {
			return
		}
	}
}
