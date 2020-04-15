// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type ParcelPacket struct {
	ParcelId     ShipmentID
	ReturnId     ShipmentID                     `insolar-transport:"Packet=1;optional=PacketFlags[0]"`
	RepeatedSend bool                           `insolar-transport:"aliasOf=PacketFlags[1]"`
	ParcelType   apinetwork.PayloadCompleteness `insolar-transport:"send=ignore;aliasOf=PacketFlags[2]"`

	Data apinetwork.Serializable
}

const ( // Flags for ParcelPacket
	ReturnIdFlag apinetwork.FlagIndex = iota
	RepeatedSendFlag
	WithHeadFlag // for DeliveryParcelBody only
)

func (p *ParcelPacket) SerializeTo(ctx apinetwork.SerializationContext, writer io.Writer) error {
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

	size := uint(ShipmentIdByteSize)
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

func (p *ParcelPacket) DeserializePayload(f apinetwork.DeserializationFactory, packet *apinetwork.Packet, reader *iokit.LimitedReader) error {
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
			b = make([]byte, ShipmentIdByteSize<<1)
		} else {
			b = make([]byte, ShipmentIdByteSize)
		}
		if _, err := io.ReadFull(reader, b); err != nil {
			return err
		}
		p.ParcelId = ShipmentIdReadFromBytes(b)
		if hasReturn {
			p.ReturnId = ShipmentIdReadFromBytes(b[ShipmentIdByteSize:])
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
