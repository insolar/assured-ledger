// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ io.Writer = &Header{}
var _ io.Reader = &Header{}

/*
	ByteSize=16
*/
type Header struct {
	/*
		Functions of TargetID, SourceID and ReceiverID depends on ProtocolType
	*/
	ReceiverID uint32 // NB! For hash/signature calculation MUST be considered as 0, actual value can be different

	ProtocolAndPacketType  uint8  `insolar-transport:"[0:3]=header:Packet;[4:7]=header:Protocol"` // [00-03]PacketType [04-07]ProtocolType
	PacketFlags            uint8  `insolar-transport:"[0]=IsRelayRestricted;[1]=IsBodyEncrypted;[2:]=flags:PacketFlags"`
	HeaderAndPayloadLength uint16 // depends on bitsByteLength[ProtocolType], [15]=IsExcessiveLength
	SourceID               uint32 // may differ from actual sender when relay is in use, MUST NOT =0
	TargetID               uint32 // indicates final destination, if =0 then there is no relay allowed by sender and receiver MUST decline a packet if actual sender != source
}

type ProtocolType uint8

const (
	ProtocolUnknown ProtocolType = iota
	ProtocolTypePulsar
	ProtocolTypeGlobulaConsensus
	ProtocolTypeNodeMessage
)

var maskByteLength = []uint16{
	ProtocolTypePulsar:           1<<11 - 1,
	ProtocolTypeGlobulaConsensus: 1<<11 - 1,
	ProtocolTypeNodeMessage:      math.MaxUint16, // can use excessive lengths
}

const (
	packetTypeBitSize = 4
	packetTypeMask    = 1<<packetTypeBitSize - 1 // 0b00001111
	PacketTypeMax     = packetTypeMask

	protocolTypeBitSize = 8 - packetTypeBitSize
	protocolTypeShift   = protocolTypeBitSize
	ProtocolTypeMax     = 1<<protocolTypeBitSize - 1
	protocolTypeMask    = ProtocolTypeMax << protocolTypeShift

	payloadLengthBits          = 15
	payloadExcessiveLengthMask = 1 << payloadLengthBits
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
	return h.ProtocolAndPacketType >= 1<<protocolTypeShift
}

func (h *Header) IsZero() bool {
	return h.ProtocolAndPacketType == 0
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

func (h *Header) GetPayloadLength() uint16 {
	if h.HeaderAndPayloadLength < payloadExcessiveLengthMask {
		return h.HeaderAndPayloadLength // & maskByteLength[h.GetProtocolType()]
	}
	panic(throw.IllegalState())
}

func (h *Header) SetPayloadLength(payloadLength uint16) {
	mask := maskByteLength[h.GetProtocolType()]

	if payloadLength > mask {
		panic(throw.IllegalValue())
	}
	h.HeaderAndPayloadLength = payloadLength
}

func (h *Header) GetExcessivePayloadLength() (uint16, bool) {
	return h.HeaderAndPayloadLength &^ payloadExcessiveLengthMask, h.HeaderAndPayloadLength >= payloadExcessiveLengthMask
}

func (h *Header) SetExcessivePayloadLength(payloadLength uint) uint {
	if maskByteLength[h.GetProtocolType()] != math.MaxUint16 {
		panic(throw.IllegalState())
	}
	h.HeaderAndPayloadLength = uint16(payloadLength | payloadExcessiveLengthMask)
	return payloadLength >> payloadLengthBits
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

func (h *Header) Write(b []byte) (n int, err error) {
	if len(b) < 16 {
		return 0, io.ErrUnexpectedEOF
	}

	byteOrder := binary.LittleEndian
	_ = b[15]

	h.ReceiverID = byteOrder.Uint32(b)
	h.ProtocolAndPacketType = b[4]
	h.PacketFlags = b[5]
	h.HeaderAndPayloadLength = byteOrder.Uint16(b[6:])
	h.SourceID = byteOrder.Uint32(b[8:])
	h.TargetID = byteOrder.Uint32(b[12:])
	return 16, nil
}

func (h *Header) Read(b []byte) (n int, err error) {
	switch {
	case !h.IsValid():
		return 0, throw.IllegalState()
	case len(b) == 0:
		return 0, nil
	case len(b) < 16:
		return 0, io.ErrShortBuffer
	}

	byteOrder := binary.LittleEndian
	_ = b[15]

	byteOrder.PutUint32(b, h.ReceiverID)
	b[4] = h.ProtocolAndPacketType
	b[5] = h.PacketFlags
	byteOrder.PutUint16(b[6:], h.HeaderAndPayloadLength)
	byteOrder.PutUint32(b[8:], h.SourceID)
	byteOrder.PutUint32(b[12:], h.TargetID)
	return 16, nil
}
