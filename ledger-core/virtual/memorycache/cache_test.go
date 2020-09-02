// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memorycache

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

func TestLRUMemoryCache(t *testing.T) {
	defer commontestutils.LeakTester(t)

	table := []struct {
		name     string
		strategy cacheStrategy
	}{
		{name: "trimEach=false", strategy: cacheStrategy{pgSize: 2, maxTotal: 3}},
		{name: "trimEach=true", strategy: cacheStrategy{pgSize: 2, maxTotal: 3, trimEach: true}},
	}

	createValue := func(memory []byte) descriptor.Object {
		return descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, memory, false)
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				key1, key2, key3, key4 = gen.UniqueGlobalRef(), gen.UniqueGlobalRef(), gen.UniqueGlobalRef(), gen.UniqueGlobalRef()

				value1   = createValue([]byte("value 1"))
				value1v2 = createValue([]byte("value 1 version 2"))
				value2   = createValue([]byte("value 2"))
				value3   = createValue([]byte("value 3"))
				value4   = createValue([]byte("value 4"))
			)
			mCache := NewMemoryCache(testCase.strategy)

			// add first pair, check len and cap
			{
				assert.True(t, mCache.Put(key1, value1))
				assert.Equal(t, 1, mCache.Occupied())
				assert.Equal(t, 2, mCache.Allocated())
			}
			// add second pair, check len and cap
			{
				assert.True(t, mCache.Put(key2, value2))
				assert.Equal(t, 2, mCache.Occupied())
				assert.Equal(t, 2, mCache.Allocated())
			}
			// get first key
			{
				val, ok := mCache.Get(key1)
				assert.True(t, ok)
				assert.Equal(t, []byte("value 1"), val.Memory())
			}
			// replace first key and peek value
			{
				assert.False(t, mCache.Replace(key1, value1v2))
				val, ok := mCache.Peek(key1)
				assert.True(t, ok)
				assert.Equal(t, []byte("value 1 version 2"), val.Memory())
			}
			// try to add existent key
			assert.False(t, mCache.Put(key2, value2))
			// add third pair, using replace
			{
				assert.True(t, mCache.Replace(key3, value3))
				assert.Equal(t, 3, mCache.Occupied())
				assert.Equal(t, 4, mCache.Allocated())
			}
			// add fourth pair
			{
				expectedOccupied := 4
				if testCase.strategy.trimEach {
					expectedOccupied = 3
				}

				assert.True(t, mCache.Put(key4, value4))
				assert.Equal(t, expectedOccupied, mCache.Occupied())
				assert.Equal(t, 4, mCache.Allocated())
			}
			// delete key2 and check len/cap
			{
				expectedOccupied := 3
				if testCase.strategy.trimEach {
					expectedOccupied = 2
				}

				assert.True(t, mCache.Delete(key2))
				assert.Equal(t, expectedOccupied, mCache.Occupied())
				assert.Equal(t, 4, mCache.Allocated())
			}
			// check key2 after removal
			assert.False(t, mCache.Contains(key2))
		})
	}
}

func TestLRUMemoryCache_Concurrent(t *testing.T) {
	defer commontestutils.LeakTester(t)

	mCache := NewMemoryCache(cacheStrategy{pgSize: 2, maxTotal: 5, trimEach: true})
	wg := sync.WaitGroup{}

	action := func(key reference.Global) {
		defer wg.Done()
		var (
			value1 = descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, []byte("value 1"), false)
			value2 = descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, []byte("value 2"), false)
		)

		assert.True(t, mCache.Put(key, value1))
		assert.False(t, mCache.Replace(key, value2))
		_, ok := mCache.Get(key)
		assert.True(t, ok)
		assert.Greater(t, mCache.Occupied(), 0)
		assert.Greater(t, mCache.Allocated(), 0)
		assert.True(t, mCache.Delete(key))
	}

	keys := gen.UniqueGlobalRefs(5)
	for _, key := range keys {
		wg.Add(1)
		go action(key)
	}

	wg.Wait()
}
