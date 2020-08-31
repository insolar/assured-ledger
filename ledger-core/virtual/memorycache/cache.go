// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memorycache

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cachekit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

type Key = reference.Global
type Value = descriptor.Object

func NewMemoryCache() *LRUMemoryCache {
	cs := cacheStrategy{
		pgSize:   1024,
		maxTotal: 20,
	}
	c := &LRUMemoryCache{
		keys:   map[Key]cachekit.Index{},
		values: [][]valueEntry{make([]valueEntry, cs.AllocationPageSize())},
	}
	c.core = cachekit.NewCore(cs, c.trimBatch)

	return c
}

type LRUMemoryCache struct {
	values [][]valueEntry
	keys   map[Key]cachekit.Index

	core cachekit.Core
}

type valueEntry struct {
	key   Key
	value Value
}

// Allocated returns a total number of cache entries allocated, but some of them may be unused.
// NB! Cache can only grow.
func (p *LRUMemoryCache) Allocated() int {
	return len(p.values) * cap(p.values[0])
}

// Occupied returns a number of added / available cache entries.
func (p *LRUMemoryCache) Occupied() int {
	return len(p.keys)
}

// Put adds value with the given key.
// If key was already added, then cached value remains unchanged and the function returns (false).
// Access to the key is always updated.
func (p *LRUMemoryCache) Put(key Key, value Value) bool {
	idx, ok := p.keys[key]
	if ok {
		p.core.Touch(idx)
		return false
	}
	idx, _ = p.core.Add()
	p.keys[key] = idx
	p.putEntry(idx, valueEntry{key, value})
	return true
}

// Replace adds or replaces value with the given key. If key was already added, then cached value is updated and the function returns (false).
// Access to the key is always updated.
func (p *LRUMemoryCache) Replace(key Key, value Value) bool {
	idx, ok := p.keys[key]
	if ok {
		p.core.Touch(idx)
		p.putEntry(idx, valueEntry{key, value})
		return false
	}
	idx, _ = p.core.Add()
	p.keys[key] = idx
	p.putEntry(idx, valueEntry{key, value})
	return true
}

// Get returns value and presence flag for the given key.
// Access to the key is updated when key exists.
func (p *LRUMemoryCache) Get(key Key) (Value, bool) {
	idx, ok := p.keys[key]
	if !ok {
		return nil, false
	}
	p.core.Touch(idx)
	ce := p.getEntry(idx)
	return ce.value, true
}

// Peek returns value and presence flag for the given key.
// Access to the key is not updated.
func (p *LRUMemoryCache) Peek(key Key) (Value, bool) {
	idx, ok := p.keys[key]
	if !ok {
		return nil, false
	}
	ce := p.getEntry(idx)
	return ce.value, true
}

// Contains returns (true) when the key is present.
// Access to the key is not updated.
func (p *LRUMemoryCache) Contains(key Key) bool {
	_, ok := p.keys[key]
	return ok
}

// Delete removes key and zero out relevant value. Returns (false) when key wasn't present.
// Access to the key is not updated. Cache entry will become unavailable, but will only be freed after relevant expiry / eviction.
func (p *LRUMemoryCache) Delete(key Key) bool {
	idx, ok := p.keys[key]
	if !ok {
		return false
	}
	ce := p.getEntry(idx)
	if ce.key != key {
		panic(throw.IllegalState())
	}
	p.core.Delete(idx)
	delete(p.keys, ce.key)
	*ce = valueEntry{}
	return true
}

func (p *LRUMemoryCache) putEntry(idx cachekit.Index, entry valueEntry) {
	pgSize := cap(p.values[0])
	pgN := idx / pgSize

	if pgN == len(p.values) {
		p.values = append(p.values, make([]valueEntry, pgSize))
	}
	p.values[pgN][idx%pgSize] = entry
}

func (p *LRUMemoryCache) getEntry(idx cachekit.Index) *valueEntry {
	pgSize := cap(p.values[0])
	return &p.values[idx/pgSize][idx%pgSize]
}

func (p *LRUMemoryCache) trimBatch(trimmed []uint32) {
	for _, idx := range trimmed {
		ce := p.getEntry(cachekit.Index(idx))
		delete(p.keys, ce.key)
		*ce = valueEntry{}
	}
}

type cacheStrategy struct {
	pgSize, maxTotal int
}

func (v cacheStrategy) TrimOnEachAddition() bool {
	return false
}

func (v cacheStrategy) CurrentAge() cachekit.Age {
	return 0
}

func (v cacheStrategy) AllocationPageSize() int {
	return v.pgSize
}

func (v cacheStrategy) InitGenerationCapacity() (pageSize int, useFence bool) {
	return v.pgSize, false
}

func (v cacheStrategy) NextGenerationCapacity(prevLen int, prevCap int) (int, bool) {
	return v.pgSize, false
}

func (v cacheStrategy) CanTrimGenerations(totalCount, freqGenCount int, recent, rarest, oldest cachekit.Age) int {
	return 0
}

func (v cacheStrategy) CanTrimEntries(totalCount int, recent, oldest cachekit.Age) int {
	return totalCount - v.maxTotal
}

func (v cacheStrategy) CanAdvanceGeneration(curLen int, curCap int, hitRemains uint64, start, end cachekit.Age) (createGeneration bool, hitLimit uint64, ageLimit cachekit.Age) {
	return false, math.MaxUint64, math.MaxInt64
}

func (v cacheStrategy) InitialAdvanceLimits(curCap int, start cachekit.Age) (hitLimit uint64, ageLimit cachekit.Age) {
	return math.MaxUint64, math.MaxInt64
}
