package protokit

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func SizeTag(fieldID int) int {
	switch {
	case fieldID <= 0:
	case fieldID > MaxFieldID:
	default:
		return SizeVarint32(uint32(fieldID))
	}
	panic(throw.IllegalValue())
}

func SizeVarint32(x uint32) int {
	switch {
	case x < 1<<7:
		return 1
	case x < 1<<14:
		return 2
	case x < 1<<21:
		return 3
	case x < 1<<28:
		return 4
	}
	return 5
}

// SizeVarint returns the varint encoding size of an integer.
func SizeVarint64(x uint64) int {
	switch {
	case x < 1<<7:
		return 1
	case x < 1<<14:
		return 2
	case x < 1<<21:
		return 3
	case x < 1<<28:
		return 4
	case x < 1<<35:
		return 5
	case x < 1<<42:
		return 6
	case x < 1<<49:
		return 7
	case x < 1<<56:
		return 8
	case x < 1<<63:
		return 9
	}
	return 10
}

func EncodeVarint(w io.ByteWriter, u uint64) error {
	for u > 0x7F {
		if err := w.WriteByte(byte(u | 0x80)); err != nil {
			return err
		}
		u >>= 7
	}
	return w.WriteByte(byte(u))
}

func EncodeVarintToBytes(b []byte, u uint64) (n int) {
	for u > 0x7F {
		b[n] = byte(u | 0x80)
		n++
		u >>= 7
	}
	b[n] = byte(u)
	n++
	return
}

func EncodeVarintToBytesWithError(b []byte, u uint64) (n int, err error) {
	for u > 0x7F {
		if n >= len(b) {
			return 0, io.ErrShortBuffer
		}
		b[n] = byte(u | 0x80)
		n++
		u >>= 7
	}
	if n >= len(b) {
		return 0, io.ErrShortBuffer
	}
	b[n] = byte(u)
	n++
	return
}

func EncodeFixed64(w io.ByteWriter, u uint64) error {
	if err := EncodeFixed32(w, uint32(u)); err != nil {
		return err
	}
	return EncodeFixed32(w, uint32(u>>32))
}

func EncodeFixed32(w io.ByteWriter, u uint32) error {
	if err := w.WriteByte(byte(u)); err != nil {
		return err
	}
	u >>= 8
	if err := w.WriteByte(byte(u)); err != nil {
		return err
	}
	u >>= 8
	if err := w.WriteByte(byte(u)); err != nil {
		return err
	}
	u >>= 8
	if err := w.WriteByte(byte(u)); err != nil {
		return err
	}
	return nil
}

func EncodeFixed64ToBytes(b []byte, u uint64) (int, error) {
	if len(b) < 8 {
		return 0, io.ErrShortBuffer
	}

	b[0] = byte(u)
	b[1] = byte(u >> 8)
	b[2] = byte(u >> 16)
	b[3] = byte(u >> 24)
	b[4] = byte(u >> 32)
	b[5] = byte(u >> 40)
	b[6] = byte(u >> 48)
	b[7] = byte(u >> 56)
	return 8, nil
}

func EncodeFixed32ToBytes(b []byte, u uint32) (int, error) {
	if len(b) < 4 {
		return 0, io.ErrShortBuffer
	}

	b[0] = byte(u)
	b[1] = byte(u >> 8)
	b[2] = byte(u >> 16)
	b[3] = byte(u >> 24)
	return 4, nil
}
