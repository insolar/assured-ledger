// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ratelimiter

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewPeriodManager(scale uint32) (*PeriodManager, func(uint)) {
	if scale == 0 {
		panic(throw.IllegalValue())
	}
	bm := &PeriodManager{amountScale: scale}
	bm.init()
	return bm, bm.nextPeriod
}

type PeriodManager struct {
	currentPeriod atomickit.Uint64
	signal        sync.Mutex
	amountScale   uint32
}

func (p *PeriodManager) init() {
	if p.currentPeriod.Load() != 0 {
		panic(throw.IllegalState())
	}
	p.signal.Lock()
	p.nextPeriod(1)
}

func (p *PeriodManager) nextPeriod(increment uint) {
	p.currentPeriod.Add(uint64(increment))
	p.signal.Unlock()
	p.signal.Lock()
}

func (p *PeriodManager) waitNextPeriod(x uint64) {
	if p.currentPeriod.Load() != x {
		return
	}
	p.signal.Lock()
	p.signal.Unlock()
}

func (p *PeriodManager) NewBucket(cfg bucketConfig) *Bucket {
	return NewBucket(p, cfg)
}

func (p *PeriodManager) NewRWBucket(cfgR, cfgW bucketConfig) *RWBucket {
	return NewRWBucket(p, cfgR, cfgW)
}
