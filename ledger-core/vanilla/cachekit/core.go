// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cachekit

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Index = int
type Age = int64
type GenNo = int

// Strategy defines behavior of cache Core
type Strategy interface {
	// TrimOnEachAddition is called once on Core creation. When result is true, then CanTrimEntries will be invoked on every addition.
	// When this value is false, cache capacity can be exceeded upto an average size of a generation page.
	TrimOnEachAddition() bool

	// CurrentAge should provide time marks when this Strategy needs to use time-based retention.
	CurrentAge() Age

	// GetAllocationPage is called once on Core creation to provide a size of entry pages (# of items).
	// It is recommended for cache implementation to use same size for paged storage.
	GetAllocationPage() int

	// NextGenerationCapacity is called on creation of every generation page.
	// Parameters are length and capacity of the previous generation page, or (-1, -1) for a first page,
	// It returns a capacity for a new page and a flag when fencing must be applied. A new generation page will be created when capacity is exhausted.
	// Fence - is a per-generation map to detect and ignore multiple touches for the same entry. It reduced cost to track frequency to once per generations
	// and is relevant for heavy load scenario only.
	NextGenerationCapacity(prevLen int, prevCap int) (pageSize int, useFence bool)

	// CanAdvanceGeneration is intended to provide custom logic to switch to a new generation page.
	// This logic can consider a number of hits and age of a current generation.
	// This function is called on a first update being added to a new generation page, and will be called again when provided limits are exhausted.
	// When (createGeneration) is (true) - a new generation page will be created and the given limits (hitCount) and (ageLimit) will be applied.
	// When (createGeneration) is (false) - the given limits (hitCount) and (ageLimit) will be applied to the current generation page.
	// NB! A new generation page will always be created when capacity is exhausted. See NextGenerationCapacity.
	CanAdvanceGeneration(curLen int, curCap int, hitRemains uint64) (createGeneration bool, hitLimit uint64, ageLimit Age)

	// CanTrimGenerations should return a number of LFU generation pages to be trimmed. This trim does NOT free cache entries, but compacts
	// generation pages by converting LFU into LRU entries.
	// It receives a total number of entries, a total number of LFU generation pages, and ages of the recent generation,
	// of the least-frequent generation (rarest) and of the oldest LRU generation.
	CanTrimGenerations(totalCount, freqGenCount int, recent, rarest, oldest Age) int
	// CanTrimEntries should return a number entries to be cleaned up from the cache.
	// It receives a total number of entries, and ages of the recent and of the the oldest LRU generation.
	CanTrimEntries(totalCount int, recent, oldest Age) int
}

type TrimFunc = func(trimmed []uint32) // these values are []Index

// NewCore creates a new core instance for the given Strategy and provides behavior that combines LFU and LRU strategies:
// * recent and the most frequently used entries are handled by LRU strategy. Accuracy of LRU logic depends on number*size of generation pages.
// * other entries are handled by LRU strategy.
// Cache implementation must provide a trim callback.
// The trim call back can be called during Add, Touch and Update operations.
// Provided instance can be copied, but only one copy can be used.
// Core uses entry pages to track entries (all pages of the same size) and generation pages (size can vary).
// Every cached entry gets a unique index, that can be reused after deletion / expiration of entries.
func NewCore(s Strategy, trimFn TrimFunc) Core {
	switch {
	case s == nil:
		panic(throw.IllegalValue())
	case trimFn == nil:
		panic(throw.IllegalValue())
	}

	return Core{
		alloc: newAllocationTracker(s.GetAllocationPage()),
		strat: s,
		trimFn: trimFn,
		trimEach: s.TrimOnEachAddition(),
	}
}

// Core provides all data management functions for cache implementations.
// This implementation is focused to minimize a number of links and memory overheads per entry.
// Behavior is regulated by provided Strategy.
// Core functions are thread-unsafe and must be protected by cache's implementation.
type Core struct {
	alloc    allocationTracker
	strat    Strategy
	trimFn   TrimFunc
	trimEach bool

	hitLimit uint64
	ageLimit Age

	recent *generation
	rarest *generation
	oldest *generation
}

func (p *Core) Allocated() int {
	return p.alloc.AllocatedCount()
}

func (p *Core) Occupied() int {
	return p.alloc.Count()
}

func (p *Core) Add() (Index, GenNo) {
	n := p.alloc.Add(1)
	return n, p.addToGen(n, true)
}

func (p *Core) Touch(index Index) GenNo {
	if _, ok := p.alloc.Inc(index); !ok {
		return -1
	}
	return p.addToGen(index, false)
}

func (p *Core) addToGen(index Index, added bool) GenNo {
	age := p.strat.CurrentAge()

	if p.recent == nil {
		p.recent = &generation{
			start:  age,
			end:    age,
		}
		p.recent.init(p.strat.NextGenerationCapacity(-1, -1))

		p.rarest = p.recent
		p.oldest = p.recent
		return p.recent.Add(index, age)
	}

	p.checkGen(age, added)
	return p.recent.Add(index, age)
}

