package longbits

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitBuilder_FirstLow_BitOrder(t *testing.T) {
	bb := BitBuilder{}
	bb.AppendBit(0)
	bb.AppendBit(1)
	bb.AppendBit(1)
	bb.AppendBit(0)
	bb.AppendBit(1)
	bb.AppendBit(1)
	bb.AppendBit(0)
	bb.AppendBit(1)

	bb.AppendByte(0xB6)
	bb.AppendSubByte(0x06, 4)
	bb.AppendSubByte(0x0B, 4)

	require.Equal(t, []byte{0xB6, 0xB6, 0xB6}, bb.dump())
	require.Equal(t, 24, bb.BitLen())
}

func TestBitBuilder_FirstHigh_BitOrder(t *testing.T) {
	bb := NewBitBuilder(MSB, 0)
	bb.AppendBit(0)
	bb.AppendBit(1)
	bb.AppendBit(1)
	bb.AppendBit(0)
	bb.AppendBit(1)
	bb.AppendBit(1)
	bb.AppendBit(0)
	bb.AppendBit(1)

	bb.AppendByte(0xB6)
	bb.AppendSubByte(0x06, 4)
	bb.AppendSubByte(0x0B, 4)

	require.Equal(t, []byte{0x6D, 0x6D, 0x6D}, bb.dump())
	require.Equal(t, 24, bb.BitLen())
}

func TestBitBuilder_FirstLow(t *testing.T) {
	bb := BitBuilder{}
	bb.AppendBit(0)
	bb.AppendBit(0)
	bb.AppendBit(1)
	bb.AppendBit(0)
	bb.AppendBit(1)
	bb.AppendBit(1)

	require.Equal(t, []byte{0x34}, bb.dump())
	require.Equal(t, 6, bb.BitLen())

	bb.AppendNBit(7, 1)
	require.Equal(t, []byte{0xF4, 0x1F}, bb.dump())
	require.Equal(t, 13, bb.BitLen())

	bb.AppendByte(0xAB)
	require.Equal(t, []byte{0xF4, 0x7F, 0x15}, bb.dump())
	require.Equal(t, 21, bb.BitLen())

	bb.PadWithBit(0)
	require.Equal(t, []byte{0xF4, 0x7F, 0x15}, bb.dump())
	require.Equal(t, 24, bb.BitLen())

	bb.PadWithBit(0)
	require.Equal(t, []byte{0xF4, 0x7F, 0x15}, bb.dump())
	require.Equal(t, 24, bb.BitLen())

	bb.AppendAlignedByte(0x01)
	bb.AppendByte(0x02)

	require.Equal(t, []byte{0xF4, 0x7F, 0x15, 0x01, 0x02}, bb.dump())
	require.Equal(t, 40, bb.BitLen())

	bb.AppendSubByte(0x5D, 4)
	require.Equal(t, []byte{0xF4, 0x7F, 0x15, 0x01, 0x02, 0x0D}, bb.dump())
	require.Equal(t, 44, bb.BitLen())

	bb.AppendNBit(15, 1)
	require.Equal(t, []byte{0xF4, 0x7F, 0x15, 0x01, 0x02, 0xFD, 0xFF, 0x07}, bb.dump())
	require.Equal(t, 59, bb.BitLen())

	bb.PadWithBit(1)
	require.Equal(t, []byte{0xF4, 0x7F, 0x15, 0x01, 0x02, 0xFD, 0xFF, 0xFF}, bb.dump())
	require.Equal(t, 64, bb.BitLen())

	bb.PadWithBit(1)
	require.Equal(t, []byte{0xF4, 0x7F, 0x15, 0x01, 0x02, 0xFD, 0xFF, 0xFF}, bb.dump())
	require.Equal(t, 64, bb.BitLen())
}

