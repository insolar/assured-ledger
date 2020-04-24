// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package protokit

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
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
