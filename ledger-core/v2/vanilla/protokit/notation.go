// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package protokit

import (
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
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

	ObjectMarker = illegalUtf8FirstByte | byte(WireVarint) //16:varint
	//  _Marker		 = illegalUtf8FirstByte | byte(WireFixed64)	//16:fixed64
	//  _Marker		 = illegalUtf8FirstByte | byte(WireBytes)	//16:bytes
	//	DO_NOT_USE	 = illegalUtf8FirstByte | byte(WireStartGroup)	//16:groupStart
	BinaryMarker = illegalUtf8FirstByte | byte(WireEndGroup) //16:groupEnd
//  _Marker		 = illegalUtf8FirstByte | byte(WireFixed32)	//16:fixed32
)

const PolymorphFieldID = illegalUtf8FirstByte >> WireTypeBits // = 16

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

// Content type detection of a notation-friendly payload.
type ContentType uint8

const (
	/* Content is unclear, can either be text or binary */
	ContentUndefined ContentType = iota
	/* Content is text */
	ContentText
	/* Content is binary */
	ContentBinary
	/* Content is protobuf that follows the notation, but doesn't have ObjectMarker */
	ContentMessage
	/* Content is protobuf that follows the notation and has ObjectMarker */
	ContentObject
)

// Provides content type detection of a notation-friendly payload.
func ContentTypeOf(firstByte byte) ContentType {
	switch {
	case firstByte < 10 /* LF */ :
		if firstByte == 0 {
			return ContentBinary
		}
		return ContentUndefined
	case firstByte >= legalUtf8:
		return ContentText
	case firstByte < illegalUtf8FirstByte:
		return ContentText
	}

	switch wt := firstByte & maskWireType; {
	case wt == ObjectMarker&maskWireType:
		return ContentObject
	case wt == BinaryMarker&maskWireType:
		return ContentBinary
	case WireType(wt).IsValid():
		return ContentMessage
	default:
		return ContentUndefined
	}
}

func PeekContentType(r io.ByteScanner) (ContentType, error) {
	b, err := r.ReadByte()
	if err != nil {
		return ContentUndefined, err
	}
	err = r.UnreadByte()
	return ContentTypeOf(b), err
}

var ErrUnexpectedHeader = errors.New("unexpected header")

func PeekContentTypeAndPolymorphID(r io.ByteScanner) (ContentType, uint64, error) {
	b, err := r.ReadByte()
	if err != nil {
		return ContentUndefined, 0, err
	}
	if ct := ContentTypeOf(b); ct != ContentObject {
		return ct, 0, r.UnreadByte()
	}
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
	return ContentObject, id, nil
}

func PeekContentTypeAndPolymorphIDFromBytes(b []byte) (ContentType, uint64, error) {
	n := len(b)
	if n == 0 {
		return ContentUndefined, 0, nil
	}
	ct := ContentTypeOf(b[0])

	switch ct {
	case ContentBinary, ContentUndefined:
		return ct, 0, nil
	case ContentText:
		if n == 1 && b[0] >= illegalUtf8FirstByte {
			// only one byte can't be >=0x80
			return ContentText, 0, throw.FailHere("bad utf")
		}
		return ContentText, 0, nil
	case ContentMessage:
		if n == 1 || n == 2 && b[0] >= 0x80 {
			return ContentMessage, 0, throw.FailHere("bad message")
		}
		return ContentMessage, 0, nil
	case ContentObject:
		switch {
		case n < 3 || b[1] != 0x01:
		default:
			id, n := DecodeVarintFromBytes(b[2:])
			if n == 0 {
				break
			}
			return ContentObject, id, nil
		}
		return ContentObject, 0, throw.FailHere("bad message")
	default:
		panic(throw.Impossible())
	}
}
