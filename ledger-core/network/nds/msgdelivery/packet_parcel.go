// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

var _ uniproto.ProtocolPacket = &ParcelPacket{}

type ParcelPacket struct {
	PulseNumber  pulse.Number `insolar-transport:"aliasOf=Packet.PulseNumber"`
	ParcelID     ShortShipmentID
	ReturnID     ShortShipmentID           `insolar-transport:"Packet=1,2;optional=PacketFlags[0]"`
	BodyScale    uint8                     `insolar-transport:"Packet=1"` // bits.Len(byteSize(body+head))
	TTLCycles    uint8                     `insolar-transport:"Packet=1"` // adjusted by PN
	ParcelType   nwapi.PayloadCompleteness `insolar-transport:"send=ignore"`
	RepeatedSend bool                      `insolar-transport:"aliasOf=PacketFlags[1]"`

	Data nwapi.SizeAwareSerializer // nwapi.Serializable

	OnDataReceiveFn func(*uniproto.Packet, *ParcelPacket, func() error) error `insolar-transport:"send=ignore"`
}

const ( // Flags for ParcelPacket
	ReturnIDFlag uniproto.FlagIndex = iota
	RepeatedSendFlag
	HasBody
)

func (p *ParcelPacket) PreparePacket() (packet uniproto.PacketTemplate, dataSize uint, fn uniproto.PayloadSerializerFunc) {
	if p.ParcelID == 0 {
		panic(throw.IllegalState())
	}

	initPacket(DeliveryParcelComplete, &packet.Packet)

	dataSize = uint(ShortShipmentIDByteSize)

	switch p.ParcelType {
	case nwapi.CompletePayload: // , nwapi.HeadOnlyPayload
		//
	case nwapi.PartialPayload:
		if p.BodyScale == 0 {
			panic(throw.IllegalState())
		}
		dataSize += 2
		packet.Header.SetPacketType(uint8(DeliveryParcelHead))
	default:
		panic(throw.IllegalState())
	}

	if p.ReturnID != 0 {
		dataSize += uint(ShortShipmentIDByteSize)
		packet.Header.SetFlag(ReturnIDFlag, true)
	}
	dataSize += p.Data.ByteSize()

	packet.Header.SetFlag(RepeatedSendFlag, p.RepeatedSend)
	packet.PulseNumber = p.PulseNumber

	return packet, dataSize, p.SerializePayload
}

func (p *ParcelPacket) SerializePayload(ctx nwapi.SerializationContext, packet *uniproto.Packet, writer *iokit.LimitedWriter) error {
	if p.ParcelID == 0 {
		panic(throw.IllegalValue())
	}

	if err := p.ParcelID.SimpleWriteTo(writer); err != nil {
		return err
	}
	if p.ReturnID != 0 {
		if err := p.ReturnID.SimpleWriteTo(writer); err != nil {
			return err
		}
	}
	if packet.Header.GetPacketType() == uint8(DeliveryParcelHead) {
		if p.BodyScale == 0 {
			panic(throw.IllegalValue())
		}
		if _, err := writer.Write([]byte{p.BodyScale, p.TTLCycles}); err != nil {
			return err
		}
	}
	return p.Data.SerializeTo(ctx, writer)
}

func (p *ParcelPacket) DeserializePayload(ctx nwapi.DeserializationContext, packet *uniproto.Packet, reader *iokit.LimitedReader) error {
	pt := PacketType(packet.Header.GetPacketType())
	switch pt {
	case DeliveryParcelHead:
		p.ParcelType = nwapi.PartialPayload
	case DeliveryParcelComplete:
		p.ParcelType = nwapi.CompletePayload
	default:
		panic(throw.Impossible())
	}
	p.PulseNumber = packet.PulseNumber

	{
		bufSize := ShortShipmentIDByteSize
		if pt == DeliveryParcelHead {
			bufSize += 2
		}
		hasReturn := packet.Header.HasFlag(ReturnIDFlag)
		if hasReturn {
			bufSize += ShortShipmentIDByteSize
		}
		b := make([]byte, bufSize)
		if _, err := io.ReadFull(reader, b); err != nil {
			return err
		}

		p.ParcelID = ShortShipmentIDReadFromBytes(b)
		if p.ParcelID == 0 {
			return throw.IllegalValue()
		}

		bufSize = ShortShipmentIDByteSize
		if hasReturn {
			p.ReturnID = ShortShipmentIDReadFromBytes(b[bufSize:])
			if p.ReturnID == 0 {
				return throw.IllegalValue()
			}
			bufSize += ShortShipmentIDByteSize
		}

		if pt == DeliveryParcelHead {
			p.BodyScale = b[bufSize]
			if p.BodyScale == 0 {
				return throw.IllegalValue()
			}
			p.TTLCycles = b[bufSize+1]
		}
	}

	p.RepeatedSend = packet.Header.HasFlag(RepeatedSendFlag)

	readDataFn := func() error {
		if des, ok := p.Data.(nwapi.SizeAwareDeserializer); ok {
			return des.DeserializeFrom(ctx, reader)
		}
		p.Data = nil

		f := ctx.GetPayloadFactory()
		d, err := f.DeserializePayloadFrom(ctx, p.ParcelType, reader)
		if err != nil {
			return err
		}
		p.Data = d
		return nil
	}

	if p.OnDataReceiveFn != nil {
		return p.OnDataReceiveFn(packet, p, readDataFn)
	}
	return readDataFn()
}
