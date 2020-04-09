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

const ProtocolMessageDelivery = apinetwork.ProtocolTypeMessageDelivery

var MessageDeliveryProtocolDescriptor = apinetwork.ProtocolDescriptor{
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
	pt.Header.SetProtocolType(ProtocolMessageDelivery)
	pt.Header.SetPacketType(uint8(tp))
	return pt
}

// Flags for DeliveryState
const (
	BodyRqFlag apinetwork.FlagIndex = iota
	RejectListFlag
)

type DeliveryStatePacket struct {
	BodyRq     ParcelId   `insolar-transport:"optional=PacketFlags[0]"` //
	RejectList []ParcelId `insolar-transport:"optional=PacketFlags[1]"` //
	AckList    []ParcelId `insolar-transport:"list=nocount"`            // length is determined by packet size
}

func FillDeliveryStatePacket(maxPacketSize int, bodyRq ParcelId, ackList *[]ParcelId, rejectList *[]ParcelId) (m DeliveryStatePacket) {
	if bodyRq != 0 {
		m.BodyRq = bodyRq
		maxPacketSize -= ParcelIdByteSize
	}
	maxIdCount := maxPacketSize / ParcelIdByteSize
	maxIdCount, m.AckList = moveParcelIdList(maxIdCount, ackList)
	if maxIdCount > 1 { // 1 is to reserve space for varint count
		_, m.RejectList = moveParcelIdList(maxIdCount-1, rejectList)
	}
	return
}

