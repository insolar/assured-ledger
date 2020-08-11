// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package longbits

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestByteString_ReadAt(t *testing.T) {
	for _, tc := range []struct {
		str    ByteString
		input  []byte
		off    int64
		res    int
		output []byte
		err    error
	}{
		{
			str:    ByteString("123"),
			input:  []byte{0x0, 0x0},
			output: []byte("23"),
			off:    1,
			res:    2,
		},
		{
			str:    ByteString("123"),
			input:  []byte{0x0, 0x0},
			output: []byte{0x0, 0x0},
			off:    4,
			err:    io.ErrUnexpectedEOF,
		},
		{
			str:    ByteString("123"),
			input:  []byte{},
			output: []byte{},
			off:    3,
		},
		{
			str: ByteString("123"),
			off: 3,
		},
	} {
		res, err := tc.str.ReadAt(tc.input, tc.off)
		require.Equal(t, tc.res, res)
		require.Equal(t, tc.err, err)
		require.Equal(t, tc.output, tc.input)
	}
}

func TestByteString_IsEmpty(t *testing.T) {
	require.Equal(t, true, ByteString("").IsEmpty())
	require.Equal(t, false, ByteString("1").IsEmpty())
}

func TestFill(t *testing.T) {
	require.Equal(t, EmptyByteString, Fill(0, 0x1))
	require.Equal(t, ByteString("\x01"), Fill(1, 0x1))
	require.Equal(t, ByteString("\x00"), Fill(1, 0))
}

func TestByteString_SearchBit(t *testing.T) {
	for _, tc := range []struct {
		ByteString ByteString
		startAt    int
		bit        bool
		res        int
		panic      *string
	}{
		{
			ByteString: ByteString("\x02"),
			startAt:    1,
			res:        2,
		},
		{
			ByteString: ByteString("\x02"),
			startAt:    8,
			res:        -1,
		},
		{
			ByteString: ByteString("\x02\x00"),
			startAt:    9,
			res:        9,
		},
		{
			ByteString: ByteString("\xff\x02"),
			startAt:    0,
			res:        8,
		},
		{
			ByteString: ByteString("\x00\x02"),
			startAt:    9,
			bit:        true,
			res:        9,
		},
		{
			ByteString: ByteString("\x00\x02"),
			startAt:    0,
			bit:        true,
			res:        9,
		},
		{
			ByteString: ByteString("\x00\x00"),
			startAt:    8,
			bit:        true,
			res:        -1,
		},
		{
			ByteString: ByteString("\xFF\xFF"),
			startAt:    0,
			res:        -1,
		},
		{
			ByteString: ByteString("\x01"),
			startAt:    -1,
			panic:      newString("illegal value"),
		},
	} {
		if tc.panic != nil {
			require.PanicsWithValue(t, *tc.panic, func() { tc.ByteString.SearchBit(tc.startAt, tc.bit) }, tc.ByteString)
			continue
		}
		require.Equal(t, tc.res, tc.ByteString.SearchBit(tc.startAt, tc.bit), tc.ByteString)
	}
}

func TestByteString_BitPos(t *testing.T) {
	for _, tc := range []struct {
		ByteString ByteString
		index      int
		bytePos    int
		bitPos     uint8
		panic      *string
	}{
		{
			ByteString: ByteString("\x02"),
			index:      1,
			bytePos:    0,
			bitPos:     0x1,
		},
		{
			ByteString: ByteString("\x02"),
			index:      8,
			panic:      newString("out of bounds"),
		},
		{
			ByteString: ByteString("\x02\x00"),
			index:      9,
			bytePos:    1,
			bitPos:     0x1,
		},
		{
			ByteString: ByteString("\xff\x02"),
			index:      0,
			bytePos:    0,
			bitPos:     0,
		},
		{
			ByteString: ByteString("\x01"),
			index:      -1,
			panic:      newString("illegal value"),
		},
	} {
		if tc.panic != nil {
			require.Panics(t, func() { tc.ByteString.BitPos(tc.index) }, tc.ByteString)
			continue
		}
		bytePos, bitPos := tc.ByteString.BitPos(tc.index)
		require.Equal(t, tc.bytePos, bytePos, tc.ByteString)
		require.Equal(t, tc.bitPos, bitPos, tc.ByteString)
	}
}

func TestByteString_CutOutBits64(t *testing.T) {
	for _, tc := range []struct {
		ByteString ByteString
		res        Bits64
	}{
		{
			ByteString: ByteString(""),
			res:        Bits64{},
		},
		{
			ByteString: ByteString("\x02"),
			res:        Bits64{0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		{
			ByteString: ByteString("\x02\x01\x02\x01\x02\x01\x02\x01\x02\x01"),
			res:        Bits64{0x2, 0x1, 0x2, 0x1, 0x1, 0x2, 0x1, 0x1},
		},
	} {
		require.Equal(t, tc.res, tc.ByteString.CutOutBits64())
	}
}

func TestByteString_FoldToBits64(t *testing.T) {
	for _, tc := range []struct {
		ByteString ByteString
		res        Bits64
	}{
		{
			ByteString: ByteString(""),
			res:        Bits64{},
		},
		{
			ByteString: ByteString("\x02"),
			res:        Bits64{0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		{
			ByteString: ByteString("\x03\x03\x02\x01\x02\x01\x02\x01\x02\x01"),
			res:        Bits64{0x1, 0x2, 0x2, 0x1, 0x2, 0x1, 0x2, 0x1},
		},
	} {
		require.Equal(t, tc.res, tc.ByteString.FoldToBits64())
	}
}

func TestByteString_String(t *testing.T) {
	for _, tc := range []struct {
		ByteString ByteString
		res        string
	}{
		{
			ByteString: ByteString(""),
			res:        "bits[0]0x00000000",
		},
		{
			ByteString: ByteString("\x02"),
			res:        "bits[8]0x00000002",
		},
		{
			ByteString: ByteString("\x03\x03\x02\x01\x02\x01\x02\x01\x02\x01"),
			res:        "bits[80]0x102010201020201",
		},
	} {
		require.Equal(t, tc.res, tc.ByteString.String())
	}
}

func TestByteStringEqual(t *testing.T) {
	require.False(t, EmptyByteString.Equal(nil))
	require.True(t, EmptyByteString.Equal(EmptyByteString))
	require.True(t, EmptyByteString.Equal(WrapBytes([]byte{})))
	require.True(t, WrapStr("abc").Equal(WrapBytes([]byte("abc"))))
}

func TestByteStringFold64Unaligned(t *testing.T) {
	var b [64 * 3 / 8]byte
	for i := range b {
		require.Zero(t, CopyBytes(b[:i+1]).FoldToUint64())
		require.Zero(t, CopyBytes(b[:i+1]).FoldToBits64())
	}
}

func TestByteStringCutOut64Unaligned(t *testing.T) {
	var b [64 * 3 / 8]byte
	for i := range b {
		require.Zero(t, CopyBytes(b[:i+1]).CutOutUint64())
		require.Zero(t, CopyBytes(b[:i+1]).CutOutBits64())
	}
}
