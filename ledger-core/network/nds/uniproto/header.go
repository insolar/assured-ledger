// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	HeaderByteSizeMin = 16
	HeaderByteSizeMax = HeaderByteSizeMin + 4
)

/*
	ByteSize=16-20
*/
// Header represents a header of uniproto packet.
// It is non-encrypted part of a packet with minimal set of identities, flags and packet size.
type Header struct {
	// Functions of TargetID, SourceID and ReceiverID depends on ProtocolType

	ReceiverID uint32 // NB! For hash/signature calculation MUST be considered as 0, actual value can be different

	ProtocolAndPacketType  uint8  `insolar-transport:"[0:3]=header:Packet;[4:7]=header:Protocol"` // [00-03]PacketType [04-07]ProtocolType
	PacketFlags            uint8  `insolar-transport:"[0]=IsRelayRestricted;[1]=IsBodyEncrypted;[2:]=flags:PacketFlags"`
	HeaderAndPayloadLength uint16 `insolar-transport:"[14]=reserved;[15]=IsExcessiveLength"`
	SourceID               uint32 // may differ from actual sender when relay is in use
	TargetID               uint32 // indicates final destination, if IsRelayRestricted then there is no relay allowed by sender and receiver MUST decline a packet if actual sender != source

	// <<<< here is end of HeaderByteSizeMin

	ExcessiveLength uint32 `insolar-transport:"optional=IsExcessiveLength"`
}

/*
	Invalid combinations of Receiver, Source and Target

	R	S	T	Description
	-----------------------
	0	_	N	Invalid
	_	N	N	Invalid, loopback
	N	N	_   Invalid, loopback
	_	_	0	Invalid without OptionalTarget
	_   N   _   Invalid with NoSourceID
	_	0	_	Invalid without NoSourceID
	N	_	K	Invalid when IsRelayRestricted == true
*/

// ProtocolType defines a type of a protocol for a packet. Can be [0, 15].
// ATTENTION! To provide compatibility with HTTP GET, PUT and POST following restrictions apply
// 1) "POST /", "HEAD /" - Protocol=2, Packet=0 must have PacketFlags[5:] = 0
// 2) "GET /", "PUT /" Protocol=2, Packet=0x0F is forbidden
// 3) "OPTION" - Protocol=4, Packet=0xF must have PacketFlags[6:] = 0

//go:generate stringer -type=ProtocolType
type ProtocolType uint8

const (
	ProtocolTypePulsar ProtocolType = iota
	ProtocolTypeGlobulaConsensus
	ProtocolTypeJoinCandidate
	ProtocolTypeMessageDelivery
)

const ProtocolTypeInterprocess = ProtocolTypePulsar

const (
	packetTypeBitSize = 4
	PacketTypeCount   = 1 << packetTypeBitSize
	PacketTypeMax     = PacketTypeCount - 1
	packetTypeMask    = PacketTypeMax

	protocolTypeBitSize = 8 - packetTypeBitSize
	protocolTypeShift   = protocolTypeBitSize
	ProtocolTypeCount   = 1 << protocolTypeBitSize
	ProtocolTypeMax     = ProtocolTypeCount - 1
	protocolTypeMask    = ProtocolTypeMax << protocolTypeShift

	payloadLengthBits   = 14
	payloadLengthMask   = 1<<payloadLengthBits - 1
	unlimitedLengthFlag = 1 << 14 // reserved
	excessiveLengthFlag = 1 << 15

	MinLengthBits         = 5
	SmallLengthBits       = payloadLengthBits
	ExcessiveLengthBits   = SmallLengthBits + 32
	MaxLengthBits         = ExcessiveLengthBits
	MaxNonExcessiveLength = 1<<payloadLengthBits - 1

//	MaxNonExcessivePayloadLength = MaxNonExcessiveLength - HeaderByteSizeMin
)

type FlagIndex uint8

const (
	flagIsRelayRestricted = 0
	flagIsBodyEncrypted   = 1
)

const (
	reservedFlagSize = 2
	maxFlagIndex     = 5
)

func (h *Header) IsValid() bool {
	return h.HeaderAndPayloadLength != 0
}

func (h *Header) IsZero() bool {
	return h.ProtocolAndPacketType == 0 && h.HeaderAndPayloadLength == 0
}

func (h *Header) IsTargeted() bool {
	return h.TargetID != 0
}

func (h *Header) IsForRelay() bool {
	return h.ReceiverID != 0 && h.TargetID != 0 && h.TargetID != h.ReceiverID
}

func (h *Header) GetProtocolType() ProtocolType {
	return ProtocolType((h.ProtocolAndPacketType & protocolTypeMask) >> protocolTypeShift)
}

