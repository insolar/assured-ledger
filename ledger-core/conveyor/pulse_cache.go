// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type BeatData = managed.BeatData

// Cache that keeps (1) a PD younger than minRange (2) PD touched less than accessRotations ago.
// Safe for concurrent access.
// WARNING! Cache size is not directly limited.
// TODO PLAT-19 eviction function is not efficient for 100+ PDs and/or accessRotations > 10
type PulseDataCache struct {
	pdm       *PulseDataManager
	mutex     sync.RWMutex
	minRange  uint32
	cache     map[pulse.Number]*cacheEntry // enable reuse for SMs in Antique slot
	access    []map[pulse.Number]struct{}
	accessIdx int
}

func (p *PulseDataCache) Init(pdm *PulseDataManager, minRange uint32, accessRotations int) {
	if p.access != nil {
		panic("illegal state")
	}
	if accessRotations < 0 {
		panic("illegal value")
	}
	if pdm == nil {
		panic("illegal value")
	}
	p.pdm = pdm
	p.minRange = minRange
	p.access = make([]map[pulse.Number]struct{}, accessRotations)
	p.access[0] = make(map[pulse.Number]struct{})
}

func (p *PulseDataCache) GetMinRange() uint32 {
	return p.minRange
}

func (p *PulseDataCache) EvictAndRotate(currentPN pulse.Number) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p._evict(currentPN)
	p._rotate()
	p._touch(currentPN) // to retain current PD at corner cases
}

func (p *PulseDataCache) EvictNoRotate(currentPN pulse.Number) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p._evict(currentPN)
}

func (p *PulseDataCache) Rotate() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p._rotate()
}

func (p *PulseDataCache) _evict(currentPN pulse.Number) {
	cpn := currentPN.AsUint32()
	if uint32(pulse.MinTimePulse)+p.minRange >= cpn {
		// must keep all
		return
	}
	minPN := pulse.OfUint32(cpn - p.minRange)

outer:
	for pn := range p.cache {
		if pn >= minPN {
			continue
		}
		for _, am := range p.access {
			if _, ok := am[pn]; ok {
				continue outer
			}
		}

		delete(p.cache, pn)
	}
}

type accessState uint8

const (
	miss accessState = iota
	hit
	hitNoTouch
)

func (p *PulseDataCache) getNoTouch(pn pulse.Number) (*cacheEntry, accessState) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p._getRO(pn)
}

func (p *PulseDataCache) _getRO(pn pulse.Number) (*cacheEntry, accessState) {
	if p.cache != nil {
		if pd, ok := p.cache[pn]; ok {
			if p._wasTouched(pn) {
				return pd, hit
			}
			return pd, hitNoTouch
		}
	}
	return nil, miss
}

func (p *PulseDataCache) getAndTouch(pn pulse.Number) *cacheEntry {
	ce, m := p.getNoTouch(pn)
	switch {
	case m != hitNoTouch:
		return ce
	case ce == nil:
		panic(throw.IllegalState())
	}

	p.mutex.Lock()
	p._touch(pn)
	p.mutex.Unlock()

	return ce
}

func (p *PulseDataCache) getPulseSlot(pn pulse.Number) *PulseSlot {
	if ce := p.getAndTouch(pn); ce != nil {
		return &ce.ps
	}
	return nil
}

func (p *PulseDataCache) Get(pn pulse.Number) BeatData {
	return p.getAndTouch(pn)._cacheData()
}

func (p *PulseDataCache) Check(pn pulse.Number) BeatData {
	pr, _ := p.getNoTouch(pn)
	return pr._cacheData()
}

func (p *PulseDataCache) Contains(pn pulse.Number) bool {
	_, m := p.getNoTouch(pn)
	return m != miss
}

func (p *PulseDataCache) Touch(pn pulse.Number) bool {
	switch _, m := p.getNoTouch(pn); m {
	case miss:
		return false
	case hit:
		return true
	}
	p.mutex.Lock()
	p._touch(pn)
	p.mutex.Unlock()
	return true
}

func (p *PulseDataCache) Put(pd BeatData) {
	if pd.Range == nil {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	pns := make([]pulse.Number, 0, 4)
	var ce *cacheEntry
	pd.Range.EnumNonArticulatedNumbers(func(pn pulse.Number, _, _ uint16) bool {
		switch ece, m := p._getRO(pn); {
		case m == miss:
			//
		case !pd.Range.Equal(ece.pd.Range):
			panic(throw.New("mismatched pulse.Range", struct{ PN pulse.Number}{pn}))
		case m == hitNoTouch:
			//
		case ce == nil:
			ce = ece
			return false
		case ce != ece:
			panic(throw.New("inconsistent pulse.Range cache", struct{ PN pulse.Number}{pn}))
		default:
			return false
		}

		pns = append(pns, pn)
		return false
	})

	switch {
	case ce == nil:
		if len(pns) == 0 {
			panic(throw.IllegalValue())
		}
		ce = newCacheEntry(p.pdm, pd)
	case len(pns) == 0:
		return
	}

	if p.cache == nil {
		p.cache = make(map[pulse.Number]*cacheEntry)
	}

	for _, pn := range pns {
		p.cache[pn] = ce
		p._touch(pn)
	}
}

func (p *PulseDataCache) _wasTouched(pn pulse.Number) bool {
	_, ok := p.access[p.accessIdx][pn]
	return ok
}

func (p *PulseDataCache) _touch(pn pulse.Number) {
	p.access[p.accessIdx][pn] = struct{}{}
}

func (p *PulseDataCache) _rotate() {
	p.accessIdx++
	if p.accessIdx >= len(p.access) {
		p.accessIdx = 0
	}
	switch m := p.access[p.accessIdx]; {
	case m == nil:
		p.access[p.accessIdx] = make(map[pulse.Number]struct{})
	case len(m) == 0:
		// reuse
	default:
		p.access[p.accessIdx] = make(map[pulse.Number]struct{}, len(m))
	}
}

func newCacheEntry(pdm *PulseDataManager, pd BeatData) *cacheEntry {
	ce := &cacheEntry{pd: pd, ps: PulseSlot{pulseManager: pdm}}
	ce.ps.pulseData = ce
	return ce
}

type cacheEntry struct {
	pd BeatData
	ps PulseSlot
}

func (p cacheEntry) PulseData() pulse.Data {
	return p.pd.Range.RightBoundData()
}

func (p cacheEntry) BeatData() (BeatData, PulseSlotState) {
	return p.pd, Antique
}

func (p cacheEntry) MakePresent(BeatData, time.Time) {
	panic(throw.IllegalState())
}

func (p *cacheEntry) PulseStartedAt() time.Time {
	panic(throw.IllegalState())
}

func (p cacheEntry) MakePast() {
	panic(throw.IllegalState())
}

func (p *cacheEntry) _cacheData() BeatData {
	if p == nil {
		return BeatData{}
	}
	return p.pd
}
