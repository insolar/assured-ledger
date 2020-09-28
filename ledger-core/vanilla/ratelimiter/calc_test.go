// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ratelimiter

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMinLimit(t *testing.T) {
	require.Equal(t, int64(20), minLimit(1000, 10, 20, 100, 1000, 10000))
	require.Equal(t, int64(50), minLimit(1000, 50, 20, 100, 1000, 10000))
	require.Equal(t, int64(10), minLimit(10, 50, 20, 100, 1000, 10000))
	require.Equal(t, int64(-10), minLimit(-10, -50, -20, -100, 1000, -10000))
}

func TestMaxLimit(t *testing.T) {
	require.Equal(t, int64(10000), maxLimit(1000, 10, 20, 100, 1000, 10000))
	require.Equal(t, int64(1000), maxLimit(1000, 10, 20, 100))
}

func u(a, b, c int) [3]int {
	return [3]int{a, b, c}
}

func uR(a uint32, b uint64, c uint32) [3]int {
	return [3]int{int(a), int(b), int(c)}
}

func TestThroughputQuantizer(t *testing.T) {
	require.Equal(t, u(0, 1, 1), uR(ThroughputQuantizer(1, 1000)))
	require.Equal(t, u(0, 1, 1), uR(ThroughputQuantizer(16, 1000)))
	require.Equal(t, u(0, 1, 1), uR(ThroughputQuantizer(16, 1000, 0)))
	require.Equal(t, u(0, 1, 1), uR(ThroughputQuantizer(16, 1000, math.MaxInt64)))

	// quantizer is not accurate when traffic limits are lower than samples/sec
	require.Equal(t, u(1, 1000, 1), uR(ThroughputQuantizer(1, 1000, 256)))
	require.Equal(t, u(1, 1000, 1), uR(ThroughputQuantizer(16, 1000, 256)))

	require.Equal(t, u(1, 1000, 1), uR(ThroughputQuantizer(1, 1000, 1000)))
	require.Equal(t, u(1, 1000, 1), uR(ThroughputQuantizer(1, 1000, 2560)))
	require.Equal(t, u(1, 1000, 1), uR(ThroughputQuantizer(1, 1000, 4000)))
	require.Equal(t, u(2, 2000, 1), uR(ThroughputQuantizer(1, 1000, 8000)))
	require.Equal(t, u(8, 8000, 1), uR(ThroughputQuantizer(1, 1000, 25600)))
	require.Equal(t, u(64, 64000, 1), uR(ThroughputQuantizer(1, 1000, 256000)))
	require.Equal(t, u(256, 256000, 1), uR(ThroughputQuantizer(1, 1000, 2_560_000)))
	require.Equal(t, u(256, 256000, 1), uR(ThroughputQuantizer(1, 1000, 25600000)))
	require.Equal(t, u(256, 256000, 1), uR(ThroughputQuantizer(1, 1000, 256000000)))
	require.Equal(t, u(256, 256000, 1), uR(ThroughputQuantizer(1, 1000, 2_560_000_000)))
	require.Equal(t, u(256, 2048000, 8), uR(ThroughputQuantizer(1, 1000, 25_600_000_000)))

	require.Equal(t, u(8, 2048, 1), uR(ThroughputQuantizer(1, 256, 8192, 16384, 65536)))
	require.Equal(t, u(4, 1024, 1), uR(ThroughputQuantizer(1, 256, 8192, 16384+4096, 65536)))
}
