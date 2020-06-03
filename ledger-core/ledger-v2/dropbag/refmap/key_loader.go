// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package refmap

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/args"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/unsafekit"
)

func newBucketKeyLoader(chunks []longbits.ByteString) bucketKeyLoader {
	return bucketKeyLoader{chunks: chunks}
}

type bucketKeyLoader struct {
	chunks    []longbits.ByteString
	lastChunk longbits.ByteString
}

func (p *bucketKeyLoader) nextChunk() bool {
	switch {
	case len(p.lastChunk) > 0:
		return true
	case len(p.chunks) == 0:
		return false
	default:
		p.lastChunk = p.chunks[0]
		if len(p.lastChunk) == 0 {
			panic("illegal state")
		}
		p.chunks = p.chunks[1:]
		return true
	}
}

func (p *bucketKeyLoader) loadKeysL0AsMap(expectedL0Count, expectedL1Count int) (bigBucketMap, error) {
	keys := make(bigBucketMap, expectedL0Count)
	countL1, countNV := 0, 0

	switch err := p._loadKeys(expectedL0Count, func(_ int, dataChunk longbits.ByteString) error {
		keyBatch := bucketKeyTypeSlice.Unwrap(dataChunk).([]bucketKey)
		for _, bk := range keyBatch {
			key := unsafekit.WrapLocalRef(&bk.local)
			keys[key] = bk.value
			if bk.value.isLeaf() {
				countNV++
			} else {
				countL1++
			}
		}
		return nil
	}); {
	case err != nil:
		return nil, err
	case countL1 != expectedL1Count:
		panic("illegal state") // TODO return error
	case countL1+countNV != expectedL0Count:
		panic("illegal state") // TODO return error
	case len(keys) != expectedL0Count:
		panic("illegal state") // TODO return error
	}

	return keys, nil
}

func (p *bucketKeyLoader) loadKeys(expectedCount int) ([][]bucketKey, error) {
	var keys [][]bucketKey

	switch err := p._loadKeys(expectedCount, func(remainingCount int, dataChunk longbits.ByteString) error {
		if remainingCount == 0 && len(keys) == 0 {
			keys = make([][]bucketKey, 0, 1) // avoid waste
		}
		keyBatch := bucketKeyTypeSlice.Unwrap(dataChunk).([]bucketKey)
		keys = append(keys, keyBatch)
		return nil
	}); {
	case err != nil:
		return nil, err
	case len(keys) != expectedCount:
		panic("illegal state") // TODO return error
	}

	// makes all batches of equal size to support fast indexed access
	return p.equalizeBatches(expectedCount, keys)
}

func (p *bucketKeyLoader) _loadKeys(expectedCount int, addFn func(remainingCount int, dataChunk longbits.ByteString) error) error {
	bucketKeySize := bucketKeyType.Size()

	for remainingCount := expectedCount; remainingCount > 0; {
		if !p.nextChunk() {
			panic("insufficient length") // TODO error
		}
		dataChunk := p.lastChunk

		chunkKeyCount := len(dataChunk) / bucketKeySize
		if chunkKeyCount >= remainingCount {
			if len(dataChunk)%bucketKeySize != 0 {
				panic("unaligned") // TODO error
			}
			remainingCount -= remainingCount
			p.lastChunk = longbits.EmptyByteString
		} else {
			chunkKeyCount = remainingCount
			remainingCount = 0
			dataLen := chunkKeyCount * bucketKeySize
			dataChunk = dataChunk[:dataLen]
			p.lastChunk = p.lastChunk[dataLen:]
		}

		if err := addFn(remainingCount, dataChunk); err != nil {
			return err
		}
	}

	return nil
}

// memory-mapped objects shouldn't be copied, so the only way to make all batches equal is to split them
// so we will find the greatest common divisor (GCD) and use it to break batches apart.
//
// WARNING! Serializer MUST ensure that all batches have a GCD > MinKeyBucketBatchSize
//
func (p *bucketKeyLoader) equalizeBatches(totalCount int, keyBatches [][]bucketKey) ([][]bucketKey, error) {
	batchCount := len(keyBatches)
	if batchCount <= 1 {
		return keyBatches, nil
	}

	allSame := true
	gcd := len(keyBatches[0])

	for _, bk := range keyBatches[1 : batchCount-1] {
		if gcd < MinKeyBucketBatchSize {
			break
		}
		bkl := len(bk)
		gcd = args.GreatestCommonDivisor(gcd, bkl)
		if bkl != gcd {
			allSame = false
		}
	}

	switch {
	case gcd < MinKeyBucketBatchSize:
		panic("illegal value") // TODO error
	case !allSame:
		// check if it possible to use some powerOf2 nearby
		gcd2 := args.GreatestCommonDivisor(gcd, 1<<31)
		if gcd2 >= MinKeyBucketBatchSize && gcd2 > (gcd>>3) {
			gcd = gcd2
		}
	case len(keyBatches[batchCount-1]) <= gcd:
		// all are same, but the last one
		return keyBatches, nil
	}

	return p._equalizeBatches(totalCount, keyBatches, gcd)
}

func (p *bucketKeyLoader) _equalizeBatches(totalCount int, keyBatches [][]bucketKey, batchSize int) ([][]bucketKey, error) {
	batchCount := (totalCount + batchSize - 1) / batchSize
	result := make([][]bucketKey, 0, batchCount)

	for _, keyBatch := range keyBatches {
		kbs := len(keyBatch)
		switch {
		case kbs == batchSize:
			result = append(result, keyBatch)
			continue
		case kbs < batchSize:
			panic("illegal value") // TODO error
		}

		for base := 0; true; {
			nextBase := base + batchSize
			if nextBase >= kbs {
				result = append(result, keyBatch[base:])
				break
			}
			result = append(result, keyBatch[base:nextBase])
			base = nextBase
		}
	}

	return result, nil
}
