// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const Protocol = uniproto.ProtocolTypeMessageDelivery

var protoDescriptor = uniproto.Descriptor{
	SupportedPackets: [16]uniproto.PacketDescriptor{
		DeliveryState:      {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.SmallLengthBits},
		DeliveryParcelHead: {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.SmallLengthBits},
		DeliveryParcelBody: {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.MaxLengthBits},
	},
}

const maxStatePacketDataSize = uniproto.MaxNonExcessiveLength - uniproto.PacketByteSizeMin - 512 /* reserved for signature */

type PacketType uint8

const (
	DeliveryState PacketType = iota
	DeliveryParcelHead
	DeliveryParcelBody // or Head + Body
)

func NewPacket(tp PacketType) uniproto.Packet {
	pt := uniproto.Packet{}
	pt.Header.SetProtocolType(Protocol)
	pt.Header.SetPacketType(uint8(tp))
	return pt
}

// Flags for DeliveryState
const (
	BodyRqFlag uniproto.FlagIndex = iota
	BodyRqRejectListFlag
	BodyAckListFlag
	RejectListFlag
)

type StatePacket struct {
	// TODO the problem - BodyRq has to be retried

	// From receiver to sender
	BodyRq ShortShipmentID `insolar-transport:"optional=PacketFlags[0]"` //
	// From sender to receiver
	BodyRqRejectList []ShortShipmentID `insolar-transport:"optional=PacketFlags[1]"` // TODO serialization / deser
	// From receiver to sender
	BodyAckList []ShortShipmentID `insolar-transport:"optional=PacketFlags[2]"` // TODO serialization / deser
	// From receiver to sender
	RejectList []ShortShipmentID `insolar-transport:"optional=PacketFlags[3]"` //
	// From receiver to sender
	AckList []ShortShipmentID `insolar-transport:"list=nocount"` // length is determined by packet size
}

func (p *StatePacket) remainingSpace(maxSize int) int {
	total := 1 // for p.BodyRq
	if n := len(p.AckList); n > 0 {
		total += n
	}
	if n := len(p.BodyAckList); n > 0 {
		total += n + 1 // +1 for list length
	}
	if n := len(p.RejectList); n > 0 {
		total += n + 1 // +1 for list length
	}
	return maxSize/ShortShipmentIDByteSize - total
}

func (p *StatePacket) isEmpty() bool {
	return len(p.AckList) == 0 && p.BodyRq == 0 && len(p.BodyAckList) == 0 && len(p.RejectList) == 0
}

func (p *StatePacket) SerializeTo(ctx uniproto.SerializationContext, writer io.Writer) error {
	packet := NewPacket(DeliveryState)

	size := uint(0)

	if p.BodyRq != 0 {
		packet.Header.SetFlag(BodyRqFlag, true)
		size += ShortShipmentIDByteSize
	}
	if n := len(p.RejectList); n > 0 {
		packet.Header.SetFlag(RejectListFlag, true)
		size += uint(protokit.SizeVarint64(uint64(n)))
	}
	size += uint(len(p.AckList)) * ShortShipmentIDByteSize
	if size == 0 {
		return throw.IllegalState()
	}

	return packet.SerializeTo(ctx, writer, size, func(writer *iokit.LimitedWriter) error {
		b := make([]byte, writer.RemainingBytes())

		if p.BodyRq != 0 {
			p.BodyRq.PutTo(b)
			b = b[ShortShipmentIDByteSize:]
		}

		if n := len(p.RejectList); n > 0 {
			sz := uint(protokit.EncodeVarintToBytes(b, uint64(n)))
			b = b[sz:]

			for _, id := range p.RejectList {
				if id == 0 {
					return throw.IllegalValue()
				}
				id.PutTo(b)
				b = b[ShortShipmentIDByteSize:]
			}
		}

		for _, id := range p.AckList {
			if id == 0 {
				return throw.IllegalValue()
			}
			id.PutTo(b)
			b = b[ShortShipmentIDByteSize:]
		}
		_, err := writer.Write(b)
		return err
	})
}

func (p *StatePacket) SerializePayload(sp *uniproto.SenderPacket, writer *iokit.LimitedWriter) error {
	b := make([]byte, writer.RemainingBytes())

	if p.BodyRq != 0 {
		p.BodyRq.PutTo(b)
		b = b[ShortShipmentIDByteSize:]
	}

	if n := len(p.RejectList); n > 0 {
		sz := uint(protokit.EncodeVarintToBytes(b, uint64(n)))
		b = b[sz:]

		for _, id := range p.RejectList {
			if id == 0 {
				return throw.IllegalValue()
			}
			id.PutTo(b)
			b = b[ShortShipmentIDByteSize:]
		}
	}

	for _, id := range p.AckList {
		if id == 0 {
			return throw.IllegalValue()
		}
		id.PutTo(b)
		b = b[ShortShipmentIDByteSize:]
	}
	_, err := writer.Write(b)
	return err
}

func (p *StatePacket) DeserializePayload(packet *uniproto.Packet, reader *iokit.LimitedReader) error {
	if reader.RemainingBytes() < ShortShipmentIDByteSize {
		return throw.IllegalValue()
	}

	b := make([]byte, reader.RemainingBytes())
	if _, err := reader.Read(b); err != nil {
		return err
	}

	if packet.Header.HasFlag(BodyRqFlag) {
		p.BodyRq = ShortShipmentIDReadFromBytes(b)
		if p.BodyRq == 0 {
			return throw.IllegalValue()
		}
		b = b[ShortShipmentIDByteSize:]
	}

	if packet.Header.HasFlag(RejectListFlag) {
		count, n := protokit.DecodeVarintFromBytes(b)
		if count == 0 || n == 0 {
			return throw.IllegalValue()
		}
		b = b[n:]
		p.RejectList = make([]ShortShipmentID, 0, count)
		for ; count > 0; count-- {
			id := ShortShipmentIDReadFromBytes(b)
			b = b[ShortShipmentIDByteSize:]
			if id == 0 {
				return throw.IllegalValue()
			}
			p.RejectList = append(p.RejectList, id)
		}
	}

	if len(b)%ShortShipmentIDByteSize != 0 {
		return throw.IllegalValue()
	}

	p.AckList = make([]ShortShipmentID, 0, len(b)/ShortShipmentIDByteSize)

	for len(b) > 0 {
		id := ShortShipmentIDReadFromBytes(b)
		if id == 0 {
			return throw.IllegalValue()
		}
		b = b[ShortShipmentIDByteSize:]
		p.AckList = append(p.AckList, id)
	}

	return nil
}
