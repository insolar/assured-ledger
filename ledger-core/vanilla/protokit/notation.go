// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package protokit

import (
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

/*
	This is a notation that allows predictable parsing of any protobuf without a scheme:
		1. Textual strings are valid UTF-8 and do not start with codes less than LF (10)
		2. Any encoded protobuf message always starts with a field of id=16 (aka PolymorphFieldID)
		3. Any binary / non-parsable payload is prepended with 0 or BinaryMarker byte
*/
// TODO notation-aware pbuf parser/printer for protobuf without a scheme

const (
	illegalUtf8FirstByte byte = 0x80
	legalUtf8            byte = 0xC0

	ObjectMarker = illegalUtf8FirstByte | byte(WireVarint) // 16:varint
	//  _Marker		 = illegalUtf8FirstByte | byte(WireFixed64)	// 16:fixed64
	//  _Marker		 = illegalUtf8FirstByte | byte(WireBytes)	// 16:bytes
	//	DO_NOT_USE	 = illegalUtf8FirstByte | byte(WireStartGroup)	// 16:groupStart
	BinaryMarker = illegalUtf8FirstByte | byte(WireEndGroup) // 16:groupEnd
//  _Marker		 = illegalUtf8FirstByte | byte(WireFixed32)		// 16:fixed32
)

const PolymorphFieldID = illegalUtf8FirstByte >> WireTypeBits // = 16
const MaxSafeForPolymorphFieldID = (legalUtf8 >> WireTypeBits) - 1

// As a valid pbuf payload cant start with groupEnd tag, so we can use it as an indicator of a non-parsable payload.
// Number of BinaryMarkers is limited by valid UTF-8 codes, starting at 0xC0
const (
	GeneralBinaryMarker = BinaryMarker | iota<<WireTypeBits
	// _BinaryMarker
	// _BinaryMarker
	// _BinaryMarker
	// _BinaryMarker
	// _BinaryMarker
	// _BinaryMarker
	// _BinaryMarker
)

const MaxPolymorphFieldSize = 2 + MaxVarintSize
const MinPolymorphFieldSize = 2 + MinVarintSize

func GetPolymorphFieldSize(id uint64) int {
	return int(WireVarint.Tag(int(PolymorphFieldID)).FieldSize(id))
}

// Content type detection of a notation-friendly payload.
type ContentType uint8

const (
	/* Content is unclear */
	ContentUndefined ContentType = iota
	/* Content is text */
	ContentText
	/* Content is binary */
	ContentBinary
	/* Content is protobuf message */
	ContentMessage
	/* Content is protobuf that follows the notation and has polymorph marker */
	ContentPolymorph
)

type ContentTypeOptions uint8

const (
	// ContentOptionText indicates that the content can be text
	ContentOptionText ContentTypeOptions = 1 << iota

	// ContentOptionMessage indicates that the content can be a protobuf message
	ContentOptionMessage

	// ContentOptionNotation indicates the content has either Polymorph or Binary markers
	ContentOptionNotation
)

// Provides content type detection of a notation-friendly payload.
func PossibleContentTypes(firstByte byte) (ct ContentTypeOptions) {
	if firstByte == 0 {
		return 0
	}

	switch {
	case firstByte < '\t' /* 9 */ :
		// not a text
		if firstByte&^maskWireType == 0 {
			return 0
		}
	case firstByte >= legalUtf8:
		ct |= ContentOptionText
	case firstByte < illegalUtf8FirstByte:
		ct |= ContentOptionText
	}

	switch WireType(firstByte & maskWireType) {
	case WireVarint:
		if ct&ContentOptionText == 0 && firstByte >= illegalUtf8FirstByte {
			ct |= ContentOptionNotation
		}
		ct |= ContentOptionMessage
	case WireFixed64, WireBytes, WireFixed32, WireStartGroup:
		ct |= ContentOptionMessage
	case WireEndGroup:
		if ct&ContentOptionText == 0 {
			ct |= ContentOptionNotation
		}
	}
	return ct
}

func PeekPossibleContentTypes(r io.ByteScanner) (ContentTypeOptions, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	err = r.UnreadByte()
	return PossibleContentTypes(b), err
}

var ErrUnexpectedHeader = errors.New("unexpected header")

func PeekContentTypeAndPolymorphID(r io.ByteScanner) (ContentType, uint64, error) {
	b, err := r.ReadByte()
	if err != nil {
		return ContentUndefined, 0, err
	}

	ct := ContentUndefined
	switch pct := PossibleContentTypes(b); pct {
	case 0:
	case ContentOptionMessage:
		ct = ContentMessage
	case ContentOptionNotation:
		ct = ContentBinary
	case ContentOptionNotation | ContentOptionMessage:
		switch b, err = r.ReadByte(); {
		case err != nil:
			return ContentUndefined, 0, err
		case b != 0x01:
			return ContentMessage, 0, ErrUnexpectedHeader // we can't recover here
		}

		var id uint64
		if id, err = DecodeVarint(r); err != nil {
			return ContentUndefined, 0, err
		}
		return ContentPolymorph, id, nil

	default:
		if pct&ContentOptionText != 0 {
			ct = ContentText
			break
		}
		_ = r.UnreadByte()
		panic(throw.Impossible())
	}
	return ct, 0, r.UnreadByte()
}

func PeekContentTypeAndPolymorphIDFromBytes(b []byte) (ContentType, uint64, error) {
	n := len(b)
	if n == 0 {
		return ContentUndefined, 0, nil
	}

	switch pct := PossibleContentTypes(b[0]); pct {
	case 0:
		return ContentUndefined, 0, nil

	case ContentOptionMessage:
		if n == 1 || n == 2 && b[0] >= 0x80 {
			return ContentMessage, 0, throw.FailHere("bad message")
		}
		return ContentMessage, 0, nil

	case ContentOptionNotation:
		return ContentBinary, 0, nil

	case ContentOptionNotation | ContentOptionMessage:
		if n < 3 || b[1] != 0x01 {
			return ContentPolymorph, 0, throw.FailHere("bad message")
		}

		id, n := DecodeVarintFromBytes(b[2:])
		if n == 0 {
			return ContentPolymorph, 0, throw.FailHere("bad message")
		}
		return ContentPolymorph, id, nil

	default:
		switch {
		case pct&ContentOptionText == 0:
			panic(throw.Impossible())
		case b[0] < illegalUtf8FirstByte:
			return ContentText, 0, nil
		case n == 1:
			return ContentText, 0, throw.FailHere("bad utf")
		default:
			return ContentText, 0, nil
		}
	}
}

func DecodePolymorphFromBytes(b []byte, onlyVarint bool) (id uint64, size int, err error) {
	u, n := DecodeVarintFromBytes(b)
	if n == 0 {
		return 0, 0, throw.E("invalid wire tag, overflow")
	}
	wt, err := SafeWireTag(u)
	if err != nil {
		return 0, 0, err
	}
	switch fid := wt.FieldID(); {
	case fid == int(PolymorphFieldID):
	case fid < int(PolymorphFieldID) || fid > int(MaxSafeForPolymorphFieldID):
		return 0, 0, throw.E("invalid polymorph content")
	default:
		return 0, 0, nil
	}

	switch wt.Type() {
	case WireVarint:
	case WireFixed64, WireFixed32:
		if !onlyVarint {
			break
		}
		fallthrough
	default:
		return 0, 0, throw.E("unknown polymorph tag")
	}

	size = n
	id, n, err = wt.ReadValueFromBytes(b[n:])
	if err != nil {
		return 0, 0, err
	}
	return id, size + n, nil
}
