package args

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGCDListInt(t *testing.T) {
	require.Equal(t, 20, GCDListInt(40, 100, 60))
	require.Equal(t, 20, GCDListInt(20, 100, 40, 60))
	require.Equal(t, 1, GCDListInt(5, 100, 40, 60, 3))
}

func TestGCDListInt64(t *testing.T) {
	require.Equal(t, 20, int(GCDListInt64(40, 100, 60)))
	require.Equal(t, 20, int(GCDListInt64(20, 100, 40, 60)))
	require.Equal(t, 1, int(GCDListInt64(5, 100, 40, 60, 3)))
}

func TestGCDListUint64(t *testing.T) {
	require.Equal(t, 20, int(GCDListUint64(40, 100, 60)))
	require.Equal(t, 20, int(GCDListUint64(20, 100, 40, 60)))
	require.Equal(t, 1, int(GCDListUint64(5, 100, 40, 60, 3)))
}

func TestGreatestCommonDivisor(t *testing.T) {
	require.Equal(t, 0, GreatestCommonDivisor(0, 0))
	require.Equal(t, 1, GreatestCommonDivisor(1, 0))
	require.Equal(t, 1, GreatestCommonDivisor(1, 1))
	require.Equal(t, 100, GreatestCommonDivisor(100, 100))
	require.Equal(t, 2, GreatestCommonDivisor(2, 100))
	require.Equal(t, 1, GreatestCommonDivisor(3, 100))
	require.Equal(t, 4, GreatestCommonDivisor(4, 100))
	require.Equal(t, 5, GreatestCommonDivisor(5, 100))
	require.Equal(t, 20, GreatestCommonDivisor(20, 100))
	require.Equal(t, 20, GreatestCommonDivisor(60, 100))
	require.Equal(t, 10, GreatestCommonDivisor(90, 100))
}

func TestGreatestCommonDivisorInt64(t *testing.T) {
	require.Equal(t, 0, int(GreatestCommonDivisorInt64(0, 0)))
	require.Equal(t, 1, int(GreatestCommonDivisorInt64(1, 0)))
	require.Equal(t, 1, int(GreatestCommonDivisorInt64(1, 1)))
	require.Equal(t, 100, int(GreatestCommonDivisorInt64(100, 100)))
	require.Equal(t, 2, int(GreatestCommonDivisorInt64(2, 100)))
	require.Equal(t, 1, int(GreatestCommonDivisorInt64(3, 100)))
	require.Equal(t, 4, int(GreatestCommonDivisorInt64(4, 100)))
	require.Equal(t, 5, int(GreatestCommonDivisorInt64(5, 100)))
	require.Equal(t, 20, int(GreatestCommonDivisorInt64(20, 100)))
	require.Equal(t, 20, int(GreatestCommonDivisorInt64(60, 100)))
	require.Equal(t, 10, int(GreatestCommonDivisorInt64(90, 100)))
}

func TestGreatestCommonDivisorUint64(t *testing.T) {
	require.Equal(t, 0, int(GreatestCommonDivisorUint64(0, 0)))
	require.Equal(t, 1, int(GreatestCommonDivisorUint64(1, 0)))
	require.Equal(t, 1, int(GreatestCommonDivisorUint64(1, 1)))
	require.Equal(t, 100, int(GreatestCommonDivisorUint64(100, 100)))
	require.Equal(t, 2, int(GreatestCommonDivisorUint64(2, 100)))
	require.Equal(t, 1, int(GreatestCommonDivisorUint64(3, 100)))
	require.Equal(t, 4, int(GreatestCommonDivisorUint64(4, 100)))
	require.Equal(t, 5, int(GreatestCommonDivisorUint64(5, 100)))
	require.Equal(t, 20, int(GreatestCommonDivisorUint64(20, 100)))
	require.Equal(t, 20, int(GreatestCommonDivisorUint64(60, 100)))
	require.Equal(t, 10, int(GreatestCommonDivisorUint64(90, 100)))
}

func TestIsPowerOfTwo(t *testing.T) {
	require.True(t, IsPowerOfTwo(0))
	require.True(t, IsPowerOfTwo(1))
	require.True(t, IsPowerOfTwo(2))
	require.True(t, IsPowerOfTwo(4))
	require.True(t, IsPowerOfTwo(1024))
	require.False(t, IsPowerOfTwo(3))
	require.False(t, IsPowerOfTwo(5))
	require.False(t, IsPowerOfTwo(1023))
	require.False(t, IsPowerOfTwo(1025))
}

func TestCeilingPowerOfTwo(t *testing.T) {
	require.Equal(t, uint(0), CeilingPowerOfTwo(0))
	require.Equal(t, uint(1), CeilingPowerOfTwo(1))
	require.Equal(t, uint(2), CeilingPowerOfTwo(2))
	require.Equal(t, uint(4), CeilingPowerOfTwo(4))
	require.Equal(t, uint(8), CeilingPowerOfTwo(8))
	require.Equal(t, uint(8), CeilingPowerOfTwo(5))
	require.Equal(t, uint(8), CeilingPowerOfTwo(6))
	require.Equal(t, uint(8), CeilingPowerOfTwo(7))
}
