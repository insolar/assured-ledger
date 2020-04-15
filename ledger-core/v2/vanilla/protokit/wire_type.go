// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package protokit

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	WireTypeBits  = 3
	MaxVarintSize = binary.MaxVarintLen64
	MaxFieldID    = math.MaxUint32 >> WireTypeBits
	maskWireType  = 1<<WireTypeBits - 1
)

type WireType uint8

const (
	WireVarint WireType = iota
	WireFixed64
	WireBytes
	WireStartGroup
	WireEndGroup
	WireFixed32

	MaxWireType = iota - 1
)

type UintDecoderFunc func(io.ByteReader) (uint64, error)
type UintEncoderFunc func(io.ByteWriter, uint64) error

var wireTypes = []struct {
	name             string
	decodeFn         UintDecoderFunc
	encodeFn         UintEncoderFunc
	fieldSizeFn      func(uint64) uint64
	minSize, maxSize int8
}{
	WireFixed64: {"fixed64", DecodeFixed64, EncodeFixed64,
		nil, 8, 8},

	WireFixed32: {"fixed32", DecodeFixed32, func(w io.ByteWriter, u uint64) error {
		if u > math.MaxUint32 {
			panic(errOverflow)
		}
		return EncodeFixed32(w, uint32(u))
	}, nil, 4, 4},

	WireVarint: {"varint", DecodeVarint, EncodeVarint, func(u uint64) uint64 {
		return uint64(SizeVarint64(u))
	}, 1, MaxVarintSize},

	WireBytes: {"bytes", DecodeVarint, EncodeVarint, func(u uint64) uint64 {
		return uint64(SizeVarint64(u)) + u
	}, 1, -1},

	WireStartGroup: {name: "groupStart"},
	WireEndGroup:   {name: "groupEnd"},
}

func (v WireType) IsValid() bool {
	return v <= MaxWireType
}

func (v WireType) Tag(fieldID int) WireTag {
	if fieldID <= 0 || fieldID > MaxFieldID {
		panic("illegal value")
	}
	return WireTag(fieldID<<WireTypeBits | int(v))
}

const maxMask = int(^uint(0) >> 1)

func (v WireType) DataSize() (minSize, maxSize int) {
	min, max := int(wireTypes[v].minSize), int(wireTypes[v].maxSize)
	if min <= 0 {
		panic("illegal value")
	}
	return min, max & maxMask /* converts -1 to maxInt */
}

func (v WireType) FieldSize(tagSize int, u uint64) uint64 {
	sizeFn := wireTypes[v].fieldSizeFn
	if sizeFn != nil {
		return sizeFn(u) + uint64(tagSize)
	}
	if wireTypes[v].minSize == 0 {
		panic("illegal value")
	}
	return uint64(wireTypes[v].minSize) + uint64(tagSize)
}

func (v WireType) UintDecoder() (UintDecoderFunc, error) {
	if v <= MaxWireType {
		return wireTypes[v].decodeFn, nil
	}
	return nil, fmt.Errorf("unsupported wire type %x", v)
}

func (v WireType) UintEncoder() (UintEncoderFunc, error) {
	if v <= MaxWireType {
		return wireTypes[v].encodeFn, nil
	}
	return nil, fmt.Errorf("unsupported wire type %x", v)
}

func (v WireType) String() string {
	if v <= MaxWireType {
		if s := wireTypes[v].name; s != "" {
			return s
		}
	}
	return fmt.Sprintf("unknown%d", v)
}

func (v WireType) ReadValue(r io.ByteReader) (uint64, error) {
	decodeFn, err := v.UintDecoder()
	if err != nil {
		return 0, err
	}
	return decodeFn(r)
}

func (v WireType) WriteValue(w io.ByteWriter, u uint64) error {
	encodeFn, err := v.UintEncoder()
	if err != nil {
		return err
	}
	return encodeFn(w, u)
}
