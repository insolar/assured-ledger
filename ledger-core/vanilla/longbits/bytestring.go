// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package longbits

import (
	"encoding/hex"
	"io"
	"math/bits"
	"strings"
)

const EmptyByteString = ByteString("")

func WrapStr(s string) ByteString {
	return ByteString(s)
}

func CopyBytes(v []byte) ByteString {
	return ByteString(v)
}

func Zero(len int) ByteString {
	return Fill(len, 0)
}

func Fill(len int, fill byte) ByteString {
	if len == 0 {
		return EmptyByteString
	}
	b := make([]byte, len)
	if fill != 0 {
		for i := len - 1; i >= 0; i-- {
			b[i] = fill
		}
	}
	// lest hope for the compiler to optimize it
	return ByteString(b)
}

var _ FoldableReader = EmptyByteString

type ByteString string

func (v ByteString) IsEmpty() bool {
	return len(v) == 0
}

func (v ByteString) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(v))
	return int64(n), err
}

func (v ByteString) CopyTo(b []byte) int {
	return copy(b, v)
}

func (v ByteString) ReadAt(b []byte, off int64) (n int, err error) {
	if n, err = VerifyReadAt(b, off, len(v)); err != nil || n == 0 {
		return n, err
	}
	return copy(b, v[off:]), nil
}

func (v ByteString) AsByteString() ByteString {
	return v
}

func (v ByteString) FixedByteSize() int {
	return len(v)
}

func (v ByteString) CutOutUint64() uint64 {
	folded := v.CutOutBits64()
	return folded.FoldToUint64()
}

func (v ByteString) FoldToUint64() uint64 {
	folded := v.FoldToBits64()
	return folded.FoldToUint64()
}

func (v ByteString) BitMasked(index int) (value byte, mask uint8) {
	bytePos, bitPos := v.BitPos(index)
	mask = 1 << bitPos
	return v[bytePos] & mask, mask
}

func (v ByteString) BitBool(index int) bool {
	if b, _ := v.BitMasked(index); b != 0 {
		return true
	}
	return false
}

func (v ByteString) BitValue(index int) byte {
	if b, _ := v.BitMasked(index); b != 0 {
		return 1
	}
	return 0
}

func (v ByteString) Byte(index int) byte {
	return v[index]
}

func (v ByteString) BitPos(index int) (bytePos int, bitPos uint8) {
	bytePos, bitPos = BitPos(index)
	if bytePos >= len(v) {
		panic("out of bounds")
	}
	return bytePos, bitPos
}

func (v ByteString) BitLen() int {
	return len(v) << 3
}

func (v ByteString) SearchBit(startAt int, bit bool) int {
	if startAt < 0 {
		panic("illegal value")
	}
	if startAt>>3 >= len(v) {
		return -1
	}

	if bit {
		return v.searchBit1(startAt)
	}
	return v.searchBit0(startAt)
}

func (v ByteString) searchBit1(startAt int) int {
	bytePos := startAt >> 3
	bitPos := byte(startAt & 0x7)

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

func (v ByteString) searchBit0(startAt int) int {
	bytePos := startAt >> 3
	bitPos := byte(startAt & 0x7)

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

func (v ByteString) CutOutBits64() (folded Bits64) {
	if len(v) <= len(folded) {
		copy(folded[:], v)
		return folded
	}

	for i := range folded {
		folded[i] = v[i*(len(v)-1)/(len(folded)-1)]
	}
	return folded
}

func (v ByteString) FoldToBits64() (folded Bits64) {
	if len(v) == 0 {
		return folded
	}

	alignedLen := len(v)
	alignedLen &^= len(folded) - 1
	copy(folded[:], v[alignedLen:])

	for i := 0; i < alignedLen; i += len(folded) {
		folded[0] ^= v[i+0]
		folded[1] ^= v[i+1]
		folded[2] ^= v[i+2]
		folded[3] ^= v[i+3]
		folded[4] ^= v[i+4]
		folded[5] ^= v[i+5]
		folded[6] ^= v[i+6]
		folded[7] ^= v[i+7]
	}
	return folded
}

func (v ByteString) String() string {
	return bitsToStringDefault(&v)
}

func (v ByteString) NewIoReader() io.Reader {
	return strings.NewReader(string(v))
}

func (v ByteString) Hex() string {
	if v == "" {
		return ""
	}
	b := make([]byte, hex.EncodedLen(len(v)))
	hex.Encode(b, []byte(v))
	return string(b)
}

func (v ByteString) Equal(other FixedReader) bool {
	if other == nil {
		return false
	}
	if s, ok := other.(ByteString); ok {
		return s == v
	}
	return Equal(v, other)
}
