// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ uniproto.ProtocolPacket = &ParcelPacket{}

type ParcelPacket struct {
	ParcelId     ShortShipmentID
	ReturnId     ShortShipmentID           `insolar-transport:"Packet=1;optional=PacketFlags[0]"`
	RepeatedSend bool                      `insolar-transport:"aliasOf=PacketFlags[1]"`
	ParcelType   nwapi.PayloadCompleteness `insolar-transport:"send=ignore;aliasOf=PacketFlags[2]"`

	Data nwapi.Serializable
}

const ( // Flags for ParcelPacket
	ReturnIdFlag uniproto.FlagIndex = iota
	RepeatedSendFlag
)

func (p *ParcelPacket) PreparePacket() (packet uniproto.PacketTemplate, dataSize uint, fn uniproto.PayloadSerializerFunc) {
	if p.ParcelId == 0 {
		panic(throw.IllegalState())
	}

	initPacket(DeliveryParcelComplete, &packet.Packet)

	switch p.ParcelType {
	case nwapi.CompletePayload:
		//
	case nwapi.HeadOnlyPayload:
		packet.Header.SetPacketType(uint8(DeliveryParcelHead))
	default:
		panic(throw.IllegalState())
	}

	dataSize = uint(ShortShipmentIDByteSize)
	if p.ReturnId != 0 {
		dataSize <<= 1
		packet.Header.SetFlag(ReturnIdFlag, true)
	}
	dataSize += p.Data.ByteSize()

	packet.Header.SetFlag(RepeatedSendFlag, p.RepeatedSend)

	return packet, dataSize, p.SerializePayload
}

func (p *ParcelPacket) SerializePayload(ctx nwapi.SerializationContext, _ *uniproto.Packet, writer *iokit.LimitedWriter) error {
	if err := p.ParcelId.SimpleWriteTo(writer); err != nil {
		return err
	}
	if p.ReturnId != 0 {
		if err := p.ReturnId.SimpleWriteTo(writer); err != nil {
			return err
		}
	}
	return p.Data.SerializeTo(ctx, writer)
}

func (p *ParcelPacket) DeserializePayload(ctx nwapi.DeserializationContext, packet *uniproto.Packet, reader *iokit.LimitedReader) error {
	switch PacketType(packet.Header.GetPacketType()) {
	case DeliveryParcelHead:
		p.ParcelType = nwapi.HeadOnlyPayload
	case DeliveryParcelComplete:
		p.ParcelType = nwapi.CompletePayload
	default:
		panic(throw.Impossible())
	}

	{
		hasReturn := packet.Header.HasFlag(ReturnIdFlag)
		var b []byte
		if hasReturn {
			b = make([]byte, ShortShipmentIDByteSize<<1)
		} else {
			b = make([]byte, ShortShipmentIDByteSize)
		}
		if _, err := io.ReadFull(reader, b); err != nil {
			return err
		}
		p.ParcelId = ShortShipmentIDReadFromBytes(b)
		if hasReturn {
			p.ReturnId = ShortShipmentIDReadFromBytes(b[ShortShipmentIDByteSize:])
		}
	}

	p.RepeatedSend = packet.Header.HasFlag(RepeatedSendFlag)

	if p.Data != nil {
		return p.Data.DeserializeFrom(ctx, reader)
	}

	f := ctx.GetPayloadFactory()
	d, err := f.DeserializePayloadFrom(p.ParcelType, reader)
	if err != nil {
		return err
	}
	p.Data = d
	return nil
}