func TestBitBuilder_FirstHigh(t *testing.T) {
	bb := NewBitBuilder(MSB, 0)
	bb.AppendBit(0)
	bb.AppendBit(0)
	bb.AppendBit(1)
	bb.AppendBit(0)
	bb.AppendBit(1)
	bb.AppendBit(1)

	require.Equal(t, []byte{0x2C}, bb.dump())
	require.Equal(t, 6, bb.BitLen())

	bb.AppendNBit(7, 1)
	require.Equal(t, []byte{0x2F, 0xF8}, bb.dump())
	require.Equal(t, 13, bb.BitLen())

	bb.AppendByte(0xAB)
	require.Equal(t, []byte{0x2F, 0xFE, 0xA8}, bb.dump())
	require.Equal(t, 21, bb.BitLen())

	bb.PadWithBit(0)
	require.Equal(t, []byte{0x2F, 0xFE, 0xA8}, bb.dump())
	require.Equal(t, 24, bb.BitLen())

	bb.PadWithBit(0)
	require.Equal(t, []byte{0x2F, 0xFE, 0xA8}, bb.dump())
	require.Equal(t, 24, bb.BitLen())

	bb.AppendAlignedByte(0x01)
	bb.AppendByte(0x02)

	require.Equal(t, []byte{0x2F, 0xFE, 0xA8, 0x80, 0x40}, bb.dump())
	require.Equal(t, 40, bb.BitLen())

	bb.AppendSubByte(0x5D, 4)
	require.Equal(t, []byte{0x2F, 0xFE, 0xA8, 0x80, 0x40, 0xB0}, bb.dump())
	require.Equal(t, 44, bb.BitLen())

	bb.AppendNBit(15, 1)
	require.Equal(t, []byte{0x2F, 0xFE, 0xA8, 0x80, 0x40, 0xBF, 0xFF, 0xE0}, bb.dump())
	require.Equal(t, 59, bb.BitLen())

	bb.PadWithBit(1)
	require.Equal(t, []byte{0x2F, 0xFE, 0xA8, 0x80, 0x40, 0xBF, 0xFF, 0xFF}, bb.dump())
	require.Equal(t, 64, bb.BitLen())

	bb.PadWithBit(1)
	require.Equal(t, []byte{0x2F, 0xFE, 0xA8, 0x80, 0x40, 0xBF, 0xFF, 0xFF}, bb.dump())
	require.Equal(t, 64, bb.BitLen())
}

func TestBitBuilder_FirstLow_SubByte(t *testing.T) {
	bb := BitBuilder{}
	for i := byte(0); i < 8; i++ {
		bb.AppendSubByte(i, 3)
	}
	require.Equal(t, 24, bb.BitLen())
	require.Equal(t, []byte{0x88, 0xC6, 0xFA}, bb.dump())
}

func TestBitBuilder_FirstHigh_SubByte(t *testing.T) {
	bb := NewBitBuilder(MSB, 0)
	for i := byte(0); i < 8; i++ {
		bb.AppendSubByte(i, 3)
	}
	require.Equal(t, 24, bb.BitLen())
	require.Equal(t, []byte{0x11, 0x63, 0x5F}, bb.dump())
}

func TestBitBuilder_Done(t *testing.T) {
	t.Run("Aligned", func(t *testing.T) {
		bb := NewBitBuilder(MSB, 0)
		bb.AppendBit(0)
		bb.AppendBit(0)
		bb.AppendBit(1)
		bb.AppendBit(0)
		bb.AppendBit(1)
		bb.AppendBit(1)
		bb.AppendBit(1)
		bb.AppendBit(1)

		doneBytes, doneLen := bb.Done()
		require.Equal(t, []byte{0x2f}, doneBytes)
		require.Equal(t, 0, doneLen)
	})

	t.Run("NonAligned", func(t *testing.T) {
		bb := NewBitBuilder(MSB, 0)
		bb.AppendBit(0)
		bb.AppendBit(0)
		bb.AppendBit(1)
		bb.AppendBit(0)
		bb.AppendBit(1)
		bb.AppendBit(1)

		doneBytes, doneLen := bb.Done()
		require.Equal(t, []byte{0x2c}, doneBytes)
		require.Equal(t, -2, doneLen)
	})
}