func moveParcelIdList(maxCount int, list *[]ParcelId) (int, []ParcelId) {
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

func (p *DeliveryStatePacket) SerializeTo(ctx apinetwork.SerializationContext, writer io.Writer) error {
	packet := NewPacket(DeliveryState)

	size := uint(0)

	if p.BodyRq != 0 {
		packet.Header.SetFlag(BodyRqFlag, true)
		size += ParcelIdByteSize
	}
	if n := len(p.RejectList); n > 0 {
		packet.Header.SetFlag(RejectListFlag, true)
		size += uint(protokit.SizeVarint64(uint64(n)))
	}
	size += uint(len(p.AckList)) * ParcelIdByteSize
	if size == 0 {
		return throw.IllegalState()
	}

	return packet.SerializeTo(ctx, writer, size, func(writer *iokit.LimitedWriter) error {
		b := make([]byte, writer.RemainingBytes())

		if p.BodyRq != 0 {
			p.BodyRq.PutTo(b)
			b = b[ParcelIdByteSize:]
		}

		if n := len(p.RejectList); n > 0 {
			sz := uint(protokit.EncodeVarintToBytes(b, uint64(n)))
			b = b[sz:]

			for _, id := range p.RejectList {
				if id == 0 {
					return throw.IllegalValue()
				}
				id.PutTo(b)
				b = b[ParcelIdByteSize:]
			}
		}

		for _, id := range p.AckList {
			if id == 0 {
				return throw.IllegalValue()
			}
			id.PutTo(b)
			b = b[ParcelIdByteSize:]
		}
		_, err := writer.Write(b)
		return err
	})
}

func (p *DeliveryStatePacket) DeserializePayload(packet *apinetwork.Packet, reader *iokit.LimitedReader) error {
	if reader.RemainingBytes() < ParcelIdByteSize {
		return throw.IllegalValue()
	}

	b := make([]byte, reader.RemainingBytes())
	if _, err := reader.Read(b); err != nil {
		return err
	}

	if packet.Header.HasFlag(BodyRqFlag) {
		p.BodyRq = ParcelIdReadFromBytes(b)
		if p.BodyRq == 0 {
			return throw.IllegalValue()
		}
		b = b[ParcelIdByteSize:]
	}

	if packet.Header.HasFlag(RejectListFlag) {
		count, n := protokit.DecodeVarintFromBytes(b)
		if count == 0 || n == 0 {
			return throw.IllegalValue()
		}
		b = b[n:]
		p.RejectList = make([]ParcelId, 0, count)
		for ; count > 0; count-- {
			id := ParcelIdReadFromBytes(b)
			b = b[ParcelIdByteSize:]
			if id == 0 {
				return throw.IllegalValue()
			}
			p.RejectList = append(p.RejectList, id)
		}
	}

	if len(b)%ParcelIdByteSize != 0 {
		return throw.IllegalValue()
	}

	p.AckList = make([]ParcelId, 0, len(b)/ParcelIdByteSize)

	for len(b) > 0 {
		id := ParcelIdReadFromBytes(b)
		if id == 0 {
			return throw.IllegalValue()
		}
		b = b[ParcelIdByteSize:]
		p.AckList = append(p.AckList, id)
	}

	return nil
}

type DeliveryParcelPacket struct {
	ParcelId ParcelId
	ReturnId ParcelId `insolar-transport:"Packet=1;optional=PacketFlags[0]"`

	RepeatedSend bool                           `insolar-transport:"aliasOf=PacketFlags[1]"`
	ParcelType   apinetwork.PayloadCompleteness `insolar-transport:"send=ignore;aliasOf=PacketFlags[2]"`

	Data apinetwork.Serializable
}

const ( // Flags for DeliveryParcelPacket
	ReturnIdFlag apinetwork.FlagIndex = iota
	RepeatedSendFlag
	WithHeadFlag // for DeliveryParcelBody only
)

func (p *DeliveryParcelPacket) SerializeTo(ctx apinetwork.SerializationContext, writer io.Writer) error {
	if p.ParcelId == 0 {
		return throw.IllegalState()
	}

	packet := NewPacket(DeliveryParcelBody)

	switch p.ParcelType {
	case apinetwork.CompletePayload:
		packet.Header.SetFlag(WithHeadFlag, true)
	case apinetwork.BodyPayload:
	case apinetwork.HeadPayload:
		packet.Header.SetPacketType(uint8(DeliveryParcelHead))
	default:
		return throw.IllegalState()
	}

	size := uint(ParcelIdByteSize)
	if p.ReturnId != 0 {
		size <<= 1
		packet.Header.SetFlag(ReturnIdFlag, true)
	}
	size += p.Data.ByteSize()

	packet.Header.SetFlag(RepeatedSendFlag, p.RepeatedSend)

	return packet.SerializeTo(ctx, writer, size, func(writer *iokit.LimitedWriter) error {
		if err := p.ParcelId.WriteTo(writer); err != nil {
			return err
		}
		if p.ReturnId != 0 {
			if err := p.ReturnId.WriteTo(writer); err != nil {
				return err
			}
		}
		return p.Data.SerializeTo(ctx, writer)
	})
}

func (p *DeliveryParcelPacket) DeserializePayload(f apinetwork.DeserializationFactory, packet *apinetwork.Packet, reader *iokit.LimitedReader) error {
	switch PacketType(packet.Header.GetPacketType()) {
	case DeliveryParcelHead:
		p.ParcelType = apinetwork.HeadPayload
	case DeliveryParcelBody:
		if packet.Header.HasFlag(WithHeadFlag) {
			p.ParcelType = apinetwork.CompletePayload
		} else {
			p.ParcelType = apinetwork.BodyPayload
		}
	}

	{
		hasReturn := packet.Header.HasFlag(ReturnIdFlag)
		var b []byte
		if hasReturn {
			b = make([]byte, ParcelIdByteSize<<1)
		} else {
			b = make([]byte, ParcelIdByteSize)
		}
		if _, err := io.ReadFull(reader, b); err != nil {
			return err
		}
		p.ParcelId = ParcelIdReadFromBytes(b)
		if hasReturn {
			p.ReturnId = ParcelIdReadFromBytes(b[ParcelIdByteSize:])
		}
	}

	p.RepeatedSend = packet.Header.HasFlag(RepeatedSendFlag)

	if p.Data != nil {
		return p.Data.DeserializeFrom(f, reader)
	}

	if d, err := f.DeserializePayloadFrom(p.ParcelType, reader); err != nil {
		return err
	} else {
		p.Data = d
	}
	return nil
}
