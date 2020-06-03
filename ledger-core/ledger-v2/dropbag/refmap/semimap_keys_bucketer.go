// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package refmap

import (
	"math/bits"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
)

type BucketSortFunc func(i, j *reference.Local) bool

type WriteBucketerConfig struct {
	ExpectedPerBucket int
	UsePrimes         bool
}

func NewWriteBucketer(keyMap *UpdateableKeyMap, valueCount int, config WriteBucketerConfig) WriteBucketer {

	keyCount := keyMap.InternedKeyCount()
	switch {
	case keyCount <= 0:
		panic("illegal value")
	case valueCount <= 0:
		panic("illegal value")
	case keyCount < valueCount:
		keyCount = valueCount
	}

	bucketIdxMask := uint32(0)
	bucketCount := (keyCount + config.ExpectedPerBucket - 1) / config.ExpectedPerBucket
	if config.UsePrimes {
		if primes := args.OddPrimes(); primes.Max() >= bucketCount {
			bucketCount = primes.Ceiling(bucketCount)
		} else {
			bitsPerBucket := uint8(bits.Len(uint(bucketCount - 1)))
			bucketCount = 1<<bitsPerBucket - 1 // can be a Mersenne number / prime
		}
	} else {
		bitsPerBucket := uint8(bits.Len(uint(bucketCount - 1)))
		bucketCount = 1 << bitsPerBucket
		bucketIdxMask = uint32(bucketCount - 1) // can use bitmask for indexing
	}

	bucketSize := (keyCount + bucketCount - 1) / bucketCount
	if keyCountExcess, minExcess := bucketCount*bucketSize-keyCount, keyCount; keyCountExcess < minExcess {
		bucketSize += (minExcess - keyCountExcess + bucketCount - 1) / bucketCount
	}

	return WriteBucketer{
		keyMap,
		uint32(bucketSize),
		uint32(config.ExpectedPerBucket),
		bucketIdxMask,
		make([]writeBucket, bucketCount),
		make([]ValueSelectorLocator, bucketSize*bucketCount),
		nil,
	}
}

type WriteBucketer struct {
	keyMap *UpdateableKeyMap

	bucketSize      uint32
	bucketKeySize   uint32
	bucketIdxMask   uint32
	bucketCounts    []writeBucket
	bucketEntries   []ValueSelectorLocator
	overflowEntries map[uint32][]ValueSelectorLocator
}

type writeBucket struct {
	//keysL0     map[uint32]struct{}
	entryCount uint32
}

type ValueSelectorLocator struct {
	selector ValueSelector
	locator  ValueLocator
}

func (p *WriteBucketer) AddValue(selector ValueSelector, locator ValueLocator) {
	mapBucket := p.keyMap.getBucket(selector.LocalId)
	if mapBucket.state == 0 {
		panic("illegal state")
	}
	writeBucketNo := mapBucket.refHash
	if p.bucketIdxMask != 0 {
		writeBucketNo &= p.bucketIdxMask
	} else {
		writeBucketNo %= uint32(len(p.bucketCounts))
	}

	bucket := &p.bucketCounts[writeBucketNo]
	n := bucket.entryCount
	if n+1 < n {
		panic("overflow")
	}
	bucket.entryCount = n + 1

	value := ValueSelectorLocator{selector, locator}
	switch {
	case n < p.bucketSize:
		selectorOfs := writeBucketNo*p.bucketSize + n
		p.bucketEntries[selectorOfs] = value
	case n == p.bucketSize:
		if p.overflowEntries == nil {
			p.overflowEntries = make(map[uint32][]ValueSelectorLocator)
		}
		p.overflowEntries[writeBucketNo] = []ValueSelectorLocator{value}
	default:
		p.overflowEntries[writeBucketNo] = append(p.overflowEntries[writeBucketNo], value)
	}
}

type BucketResolveFunc func(uint32) (*reference.Local, BucketState)

func (p *WriteBucketer) GetBucketed(bucketNo int) []ValueSelectorLocator {
	var bucketSlice []ValueSelectorLocator

	switch n := p.bucketCounts[bucketNo].entryCount; {
	case n == 0:
		return nil
	case n <= p.bucketSize:
		bucketBase := uint32(bucketNo) * p.bucketSize
		bucketSlice = p.bucketEntries[bucketBase : bucketBase+n]
		if n == 1 {
			return bucketSlice
		}
	default:
		bucketBase := uint32(bucketNo) * p.bucketSize
		bucketSlice = append([]ValueSelectorLocator(nil), p.bucketEntries[bucketBase:bucketBase+p.bucketSize]...)
		bucketSlice = append(bucketSlice, p.overflowEntries[uint32(bucketNo)]...)
	}
	return bucketSlice
}

func (p *WriteBucketer) AdjustedBucketSize() int {
	return int(p.bucketSize)
}

func (p *WriteBucketer) BucketCount() int {
	return len(p.bucketCounts)
}
