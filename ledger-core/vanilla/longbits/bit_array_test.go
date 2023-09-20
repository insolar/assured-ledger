package longbits

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitPos(t *testing.T) {
	require.Panics(t, func() { BitPos(-1) })

	for i, tc := range []struct {
		input   int
		bytePos int
		bitPos  uint8
	}{
		{ 0, 0, 0},
		{ 1, 0, 1},
		{ 2, 0, 2},
		{ 10, 1, 2},
	} {
		bytePos, bitPos := BitPos(tc.input)
		require.Equal(t, tc.bytePos, bytePos, "%d: bytePos", i)
		require.Equal(t, tc.bitPos, bitPos, "%d: bitPos", i)
	}
}

func newString(s string) *string {
	return &s
}

func TestBitSlice_BitBool(t *testing.T) {
	for _, tc := range []struct {
		slice []byte
		index int
		res   bool
		panic *string
	}{
		{
			slice: []byte{0x2},
			index: 1,
			res:   true,
		},
		{
			slice: []byte{0x0, 0x2},
			index: 9,
			res:   true,
		},
		{
			slice: []byte{0x0, 0x2},
			index: 8,
			res:   false,
		},
		{
			slice: []byte{0x1},
			index: 8,
			panic: newString("out of bounds"),
		},
	} {
		if tc.panic != nil {
			require.Panics(t, func() { BitSliceLSB(tc.slice).BitBool(tc.index) })
			continue
		}
		require.Equal(t, tc.res, BitSliceLSB(tc.slice).BitBool(tc.index))
	}
}

func TestBitSlice_BitValue(t *testing.T) {
	for _, tc := range []struct {
		slice []byte
		index int
		res   byte
		panic *string
	}{
		{
			slice: []byte{0x2},
			index: 1,
			res:   1,
		},
		{
			slice: []byte{0x0, 0x2},
			index: 9,
			res:   1,
		},
		{
			slice: []byte{0x0, 0x2},
			index: 8,
			res:   0,
		},
		{
			slice: []byte{0x1},
			index: 8,
			panic: newString("out of bounds"),
		},
	} {
		if tc.panic != nil {
			require.Panics(t, func() { BitSliceLSB(tc.slice).BitValue(tc.index) })
			continue
		}
		require.Equal(t, tc.res, BitSliceLSB(tc.slice).BitValue(tc.index))
	}
}

func TestBitSlice_BitLen(t *testing.T) {
	require.Equal(t, 0, BitSliceLSB(nil).BitLen())
	require.Equal(t, 0, BitSliceLSB([]byte{}).BitLen())
	require.Equal(t, 8, BitSliceLSB([]byte{0x1}).BitLen())
	require.Equal(t, 16, BitSliceLSB([]byte{0x1, 0x10}).BitLen())
}

func TestBitSlice_SearchBit(t *testing.T) {
	for _, tc := range []struct {
		slice   []byte
		startAt int
		bit     bool
		res     int
		panic   *string
	}{
		{
			slice:   []byte{0x2},
			startAt: 1,
			res:     2,
		},
		{
			slice:   []byte{0x2},
			startAt: 8,
			res:     -1,
		},
		{
			slice:   []byte{0x0, 0x2},
			startAt: 9,
			res:     10,
		},
		{
			slice:   []byte{0xFF, 0x2},
			startAt: 0,
			res:     8,
		},
		{
			slice:   []byte{0x0, 0x2},
			startAt: 9,
			bit:     true,
			res:     9,
		},
		{
			slice:   []byte{0x0, 0x2},
			startAt: 0,
			bit:     true,
			res:     9,
		},
		{
			slice:   []byte{0x0, 0x0},
			startAt: 8,
			bit:     true,
			res:     -1,
		},
		{
			slice:   []byte{0xFF, 0xFF},
			startAt: 0,
			res:     -1,
		},
		{
			slice:   []byte{0x1},
			startAt: -1,
			panic:   newString("illegal value"),
		},
	} {
		if tc.panic != nil {
			require.Panics(t, func() { BitSliceLSB(tc.slice).SearchBit(tc.startAt, tc.bit) })
			continue
		}
		require.Equal(t, tc.res, BitSliceLSB(tc.slice).SearchBit(tc.startAt, tc.bit))
	}
}

func TestBitSlice_Byte(t *testing.T) {
	require.Equal(t, byte(0x1), BitSliceLSB([]byte{0x1}).Byte(0))
	require.Equal(t, byte(0x10), BitSliceLSB([]byte{0x1, 0x10}).Byte(1))
	require.Panics(t, func() { BitSliceLSB([]byte{0x1}).Byte(1) })
}
