package wormmap

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/keyset"
)

func NewLazyEntryMap(prefixLen uint8, expectedKeyCount int, hashSeed uint32, loaderFn EntryMapLoaderFunc) LazyEntryMap {
	switch {
	case prefixLen >= 32:
		panic("illegal value")
	case expectedKeyCount < 0:
		panic("illegal value")
	case expectedKeyCount > 0 && loaderFn == nil:
		panic("illegal value")
	case 1<<prefixLen > expectedKeyCount:
		panic("illegal value")
	}

	return LazyEntryMap{
		loader:           loaderFn,
		expectedKeyCount: expectedKeyCount,
		hashSeed:         hashSeed,
		prefixLen:        prefixLen,
	}
}

var _ keyset.KeyList = &LazyEntryMap{}

type EntryMapLoaderFunc func(bucketIndex int, addFn func(Entry) bool) (remainingBuckets int, fnResult bool)

type LazyEntryMap struct {
	entries map[Key]Entry
	loader  EntryMapLoaderFunc

	expectedKeyCount int
	hashSeed         uint32
	prefixLen        uint8
}

func (p *LazyEntryMap) EnumKeys(fn func(k keyset.Key) bool) bool {
	if p.loader != nil {
		panic("illegal state - map is partial")
	}
	for k, _ := range p.entries {
		if fn(k) {
			return true
		}
	}
	return false
}

func (p *LazyEntryMap) Count() int {
	if p.loader != nil {
		return p.expectedKeyCount
	}
	return len(p.entries)
}

func (p *LazyEntryMap) Contains(k keyset.Key) bool {
	if _, ok := p.entries[k]; ok {
		return true
	}
	if p.loader == nil {
		return false
	}

	// bucketIndex := k.GoMapHashWithSeed(p.hashSeed) & (1<<p.prefixLen - 1)
	// hasAdded := false
	//
	// if p.entries == nil {
	// 	p.entries = make(map[Key]Entry, p.expectedKeyCount)
	// }
	//
	// if remainingBuckets, _ := p.loader(int(bucketIndex), func(entry Entry) bool {
	// 	hasAdded = true
	// 	p.entries[entry.Key] = entry
	// 	return false
	// }); remainingBuckets == 0 {
	// 	p.loader = nil
	// }
	//
	// if hasAdded {
	// 	_, ok := p.entries[k]
	// 	return ok
	// }
	return false
}
