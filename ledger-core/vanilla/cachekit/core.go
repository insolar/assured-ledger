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

	// AllocationPageSize is called once on Core creation to provide a size of entry pages (# of items).
	// It is recommended for cache implementation to use same size for paged storage.
	AllocationPageSize() int

	// NextGenerationCapacity is called on creation of every generation page.
	// Parameters are length and capacity of the previous generation page, or (-1, -1) for a first page,
	// It returns a capacity for a new page and a flag when fencing must be applied. A new generation page will be created when capacity is exhausted.
	// Fence - is a per-generation map to detect and ignore multiple touches for the same entry. It reduced cost to track frequency to once per generations
	// and is relevant for heavy load scenario only.
	NextGenerationCapacity(prevLen int, prevCap int) (pageSize int, useFence bool)
	// InitGenerationCapacity is an analogue of NextGenerationCapacity but when a first page is created.
	InitGenerationCapacity() (pageSize int, useFence bool)

	// CanAdvanceGeneration is intended to provide custom logic to switch to a new generation page.
	// This logic can consider a number of hits and age of a current generation.
	// This function is called on a first update being added to a new generation page, and will be called again when provided limits are exhausted.
	// When (createGeneration) is (true) - a new generation page will be created and the given limits (hitCount) and (ageLimit) will be applied.
	// When (createGeneration) is (false) - the given limits (hitCount) and (ageLimit) will be applied to the current generation page.
	// NB! A new generation page will always be created when capacity is exhausted. See NextGenerationCapacity.
	CanAdvanceGeneration(curLen int, curCap int, hitRemains uint64, start, end Age) (createGeneration bool, hitLimit uint64, ageLimit Age)

	// InitialAdvanceLimits is an analogue of CanAdvanceGeneration applied when a generation is created.
	// Age given as (start) is the age of the first record to be added.
	InitialAdvanceLimits(curCap int, start Age) (hitLimit uint64, ageLimit Age)

	// CanTrimGenerations should return a number of LFU generation pages to be trimmed. This trim does NOT free cache entries, but compacts
	// generation pages by converting LFU into LRU entries.
	// It receives a total number of entries, a total number of LFU generation pages, and ages of the recent generation,
	// of the least-frequent generation (rarest) and of the oldest LRU generation.
	// Zero or negative result will skip trimming.
	CanTrimGenerations(totalCount, freqGenCount int, recent, rarest, oldest Age) int

	// CanTrimEntries should return a number entries to be cleaned up from the cache.
	// It receives a total number of entries, and ages of the recent and of the the oldest LRU generation.
	// Zero or negative result will skip trimming.
	// Trimming, initiated by positive result of CanTrimEntries may prevent CanTrimGenerations to be called.
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
		alloc: newAllocationTracker(s.AllocationPageSize()),
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
	n := p.alloc.Add(2) // +1 is for tracking oldest
	return n, p.addToGen(n, true, false)
}

// Delete can ONLY be called once per index, otherwise counting will be broken
func (p *Core) Delete(idx Index) {
	p.alloc.Dec(idx) // this will remove +1 for oldest tracking
	                 // so the entry will be removed at the "rarest" generation
}

func (p *Core) Touch(index Index) GenNo {
	_, overflow, ok := p.alloc.Inc(index)
	if !ok {
		return -1
	}
	return p.addToGen(index, false, overflow)
}

func (p *Core) addToGen(index Index, addedEntry, overflow bool) GenNo {
	age := p.strat.CurrentAge()
	if p.recent == nil {
		p.recent = &generation{
			start:  age,
			end:    age,
		}
		p.recent.init(p.strat.InitGenerationCapacity())

		p.rarest = p.recent
		p.oldest = p.recent
		genNo, _ := p.recent.Add(index, age) // first entry will always be added
		return genNo
	}

	p.useOrAdvanceGen(age, addedEntry)

	genNo, addedEvent := p.recent.Add(index, age)
	switch {
	case addedEvent:
	case addedEntry:
		panic(throw.Impossible())
	case !overflow:
		_, _ = p.alloc.Dec(index)
	}
	return genNo
}

