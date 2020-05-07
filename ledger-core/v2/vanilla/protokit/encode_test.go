// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package protokit

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeVarint(t *testing.T) {
	lastSize := 1
	require.Equal(t, lastSize, testEncodeDecode(t, 0))
	require.Equal(t, lastSize, testEncodeDecode(t, 1))

	v := uint64(0x80)
	for v != 0 {
		n := testEncodeDecode(t, v-1)
		require.Equal(t, lastSize, n)

		lastSize++

		n = testEncodeDecode(t, v)
		require.Equal(t, lastSize, n)

		n = testEncodeDecode(t, v+1)
		require.Equal(t, lastSize, n)
		v <<= 7
	}
}

func testEncodeDecode(t *testing.T, v uint64) int {
	var b [MaxVarintSize]byte

	n := EncodeVarintToBytes(b[:], v)
	require.Equal(t, n, SizeVarint64(v))
	if v <= math.MaxUint32 {
		require.Equal(t, n, SizeVarint32(uint32(v)))
	}
	u, n2 := DecodeVarintFromBytes(b[:])
	require.Equal(t, n, n2)
	require.Equal(t, v, u)

	buf := bytes.Buffer{}

	err := EncodeVarint(&buf, v)
	require.NoError(t, err)
	u, err = DecodeVarint(&buf)
	require.NoError(t, err)
	require.Equal(t, v, u)

	return n
}

func TestEncodeDecodeFixed32(t *testing.T) {
	v := uint32(1)
	for {
		buf := bytes.Buffer{}
		err := EncodeFixed32(&buf, v)
		require.NoError(t, err)
		u, err := DecodeFixed32(&buf)
		require.NoError(t, err)
		require.Equal(t, uint64(v), u)
		if v != 0 {
			break
		}
		v <<= 1
	}
}

func TestEncodeDecodeFixed64(t *testing.T) {
	v := uint64(1)
	for {
		buf := bytes.Buffer{}
		err := EncodeFixed64(&buf, v)
		require.NoError(t, err)
		u, err := DecodeFixed64(&buf)
		require.NoError(t, err)
		require.Equal(t, v, u)
		if v != 0 {
			break
		}
		v <<= 1
	}
}
