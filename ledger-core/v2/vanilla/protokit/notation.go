// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package protokit

import "io"

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
	if err == nil {
		err = r.UnreadByte()
	}
	if err != nil {
		return ContentUndefined, err
	}
	return ContentTypeOf(b), nil
}
