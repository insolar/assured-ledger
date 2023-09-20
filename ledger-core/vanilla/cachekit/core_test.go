package cachekit

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicLimits(t *testing.T) {
	c := NewUintCache(newStrategy(5, 15, false, true))

	for i := 1; i <= 20; i++ {
		s := strconv.Itoa(i)
		c.Put(uint64(i), "V" + s)
	}

	require.Equal(t, 15, c.Occupied())
	require.Equal(t, 15, c.core.Occupied())
	require.Equal(t, 20, c.Allocated())

	for i := 1; i <= 5; i++ {
		require.False(t, c.Contains(uint64(i)), i)
	}

	for i := 6; i <= 20; i++ {
		s := strconv.Itoa(i)
		require.True(t, c.Contains(uint64(i)), i)
		v, ok := c.Peek(uint64(i))
		require.True(t, ok, i)
		require.Equal(t, "V" + s, v)
	}
}

func TestDelete(t *testing.T) {
	c := NewUintCache(newStrategy(5, 15, false, true))

	for i := 1; i <= 15; i++ {
		s := strconv.Itoa(i)
		c.Put(uint64(i), "V" + s)
	}

	c.Delete(8)
	require.False(t, c.Contains(8))

	for i := 16; i <= 20; i++ {
		s := strconv.Itoa(i)
		c.Put(uint64(i), "V" + s)
	}

	require.Equal(t, 14, c.Occupied())
	require.Equal(t, 15, c.core.Occupied())
	require.Equal(t, 20, c.Allocated())

	// this loop forces creation of generations and cleanup of them
	// so the deleted item can be collected
	for i := 100; i > 0; i-- {
		c.Get(19)
		c.Get(20)
	}

	require.Equal(t, 14, c.core.Occupied())

	for i := 1; i <= 5; i++ {
		require.False(t, c.Contains(uint64(i)), i)
	}

	for i := 6; i <= 7; i++ {
		s := strconv.Itoa(i)
		require.True(t, c.Contains(uint64(i)), i)
		v, ok := c.Peek(uint64(i))
		require.True(t, ok, i)
		require.Equal(t, "V" + s, v)
	}

	require.False(t, c.Contains(8))

	for i := 9; i <= 20; i++ {
		s := strconv.Itoa(i)
		require.True(t, c.Contains(uint64(i)), i)
		v, ok := c.Peek(uint64(i))
		require.True(t, ok, i)
		require.Equal(t, "V" + s, v)
	}

	// it uses the empty slot released by deletion
	c.Put(21, "V21")

	require.Equal(t, 15, c.Occupied())
	require.Equal(t, 15, c.core.Occupied())
	require.Equal(t, 20, c.Allocated())
}

func TestCounter(t *testing.T) {
	c := NewUintCache(newStrategy(2, 2, false, true))
	c.Put(1, "A")
	c.Put(2, "B")
	for i := 100; i > 0; i-- {
		c.Get(1)
		c.Get(2)
	}

	require.Less(t, int32(2), *c.core.alloc.get(1))

	c.core.trimGenerations(math.MaxUint32)

	require.EqualValues(t, 2, *c.core.alloc.get(1))
}

func TestFencedCounter(t *testing.T) {
	c := NewUintCache(newStrategy(4, 2, true, true))
	c.Put(1, "A")
	c.Put(2, "B")
	for i := 100; i > 0; i-- {
		c.Get(1)
		c.Get(2)
	}

	require.EqualValues(t, 2, *c.core.alloc.get(1))

	c.core.trimGenerations(math.MaxUint32)

	require.EqualValues(t, 2, *c.core.alloc.get(1))
}

func TestTouch(t *testing.T) {
	c := NewUintCache(newStrategy(5, 15, true, true))

	for i := 1; i <= 20; i++ {
		s := strconv.Itoa(i)
		c.Put(uint64(i), "V" + s)

		_, _ = c.Peek(4)
		v, ok := c.Get(5)

		if i < 5 {
			require.False(t, ok, i)
		} else {
			require.True(t, ok, i)
			require.Equal(t, "V5", v)
		}
	}

	require.Equal(t, 15, c.Occupied())
	require.Equal(t, 15, c.core.Occupied())
	require.Equal(t, 20, c.Allocated())

	for i := 1; i <= 4; i++ {
		require.False(t, c.Contains(uint64(i)), i)
	}

	require.True(t, c.Contains(5))

	for i := 7; i <= 20; i++ {
		s := strconv.Itoa(i)
		require.True(t, c.Contains(uint64(i)), i)
		v, ok := c.Peek(uint64(i))
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