func (h *Header) SetProtocolType(protocolType ProtocolType) {
	if protocolType > ProtocolTypeMax {
		panic(throw.IllegalValue())
	}
	h.ProtocolAndPacketType = uint8(protocolType<<protocolTypeShift) | h.ProtocolAndPacketType&packetTypeMask
}

func (h *Header) GetPacketType() uint8 {
	return h.ProtocolAndPacketType & packetTypeMask
}

func (h *Header) SetPacketType(packetType uint8) {
	if packetType > PacketTypeMax {
		panic(throw.IllegalValue())
	}
	h.ProtocolAndPacketType = packetType | h.ProtocolAndPacketType&protocolTypeMask
}

func (h *Header) GetFullLength() (uint64, error) {
	switch {
	case h.HeaderAndPayloadLength < HeaderByteSizeMin:
		return 0, throw.IllegalValue()
	case h.HeaderAndPayloadLength <= payloadLengthMask:
		return uint64(h.HeaderAndPayloadLength), nil
	case h.ExcessiveLength == 0:
		return 0, throw.IllegalValue()
	}
	return CalcExcessivePayloadLength(h.HeaderAndPayloadLength&payloadLengthMask, h.ExcessiveLength), nil
}

func (h *Header) GetPayloadLength() (uint64, error) {
	l, err := h.GetFullLength()
	if err != nil {
		return 0, err
	}
	if sz := uint64(h.ByteSize()); l >= sz {
		return l - sz, nil
	}
	return 0, throw.IllegalValue()
}

func CalcExcessivePayloadLength(baseLength uint16, excessive uint32) uint64 {
	return uint64(baseLength) | uint64(excessive)<<payloadLengthBits
}

func (h *Header) SetPayloadLength(payloadLength uint64) uint64 {
	payloadLength += HeaderByteSizeMin
	if payloadLength <= MaxNonExcessiveLength {
		h.HeaderAndPayloadLength = uint16(payloadLength)
	} else {
		payloadLength += 4
		h.HeaderAndPayloadLength = uint16(payloadLength&payloadLengthMask | excessiveLengthFlag)
		h.ExcessiveLength = uint32(payloadLength >> payloadLengthBits)
	}
	return payloadLength
}

func (h *Header) GetPacketSize(dataSize uint, sigSize uint) uint {
	if dataSize > MaxNonExcessiveLength {
		dataSize += 4
		dataSize += sigSize
	}
	dataSize += sigSize
	dataSize += HeaderByteSizeMin
	return dataSize
}

func (h *Header) IsExcessiveLength() bool {
	return h.HeaderAndPayloadLength&excessiveLengthFlag != 0
}

func (h *Header) HasFlag(f FlagIndex) bool {
	if f > maxFlagIndex {
		panic(throw.IllegalValue())
	}
	return h.hasFlag(f + reservedFlagSize)
}

func (h *Header) GetFlagRangeInt(from, to uint8) uint8 {
	switch {
	case from > to:
		panic(throw.IllegalValue())
	case to > maxFlagIndex:
		panic(throw.IllegalValue())
	}
	mask := uint8(1)<<(to-from+1) - 1
	return (h.PacketFlags >> (reservedFlagSize + from)) & mask
}

func (h *Header) SetFlag(f FlagIndex, v bool) {
	if f > maxFlagIndex {
		panic(throw.IllegalValue())
	}
	h.setFlag(f+reservedFlagSize, v)
}

func (h *Header) IsRelayRestricted() bool {
	return h.hasFlag(flagIsRelayRestricted)
}

func (h *Header) SetRelayRestricted(isRestricted bool) {
	h.setFlag(flagIsRelayRestricted, isRestricted)
}

func (h *Header) IsBodyEncrypted() bool {
	return h.hasFlag(flagIsBodyEncrypted)
}

func (h *Header) SetBodyEncrypted(encrypted bool) {
	h.setFlag(flagIsBodyEncrypted, encrypted)
}

func (h *Header) setFlag(f FlagIndex, val bool) {
	if val {
		h.PacketFlags |= 1 << f
	} else {
		h.PacketFlags &^= 1 << f
	}
}

func (h *Header) hasFlag(f FlagIndex) bool {
	return h.PacketFlags&(1<<f) != 0
}

func (h *Header) ByteSize() uint {
	if h.HeaderAndPayloadLength&excessiveLengthFlag != 0 {
		return HeaderByteSizeMax
	}
	return HeaderByteSizeMin
}