func (p *Core) checkGen(age Age, added bool) {
	trimmed := false
	if p.trimEach && added {
		trimCount := p.strat.CanTrimEntries(p.alloc.Count(), // is already allocated
			p.recent.end, p.oldest.start)

		p.trimEntries(trimCount)
		trimmed = trimCount > 0
	}

	if p.hitLimit > 0 {
		p.hitLimit--
	}
	curLen, curCap := len(p.recent.access), cap(p.recent.access)

	switch {
	case curLen == curCap:
		p.hitLimit, p.ageLimit = 0, 0 // enforce call to CanAdvanceGeneration on next update
	case p.hitLimit == 0 || p.ageLimit <= age:
		createGen := false
		createGen, p.hitLimit, p.ageLimit = p.strat.CanAdvanceGeneration(curLen, curCap, p.hitLimit)
		if createGen {
			break
		}
		fallthrough
	default:
		return
	}

	newGen := &generation{
		genNo: p.recent.genNo + 1,
		start: age,
	}
	newGen.init(p.strat.NextGenerationCapacity(curLen, curCap))

	if newGen.start < p.recent.end {
		newGen.start = p.recent.end
	}
	newGen.end = newGen.start

	p.recent.next = newGen
	p.recent = newGen

	if !trimmed {
		totalCount := p.alloc.Count()

		trimCount := p.strat.CanTrimEntries(totalCount, p.recent.end, p.oldest.start)
		p.trimEntries(trimCount)

		if trimCount <= 0 {
			trimCount = p.strat.CanTrimGenerations(totalCount, p.recent.genNo - p.rarest.genNo + 1,
				p.recent.end, p.rarest.start, p.oldest.start)

			p.trimGenerations(trimCount)
		}
	}
}

func (p *Core) trimEntries(count int) {
	for count > 0 && p.oldest != p.rarest {
		count = p.oldest.trimOldest(p, count)
		if len(p.oldest.access) == 0 {
			p.oldest = p.oldest.next
		}
	}

	for count > 0 && p.recent != p.rarest {
		count = p.rarest.trimRarest(p, count)
		if len(p.rarest.access) == 0 {
			p.rarest = p.rarest.next
			p.oldest = p.rarest
		}
	}
}

func (p *Core) trimGenerations(count int) {
	prev := p.oldest
	if prev == p.rarest || prev.next != p.rarest {
		prev = nil
	}

	for ;count > 0 && p.recent != p.rarest; count-- {
		p.rarest.trimGeneration(p, prev)

		if len(p.rarest.access) == 0 {
			p.rarest = p.rarest.next
			if prev != nil {
				prev.next = p.rarest
			}
		} else {
			prev = p.rarest
			p.rarest = p.rarest.next
		}
	}
}

func (p *Core) ResetGenerations() {

}

/*******************************************/

type generation struct {
	access []uint32 // Index
	fence  map[uint32]struct{}
	start  Age
	end    Age
	next   *generation
	genNo  GenNo
}

func (p *generation) init(capacity int, loadFence bool) {
	p.access = make([]uint32, 0, capacity)
	if loadFence {
		p.fence = make(map[uint32]struct{}, capacity)
	}
}

func (p *generation) Add(index Index, age Age) GenNo {
	if p.end < age {
		p.end = age
	}

	idx := uint32(index)
	if p.fence != nil {
		if _, ok := p.fence[idx]; !ok {
			p.fence[idx] = struct{}{}
			p.access = append(p.access, idx)
		}
	} else {
		if n := len(p.access); n == 0 || p.access[n - 1] != idx {
			p.access = append(p.access, idx)
		}
	}

	return p.genNo
}

func (p *generation) trimOldest(c *Core, count int) int {
	indices := p.access
	i, j := 0, 0
	for _, idx := range p.access {
		i++
		if v, ok := c.alloc.Get(Index(idx)); !ok || v > 0 {
			continue
		}
		c.alloc.Delete(Index(idx))

		indices[j] = idx
		j++

		count--
		if count == 0 {
			break
		}
	}

	p.access = p.access[i:]
	if j > 0 {
		c.trimFn(indices[:j:j])
	}

	return count
}

func (p *generation) trimRarest(c *Core, count int) int {
	indices := p.access
	i, j := 0, 0
	for _, idx := range p.access {
		i++
		switch v, changed, ok := c.alloc.Dec(Index(idx)); {
		case !ok:
			// was removed
			continue
		case v > 0:
			// not yet rare
			continue
		case !changed:
			// is already among the oldest
			continue
		}
		c.alloc.Delete(Index(idx))

		indices[j] = idx
		j++
		count--
		if count == 0 {
			break
		}
	}

	p.access = p.access[i:]
	if j > 0 {
		c.trimFn(indices[:j:j])
	}

	return count
}

func (p *generation) trimGeneration(c *Core, prev *generation) {
	indices := p.access
	j := 0
	for _, idx := range p.access {
		switch v, changed, ok := c.alloc.Dec(Index(idx)); {
		case !ok:
			// was removed
			continue
		case v > 0:
			// not yet rare
			continue
		case !changed:
			// is already among the oldest
			continue
		}

		indices[j] = idx
		j++
	}

	if j == 0 {
		p.access = nil
		return
	}
	p.access = p.access[:j]
	if prev == nil {
		return
	}

	switch prevN := len(prev.access); {
	case prevN == 0:
		return
	case prevN < j && prevN + j <= cap(p.access):
		p.access = append(p.access, prev.access...)
		prev.access = nil
	case prevN + j <= cap(prev.access):
		prev.access = append(prev.access, p.access...)
		p.access = nil
	}
}

