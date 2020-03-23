// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type SerializationContext interface {
	PrepareHeader(*Header) (pulse.Number, error)
	GetPayloadSigner() cryptkit.DataSigner
	GetPayloadEncrypter() cryptkit.Encrypter
}

type DeserializationContext interface {
	VerifyHeader(*Header, pulse.Number) error
	GetPayloadVerifier() cryptkit.DataSignatureVerifier
	GetPayloadDecrypter() cryptkit.Decrypter
	GetPayloadFactory() DeserializationFactory
}

type Serializer interface {
	SerializeTo(SerializationContext, io.Writer) error
}

type Deserializer interface {
	DeserializeFrom(DeserializationContext, io.Reader) error
}

type SizeAwareSerializer interface {
	ByteSize() uint
	SerializeTo(ctx SerializationContext, writer *iokit.LimitedWriter) error
}

type SizeAwareDeserializer interface {
	DeserializeFrom(ctx DeserializationContext, reader *iokit.LimitedReader) error
}

type Serializable interface {
	SizeAwareSerializer
	SizeAwareDeserializer
}

type PayloadCompleteness uint8

const (
	CompletePayload PayloadCompleteness = iota
	BodyPayload
	HeadPayload
)

type DeserializationFactory interface {
	DeserializePayloadFrom(ctx DeserializationContext, mode PayloadCompleteness, reader *iokit.LimitedReader) (Serializable, error)
}

const (
	HeaderByteSizeMin = 16
	HeaderByteSizeMax = HeaderByteSizeMin + 4
)

/*
	ByteSize=16-20-24
*/
type Header struct {
	/*
		Functions of TargetID, SourceID and ReceiverID depends on ProtocolType
	*/
	ReceiverID uint32 // NB! For hash/signature calculation MUST be considered as 0, actual value can be different

	ProtocolAndPacketType  uint8  `insolar-transport:"[0:3]=header:Packet;[4:7]=header:Protocol"` // [00-03]PacketType [04-07]ProtocolType
	PacketFlags            uint8  `insolar-transport:"[0]=IsRelayRestricted;[1]=IsBodyEncrypted;[2:]=flags:PacketFlags"`
	HeaderAndPayloadLength uint16 `insolar-transport:"[15]=IsExcessiveLength"` // depends on bitsByteLength[ProtocolType], [15]=IsExcessiveLength
	SourceID               uint32 // may differ from actual sender when relay is in use, MUST NOT =0
	TargetID               uint32 // indicates final destination, if =0 then there is no relay allowed by sender and receiver MUST decline a packet if actual sender != source
	ExcessiveLength        uint32 `insolar-transport:"optional=IsExcessiveLength"`
	//	PulseNumber            pulse.Number `insolar-transport:"send=ignore"` // depends on protocol
}

type ProtocolType uint8

const (
	ProtocolUnknown ProtocolType = iota
	ProtocolTypeGlobulaConsensus
	ProtocolTypePulsar
	ProtocolTypeNodeMessage
)

const (
	packetTypeBitSize = 4
	packetTypeMask    = 1<<packetTypeBitSize - 1 // 0b00001111
	PacketTypeMax     = packetTypeMask

	protocolTypeBitSize = 8 - packetTypeBitSize
	protocolTypeShift   = protocolTypeBitSize
	ProtocolTypeMax     = 1<<protocolTypeBitSize - 1
	protocolTypeMask    = ProtocolTypeMax << protocolTypeShift

	payloadLengthBits = 15
	payloadLengthFlag = 1 << payloadLengthBits
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

func (h *Header) GetPayloadLength() (uint64, error) {
	if h.HeaderAndPayloadLength < payloadLengthFlag {
		return uint64(h.HeaderAndPayloadLength), nil
	}
	if h.ExcessiveLength == 0 {
		return 0, throw.IllegalValue()
	}
	return CalcExcessivePayloadLength(h.HeaderAndPayloadLength&(payloadLengthFlag-1), h.ExcessiveLength), nil
}

func CalcExcessivePayloadLength(baseLength uint16, excessive uint32) uint64 {
	return uint64(baseLength) | uint64(excessive)<<payloadLengthBits
}

func (h *Header) SetPayloadLength(payloadLength uint64) uint64 {
	payloadLength += HeaderByteSizeMin
	if payloadLength < payloadLengthFlag {
		h.HeaderAndPayloadLength = uint16(payloadLength)
	} else {
		payloadLength += 4
		h.HeaderAndPayloadLength = uint16(payloadLength | payloadLengthFlag)
		h.ExcessiveLength = uint32(payloadLength >> payloadLengthBits)
	}
	return payloadLength
}

func (h *Header) IsExcessiveLength() bool {
	return h.HeaderAndPayloadLength&payloadLengthFlag != 0
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
	if h.HeaderAndPayloadLength&payloadLengthFlag != 0 {
		return HeaderByteSizeMax
	}
	return HeaderByteSizeMin
}

func (h *Header) DeserializeFromBytes(b []byte) (uint, error) {
	if err := h.deserializeMinFromBytes(b); err != nil {
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

func (h *Header) deserializeMinFromBytes(b []byte) error {
	byteOrder := binary.LittleEndian
	_ = b[HeaderByteSizeMin-1]

	h.ReceiverID = byteOrder.Uint32(b)
	h.ProtocolAndPacketType = b[4]
	h.PacketFlags = b[5]
	h.HeaderAndPayloadLength = byteOrder.Uint16(b[6:])
	h.SourceID = byteOrder.Uint32(b[8:])
	h.TargetID = byteOrder.Uint32(b[12:])
	return nil
}

func (h *Header) deserializeExtraFromBytes(b []byte) error {
	byteOrder := binary.LittleEndian
	h.ExcessiveLength = byteOrder.Uint32(b[0:])
	if h.ExcessiveLength == 0 {
		return throw.IllegalState()
	}
	return nil
}

func (h *Header) SerializeToBytes(b []byte) (uint, error) {
	if !h.IsValid() {
		return 0, throw.IllegalState()
	}
	_ = b[HeaderByteSizeMin-1]

	byteOrder := binary.LittleEndian
	byteOrder.PutUint32(b, h.ReceiverID)
	b[4] = h.ProtocolAndPacketType
	b[5] = h.PacketFlags
	byteOrder.PutUint16(b[6:], h.HeaderAndPayloadLength)
	byteOrder.PutUint32(b[8:], h.SourceID)
	byteOrder.PutUint32(b[12:], h.TargetID)
	size := h.ByteSize()
	if size > HeaderByteSizeMin {
		if h.ExcessiveLength == 0 {
			return 0, throw.IllegalState()
		}
		byteOrder.PutUint32(b, h.ExcessiveLength)
	}
	return size, nil
}

func (h *Header) SerializeTo(writer io.Writer) error {
	if !h.IsValid() {
		return throw.IllegalState()
	}
	b := make([]byte, HeaderByteSizeMax)
	sz, err := h.SerializeToBytes(b)
	if err != nil {
		return err
	}

	switch n, err := writer.Write(b[:sz]); {
	case err != nil:
		return err
	case uint(n) != sz:
		return io.ErrShortWrite
	}
	return nil
}

func (h *Header) DeserializeFrom(reader io.Reader) error {
	b := make([]byte, HeaderByteSizeMin)
	if _, err := io.ReadAtLeast(reader, b, HeaderByteSizeMin); err != nil {
		return err
	}
	if err := h.deserializeMinFromBytes(b); err != nil {
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
