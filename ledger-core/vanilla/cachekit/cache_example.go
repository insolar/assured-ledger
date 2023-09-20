package cachekit

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Key = uint64
type Value = string

const zeroValue = ""

/*
	UintCache is an example/template implementation of a cache that uses the Core.

	To create a new cache type - copy this file, rename struct, set Key and Value type aliases accordingly, then
	set zeroValue constant in accordance with Value type.


 */

func NewUintCache(cs Strategy) *UintCache {
	c := &UintCache{
		keys: map[Key]Index{},
		values: [][]uintEntry { make([]uintEntry, cs.AllocationPageSize()) },
	}
	c.core = NewCore(cs, c.trimBatch)

	return c
}

// UintCache is an example/template implementation of a cache that uses the Core.
type UintCache struct {
	values [][]uintEntry
	keys   map[Key]Index

	core   Core
}

type uintEntry struct {
	key Key
	value Value
}

// Allocated returns a total number of cache entries allocated, but some of them may be unused.
// NB! Cache can only grow.
func (p *UintCache) Allocated() int {
	return len(p.values) * cap(p.values[0])
}

// Occupied returns a number of added / available cache entries.
func (p *UintCache) Occupied() int {
	return len(p.keys)
}

// Put adds value with the given key.
// If key was already added, then cached value remains unchanged and the function returns (false).
// Access to the key is always updated.
func (p *UintCache) Put(key Key, value Value) bool {
	idx, ok := p.keys[key]
	if ok {
		p.core.Touch(idx)
		return false
	}
	idx, _ = p.core.Add()
	p.keys[key] = idx
	p.putEntry(idx, uintEntry{key, value})
	return true
}

// Replace adds or replaces value with the given key. If key was already added, then cached value is updated and the function returns (false).
// Access to the key is always updated.
func (p *UintCache) Replace(key Key, value Value) bool {
	idx, ok := p.keys[key]
	if ok {
		p.core.Touch(idx)
		p.putEntry(idx, uintEntry{key, value})
		return false
	}
	idx, _ = p.core.Add()
	p.keys[key] = idx
	p.putEntry(idx, uintEntry{key, value})
	return true
}

// Get returns value and presence flag for the given key.
// Access to the key is updated when key exists.
func (p *UintCache) Get(key Key) (Value, bool) {
	idx, ok := p.keys[key]
	if !ok {
		return zeroValue, false
	}
	p.core.Touch(idx)
	ce := p.getEntry(idx)
	return ce.value, true
}

// Peek returns value and presence flag for the given key.
// Access to the key is not updated.
func (p *UintCache) Peek(key Key) (Value, bool) {
	idx, ok := p.keys[key]
	if !ok {
		return zeroValue, false
	}
	ce := p.getEntry(idx)
	return ce.value, true
}

// Contains returns (true) when the key is present.
// Access to the key is not updated.
func (p *UintCache) Contains(key Key) bool {
	_, ok := p.keys[key]
	return ok
}

// Delete removes key and zero out relevant value. Returns (false) when key wasn't present.
// Access to the key is not updated. Cache entry will become unavailable, but will only be freed after relevant expiry / eviction.
func (p *UintCache) Delete(key Key) bool {
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
	*ce = uintEntry{}
	return true
}

func (p *UintCache) putEntry(idx Index, entry uintEntry) {
	pgSize := cap(p.values[0])
	pgN := idx / pgSize

	if pgN == len(p.values) {
		p.values = append(p.values, make([]uintEntry, pgSize))
	}
	p.values[pgN][idx % pgSize] = entry
}

func (p *UintCache) getEntry(idx Index) *uintEntry {
	pgSize := cap(p.values[0])
	return &p.values[idx / pgSize][idx % pgSize]
}

func (p *UintCache) trimBatch(trimmed []uint32) {
	for _, idx := range trimmed {
		ce := p.getEntry(Index(idx))
		delete(p.keys, ce.key)
		*ce = uintEntry{}
	}
}

