package longbits

import (
	"math/bits"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type IndexedBits interface {
	BitBool(index int) bool
	BitValue(index int) byte
	Byte(index int) byte
	BitLen() int
	SearchBit(startAt int, bit bool) int
}

func BitPos(index int) (bytePos int, bitPos uint8) {
	if index < 0 {
		panic(throw.IllegalValue())
	}
	return index >> 3, uint8(index & 0x07)
}

type BitSlice = BitSliceLSB
type BitSliceLSB []byte

func (v BitSliceLSB) BitMasked(index int) (value byte, mask uint8) {
	bytePos, bitPos := BitPos(index)
	mask = 1 << bitPos
	return v[bytePos] & mask, mask
}

func (v BitSliceLSB) BitBool(index int) bool {
	if b, _ := v.BitMasked(index); b != 0 {
		return true
	}
	return false
}

func (v BitSliceLSB) BitValue(index int) byte {
	if b, _ := v.BitMasked(index); b != 0 {
		return 1
	}
	return 0
}

func (v BitSliceLSB) Byte(index int) byte {
	return v[index]
}

func (v BitSliceLSB) BitLen() int {
	return len(v) << 3
}

func (v BitSliceLSB) SearchBit(startAt int, bit bool) int {
	byteIndex, bitIndex := BitPos(startAt)
	if byteIndex >= len(v) {
		return -1
	}

	if bit {
		return v.searchBit1(byteIndex, bitIndex)
	}
	return v.searchBit0(byteIndex, bitIndex)
}

func (v BitSliceLSB) searchBit1(bytePos int, bitPos byte) int {
	b := v[bytePos] >> bitPos
	if b != 0 {
		return bytePos<<3 + int(bitPos) + bits.TrailingZeros8(b)
	}

	for bytePos++; bytePos < len(v); bytePos++ {
		b := v[bytePos]
		if b != 0 {
			return bytePos<<3 + bits.TrailingZeros8(b)
		}
	}
	return -1
}

func (v BitSliceLSB) searchBit0(bytePos int, bitPos byte) int {
	b := (^v[bytePos]) >> bitPos
	if b != 0 {
		return bytePos<<3 + int(bitPos) + bits.TrailingZeros8(b)
	}

	for bytePos++; bytePos < len(v); bytePos++ {
		b := v[bytePos]
		if b != 0xFF {
			return bytePos<<3 + bits.TrailingZeros8(^b)
		}
	}
	return -1
}

/*********************************************************/

type BitSliceMSB []byte

func (v BitSliceMSB) BitMasked(index int) (value byte, mask uint8) {
	bytePos, bitPos := BitPos(index)
	mask = 0x80 >> bitPos
	return v[bytePos] & mask, mask
}

func (v BitSliceMSB) BitBool(index int) bool {
	if b, _ := v.BitMasked(index); b != 0 {
		return true
	}
	return false
}

func (v BitSliceMSB) BitValue(index int) byte {
	if b, _ := v.BitMasked(index); b != 0 {
		return 1
	}
	return 0
}

func (v BitSliceMSB) Byte(index int) byte {
	return v[index]
}

func (v BitSliceMSB) BitLen() int {
	return len(v) << 3
}

func (v BitSliceMSB) SearchBit(startAt int, bit bool) int {
	byteIndex, bitIndex := BitPos(startAt)
	if byteIndex >= len(v) {
		return -1
	}

	if bit {
		return v.searchBit1(byteIndex, bitIndex)
	}
	return v.searchBit0(byteIndex, bitIndex)
}

func (v BitSliceMSB) searchBit1(bytePos int, bitPos byte) int {
	b := v[bytePos] << bitPos
	if b != 0 {
		return bytePos<<3 + int(bitPos) + bits.LeadingZeros8(b)
	}

	for bytePos++; bytePos < len(v); bytePos++ {
		b := v[bytePos]
		if b != 0 {
			return bytePos<<3 + bits.LeadingZeros8(b)
		}
	}
	return -1
}

func (v BitSliceMSB) searchBit0(bytePos int, bitPos byte) int {
	b := (^v[bytePos]) << bitPos
	if b != 0 {
		return bytePos<<3 + int(bitPos) + bits.LeadingZeros8(b)
	}

	for bytePos++; bytePos < len(v); bytePos++ {
		b := v[bytePos]
		if b != 0xFF {
			return bytePos<<3 + bits.LeadingZeros8(^b)
		}
	}
	return -1
}
