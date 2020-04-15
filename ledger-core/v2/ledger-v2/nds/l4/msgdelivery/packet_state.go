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

var ProtocolDescriptor = uniproto.Descriptor{
	SupportedPackets: [16]uniproto.PacketDescriptor{
		DeliveryState:      {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.SmallLengthBits},
		DeliveryParcelHead: {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.SmallLengthBits},
		DeliveryParcelBody: {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.MaxLengthBits},
	},
}

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
	RejectListFlag
	BodyAckListFlag
)

type StatePacket struct {
	BodyRq      ShortShipmentID   `insolar-transport:"optional=PacketFlags[0]"` //
	RejectList  []ShortShipmentID `insolar-transport:"optional=PacketFlags[1]"` //
	BodyAckList []ShortShipmentID `insolar-transport:"optional=PacketFlags[2]"` //
	AckList     []ShortShipmentID `insolar-transport:"list=nocount"`            // length is determined by packet size
}

func FillDeliveryStatePacket(maxPacketSize int, bodyRq ShortShipmentID, ackList *[]ShortShipmentID, rejectList *[]ShortShipmentID) (m StatePacket) {
	if bodyRq != 0 {
		m.BodyRq = bodyRq
		maxPacketSize -= ShortShipmentIDByteSize
	}
	maxIdCount := maxPacketSize / ShortShipmentIDByteSize
	maxIdCount, m.AckList = moveParcelIdList(maxIdCount, ackList)
	if maxIdCount > 1 { // 1 is to reserve space for varint count
		_, m.RejectList = moveParcelIdList(maxIdCount-1, rejectList)
	}
	return
}

func moveParcelIdList(maxCount int, list *[]ShortShipmentID) (int, []ShortShipmentID) {
	n := len(*list)
	if maxCount >= n {
		x := *list
		*list = nil
		return maxCount - n, x
	}
	x := *list
	// take from tail to allow better memory re-use on high load
	// also, recent packets will be acked/rejected faster that improves efficiency on high load
	*list = x[:n-maxCount]
	return maxCount, x[n-maxCount:]
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