func (p *Core) useOrAdvanceGen(age Age, added bool) {
	trimmedEntries, trimmedGens := false, false

	if p.trimEach && added {
		trimCount := p.strat.CanTrimEntries(p.alloc.Count(), // is already allocated
			p.recent.end, p.oldest.start)

		trimmedEntries = trimCount > 0
		trimmedGens = p.trimEntriesAndGenerations(trimCount)
	}

	if p.hitLimit > 0 {
		p.hitLimit--
	}
	curLen, curCap := len(p.recent.access), cap(p.recent.access)

	switch {
	case curLen == curCap:
	case p.hitLimit == 0 || p.ageLimit <= age:
		createGen := false
		createGen, p.hitLimit, p.ageLimit = p.strat.CanAdvanceGeneration(curLen, curCap, p.hitLimit, p.recent.start, p.recent.end)
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
		end: age,
	}
	newGen.init(p.strat.NextGenerationCapacity(curLen, curCap))

	p.recent.next = newGen
	p.recent = newGen

	p.hitLimit, p.ageLimit = p.strat.InitialAdvanceLimits(cap(newGen.access), age)

	if !trimmedEntries {
		trimCount := p.strat.CanTrimEntries(p.alloc.Count(), p.recent.end, p.oldest.start)
		trimmedGens = p.trimEntriesAndGenerations(trimCount)
	}

	if !trimmedGens {
		trimGenCount := p.strat.CanTrimGenerations(p.alloc.Count(), p.recent.genNo - p.rarest.genNo + 1,
				p.recent.end, p.rarest.start, p.oldest.start)
		p.trimGenerations(trimGenCount)
	}
}

func (p *Core) trimEntriesAndGenerations(count int) bool {
	for ; count > 0 && p.oldest != p.rarest; p.oldest = p.oldest.next {
		count = p.oldest.trimOldest(p, count)
	}

	if count <= 0 {
		return false
	}

	for ; count > 0 && p.recent != p.rarest; p.rarest = p.rarest.next {
		count = p.rarest.trimRarest(p, count)
		if len(p.rarest.access) != 0 {
			break
		}
		p.oldest = p.rarest
	}

	if count <= 0 {
		return true
	}

	count = p.recent.trimRarest(p, count)

	if count > 0 && p.alloc.Count() > 0 {
		panic(throw.Impossible())
	}

	return true
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

func (p *generation) Add(index Index, age Age) (GenNo, bool) {
	idx := uint32(index)
	n := len(p.access)

	switch {
	case n == 0:
		p.access = append(p.access, idx)
		if p.fence != nil {
			p.fence[idx] = struct{}{}
		}
		return p.genNo, true
	case p.end < age:
		p.end = age
	}

	switch {
	case p.fence != nil:
		if _, ok := p.fence[idx]; !ok {
			p.fence[idx] = struct{}{}
			p.access = append(p.access, idx)
			return p.genNo, true
		}
	case p.access[n - 1] != idx:
		p.access = append(p.access, idx)
		return p.genNo, true
	}

	return p.genNo, false
}

func (p *generation) trimOldest(c *Core, count int) int {
	indices := p.access
	i, j := 0, 0
	for _, idx := range p.access {
		i++
		switch v, ok := c.alloc.Dec(Index(idx)); {
		case !ok:
			// was removed
			continue
		case v > 0:
			// it seems to be revived
			// so it will be back in a while
			// restore the counter
			c.alloc.Inc(Index(idx))
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
		switch v, ok := c.alloc.Dec(Index(idx)); {
		case !ok:
			// was removed
			continue
		case v > 1:
			// it has other events
			// so it will be back in a while
			// restore the counter
			c.alloc.Inc(Index(idx))
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
		switch v, ok := c.alloc.Dec(Index(idx)); {
		case !ok:
			// was removed
			continue
		case v > 1:
			// not yet rare
			continue
		case v == 0:
			// it was explicitly removed
			// so do not pass it further
			c.alloc.Delete(Index(idx))
			continue
		}
		// v == 1 - leave for FIFO removal
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
	case prevN + j <= cap(p.access):
		prev.access = append(p.access, prev.access...)
	case prevN + j <= cap(prev.access):
		combined := prev.access[:prevN + j] // expand within capacity
		copy(combined[j:], prev.access) // move older records to the end
		copy(combined, p.access[:j]) // put newer records to the front
	}

	prev.start = p.start
	p.access = nil // enable this generation to be deleted
}