func (h *Header) DeserializeFromBytes(b []byte) (uint, error) {
	if err := h.DeserializeMinFromBytes(b); err != nil {
		return 0, err
	}
	switch fullSize := h.ByteSize(); {
	case fullSize == HeaderByteSizeMin:
		return fullSize, nil
	case fullSize != HeaderByteSizeMax:
		return 0, throw.Impossible()
	default:
		if err := h.deserializeExtraFromBytes(b[HeaderByteSizeMin:]); err != nil {
			return 0, err
		}
		return fullSize, nil
	}
}

// ErrPossibleHTTPRequest is returned when a protected HTTP header is detected
var ErrPossibleHTTPRequest = errors.New("possible HTTP request")
var DefaultByteOrder = binary.BigEndian

func (h *Header) DeserializeMinFromBytes(b []byte) error {
	_ = b[HeaderByteSizeMin-1]

	switch b[4] {
	case ' ':
		// Compatibility with HTTP: "POST /" and "HEAD /"
		if b[5] >= ' ' {
			return ErrPossibleHTTPRequest
		}
	case '/':
		// Compatibility with HTTP: "PUT /" and "GET /"
		return ErrPossibleHTTPRequest
	case 'O':
		// Compatibility with HTTP: "OPTION"
		if b[5] >= 0x40 {
			return ErrPossibleHTTPRequest
		}
	}

	h.ReceiverID = DefaultByteOrder.Uint32(b)
	h.ProtocolAndPacketType = b[4]
	h.PacketFlags = b[5]
	h.HeaderAndPayloadLength = DefaultByteOrder.Uint16(b[6:])
	h.SourceID = DefaultByteOrder.Uint32(b[8:])
	h.TargetID = DefaultByteOrder.Uint32(b[12:])

	if h.HeaderAndPayloadLength&unlimitedLengthFlag != 0 {
		return throw.Unsupported()
	}
	return nil
}

func (h *Header) DeserializeRestFromBytes(b []byte) (int, error) {
	_ = b[HeaderByteSizeMax-1]
	if !h.IsExcessiveLength() {
		return HeaderByteSizeMin, nil
	}
	if err := h.deserializeExtraFromBytes(b[HeaderByteSizeMin:]); err != nil {
		return 0, err
	}
	return HeaderByteSizeMax, nil
}

func (h *Header) deserializeExtraFromBytes(b []byte) error {
	h.ExcessiveLength = DefaultByteOrder.Uint32(b[0:])
	if h.ExcessiveLength == 0 {
		return throw.Violation("zero excessive length")
	}
	return nil
}

func (h *Header) SerializeToBytes(b []byte) uint {
	if !h.IsValid() {
		panic(throw.IllegalState())
	}
	_ = b[HeaderByteSizeMin-1]

	switch h.ProtocolAndPacketType {
	case ' ':
		// Compatibility with HTTP: "POST /" and "HEAD /"
		if h.PacketFlags >= ' ' {
			panic(ErrPossibleHTTPRequest)
		}
	case '/':
		// Compatibility with HTTP: "PUT /" and "GET /"
		panic(ErrPossibleHTTPRequest)
	}

	DefaultByteOrder.PutUint32(b, h.ReceiverID)
	b[4] = h.ProtocolAndPacketType
	b[5] = h.PacketFlags
	DefaultByteOrder.PutUint16(b[6:], h.HeaderAndPayloadLength)
	DefaultByteOrder.PutUint32(b[8:], h.SourceID)
	DefaultByteOrder.PutUint32(b[12:], h.TargetID)
	size := h.ByteSize()
	if size > HeaderByteSizeMin {
		if h.ExcessiveLength == 0 {
			panic(throw.IllegalState())
		}
		DefaultByteOrder.PutUint32(b[HeaderByteSizeMin:], h.ExcessiveLength)
	}
	return size
}

func (h *Header) DeserializeFrom(reader io.Reader) error {
	b := make([]byte, HeaderByteSizeMin)
	if _, err := io.ReadAtLeast(reader, b, HeaderByteSizeMin); err != nil {
		return err
	}
	if err := h.DeserializeMinFromBytes(b); err != nil {
		return err
	}

	switch fullSize := h.ByteSize(); {
	case fullSize == HeaderByteSizeMin:
		//
	case fullSize != HeaderByteSizeMax:
		return throw.Impossible()
	default:
		n := int(fullSize - HeaderByteSizeMin)
		if _, err := io.ReadAtLeast(reader, b, n); err != nil {
			return err
		}
		if err := h.deserializeExtraFromBytes(b[:n]); err != nil {
			return err
		}
	}
	return nil
}

func (Header) GetHashingZeroPrefix() int {
	return 4
}
