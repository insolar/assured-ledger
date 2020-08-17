// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cachekit

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicLimits(t *testing.T) {
	sc := NewUintCache(newStrategy(5, 15, false, true))

	for i := 1; i <= 20; i++ {
		s := strconv.Itoa(i)
		sc.Put(uint64(i), "V" + s)
	}

	require.Equal(t, 15, sc.Occupied())
	require.Equal(t, 15, sc.core.Occupied())
	require.Equal(t, 20, sc.Allocated())

	for i := 1; i <= 5; i++ {
		require.False(t, sc.Contains(uint64(i)), i)
	}

	for i := 6; i <= 20; i++ {
		s := strconv.Itoa(i)
		require.True(t, sc.Contains(uint64(i)), i)
		v, ok := sc.Get(uint64(i))
		require.True(t, ok, i)
		require.Equal(t, "V" + s, v)
	}
}

func TestTouch(t *testing.T) {
	sc := NewUintCache(newStrategy(5, 15, true, true))

	for i := 1; i <= 20; i++ {
		s := strconv.Itoa(i)
		sc.Put(uint64(i), "V" + s)

		_, _ = sc.Peek(4)
		v, ok := sc.Get(5)

		if i < 5 {
			require.False(t, ok, i)
		} else {
			require.True(t, ok, i)
			require.Equal(t, "V5", v)
		}
	}

	require.Equal(t, 15, sc.Occupied())
	require.Equal(t, 15, sc.core.Occupied())
	require.Equal(t, 20, sc.Allocated())

	for i := 1; i <= 4; i++ {
		require.False(t, sc.Contains(uint64(i)), i)
	}

	require.True(t, sc.Contains(5))

	for i := 7; i <= 20; i++ {
		s := strconv.Itoa(i)
		require.True(t, sc.Contains(uint64(i)), i)
		v, ok := sc.Get(uint64(i))
		require.True(t, ok, i)
		require.Equal(t, "V" + s, v)
	}
}

func BenchmarkCore(b *testing.B) {
	b.Run("fenced-16K", func(b *testing.B) {
		benchmarkCore(b, newStrategy(1024, 1<<14, true, false))
	})

	b.Run("bare-16K", func(b *testing.B) {
		benchmarkCore(b, newStrategy(1024, 1<<14, false, false))
	})

	b.Run("fenced-16M", func(b *testing.B) {
		benchmarkCore(b, newStrategy(1024, 1<<24, true, false))
	})

	b.Run("bare-16M", func(b *testing.B) {
		benchmarkCore(b, newStrategy(1024, 1<<24, false, false))
	})
}

func benchmarkCore(b *testing.B, s Strategy) {
	c := NewUintCache(s)
	b.Run("put", func(b *testing.B) {
		b.ReportAllocs()
		for i := b.N; i > 0; i-- {
			c.Put(uint64(i), "")
		}
	})

	max := c.Occupied()
	b.Run("put-dup", func(b *testing.B) {
		b.ReportAllocs()
		for i := b.N; i > 0; i-- {
			c.Put(uint64(i%max), "")
		}
	})

	b.Run("peek", func(b *testing.B) {
		b.ReportAllocs()
		for i := b.N; i > 0; i-- {
			c.Peek(uint64(i%max))
		}
	})

	b.Run("get", func(b *testing.B) {
		b.ReportAllocs()
		for i := b.N; i > 0; i-- {
			c.Get(uint64(i%max))
		}
	})

	b.Run("get-subset", func(b *testing.B) {
		b.ReportAllocs()
		for i := b.N; i > 0; i-- {
			c.Get(uint64((i&0xFF)<<10)) // access a small (256) set of records across all
		}
	})
}

func BenchmarkFence(b *testing.B) {
	b.Run("fenced", func(b *testing.B) {
		benchmarkFence(b, newStrategy(32, 4, true, false))
	})

	b.Run("bare", func(b *testing.B) {
		benchmarkFence(b, newStrategy(32, 4, false, false))
	})
}

func benchmarkFence(b *testing.B, s Strategy) {
	c := NewUintCache(s)
	c.Put(0, "")
	c.Put(1, "")

	b.Run("get", func(b *testing.B) {
		b.ReportAllocs()
		for i := b.N; i > 0; i-- {
			c.Get(uint64(i)&1)
		}
	})

}
