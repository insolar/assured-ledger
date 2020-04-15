// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const Protocol = apinetwork.ProtocolTypeMessageDelivery

var ProtocolDescriptor = apinetwork.ProtocolDescriptor{
	SupportedPackets: [16]apinetwork.ProtocolPacketDescriptor{
		DeliveryState:      {Flags: apinetwork.DatagramAllowed, LengthBits: apinetwork.SmallLengthBits},
		DeliveryParcelHead: {Flags: apinetwork.DatagramAllowed, LengthBits: apinetwork.SmallLengthBits},
		DeliveryParcelBody: {Flags: apinetwork.DatagramAllowed, LengthBits: apinetwork.MaxLengthBits},
	},
}

type PacketType uint8

const (
	DeliveryState PacketType = iota
	DeliveryParcelHead
	DeliveryParcelBody // or Head + Body
)

func NewPacket(tp PacketType) apinetwork.Packet {
	pt := apinetwork.Packet{}
	pt.Header.SetProtocolType(Protocol)
	pt.Header.SetPacketType(uint8(tp))
	return pt
}

// Flags for DeliveryState
const (
	BodyRqFlag apinetwork.FlagIndex = iota
	RejectListFlag
)

type StatePacket struct {
	BodyRq     ShipmentID   `insolar-transport:"optional=PacketFlags[0]"` //
	RejectList []ShipmentID `insolar-transport:"optional=PacketFlags[1]"` //
	AckList    []ShipmentID `insolar-transport:"list=nocount"`            // length is determined by packet size
}

func FillDeliveryStatePacket(maxPacketSize int, bodyRq ShipmentID, ackList *[]ShipmentID, rejectList *[]ShipmentID) (m StatePacket) {
	if bodyRq != 0 {
		m.BodyRq = bodyRq
		maxPacketSize -= ShipmentIdByteSize
	}
	maxIdCount := maxPacketSize / ShipmentIdByteSize
	maxIdCount, m.AckList = moveParcelIdList(maxIdCount, ackList)
	if maxIdCount > 1 { // 1 is to reserve space for varint count
		_, m.RejectList = moveParcelIdList(maxIdCount-1, rejectList)
	}
	return
}

func moveParcelIdList(maxCount int, list *[]ShipmentID) (int, []ShipmentID) {
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

func (p *StatePacket) SerializeTo(ctx apinetwork.SerializationContext, writer io.Writer) error {
	packet := NewPacket(DeliveryState)

	size := uint(0)

	if p.BodyRq != 0 {
		packet.Header.SetFlag(BodyRqFlag, true)
		size += ShipmentIdByteSize
	}
	if n := len(p.RejectList); n > 0 {
		packet.Header.SetFlag(RejectListFlag, true)
		size += uint(protokit.SizeVarint64(uint64(n)))
	}
	size += uint(len(p.AckList)) * ShipmentIdByteSize
	if size == 0 {
		return throw.IllegalState()
	}

	return packet.SerializeTo(ctx, writer, size, func(writer *iokit.LimitedWriter) error {
		b := make([]byte, writer.RemainingBytes())

		if p.BodyRq != 0 {
			p.BodyRq.PutTo(b)
			b = b[ShipmentIdByteSize:]
		}

		if n := len(p.RejectList); n > 0 {
			sz := uint(protokit.EncodeVarintToBytes(b, uint64(n)))
			b = b[sz:]

			for _, id := range p.RejectList {
				if id == 0 {
					return throw.IllegalValue()
				}
				id.PutTo(b)
				b = b[ShipmentIdByteSize:]
			}
		}

		for _, id := range p.AckList {
			if id == 0 {
				return throw.IllegalValue()
			}
			id.PutTo(b)
			b = b[ShipmentIdByteSize:]
		}
		_, err := writer.Write(b)
		return err
	})
}

func (p *StatePacket) SerializePayload(sp *apinetwork.SenderPacket, writer *iokit.LimitedWriter) error {
	b := make([]byte, writer.RemainingBytes())

	if p.BodyRq != 0 {
		p.BodyRq.PutTo(b)
		b = b[ShipmentIdByteSize:]
	}

	if n := len(p.RejectList); n > 0 {
		sz := uint(protokit.EncodeVarintToBytes(b, uint64(n)))
		b = b[sz:]

		for _, id := range p.RejectList {
			if id == 0 {
				return throw.IllegalValue()
			}
			id.PutTo(b)
			b = b[ShipmentIdByteSize:]
		}
	}

	for _, id := range p.AckList {
		if id == 0 {
			return throw.IllegalValue()
		}
		id.PutTo(b)
		b = b[ShipmentIdByteSize:]
	}
	_, err := writer.Write(b)
	return err
}

func (p *StatePacket) DeserializePayload(packet *apinetwork.Packet, reader *iokit.LimitedReader) error {
	if reader.RemainingBytes() < ShipmentIdByteSize {
		return throw.IllegalValue()
	}

	b := make([]byte, reader.RemainingBytes())
	if _, err := reader.Read(b); err != nil {
		return err
	}

	if packet.Header.HasFlag(BodyRqFlag) {
		p.BodyRq = ShipmentIdReadFromBytes(b)
		if p.BodyRq == 0 {
			return throw.IllegalValue()
		}
		b = b[ShipmentIdByteSize:]
	}

	if packet.Header.HasFlag(RejectListFlag) {
		count, n := protokit.DecodeVarintFromBytes(b)
		if count == 0 || n == 0 {
			return throw.IllegalValue()
		}
		b = b[n:]
		p.RejectList = make([]ShipmentID, 0, count)
		for ; count > 0; count-- {
			id := ShipmentIdReadFromBytes(b)
			b = b[ShipmentIdByteSize:]
			if id == 0 {
				return throw.IllegalValue()
			}
			p.RejectList = append(p.RejectList, id)
		}
	}

	if len(b)%ShipmentIdByteSize != 0 {
		return throw.IllegalValue()
	}

	p.AckList = make([]ShipmentID, 0, len(b)/ShipmentIdByteSize)

	for len(b) > 0 {
		id := ShipmentIdReadFromBytes(b)
		if id == 0 {
			return throw.IllegalValue()
		}
		b = b[ShipmentIdByteSize:]
		p.AckList = append(p.AckList, id)
	}

	return nil
}
