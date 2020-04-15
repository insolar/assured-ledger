// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l2"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ uniproto.Receiver = &packetReceiver{}

type packetReceiver struct {
	ctl *Controller
}

func (p *packetReceiver) ReceiveSmallPacket(packet *uniproto.ReceivedPacket, b []byte) {
	readFn := packet.NewSmallPayloadDeserializer(b)
	_ = p.receiveDispatcher(packet, readFn)
}

func (p *packetReceiver) ReceiveLargePacket(packet *uniproto.ReceivedPacket, preRead []byte, r io.LimitedReader) error {
	readFn := packet.NewLargePayloadDeserializer(preRead, r)
	err := p.receiveDispatcher(packet, readFn)
	return packet.DownstreamError(err)
}

func (p *packetReceiver) receiveDispatcher(packet *uniproto.ReceivedPacket, readFn uniproto.PacketDeserializerFunc) error {
	var err error
	switch pt := PacketType(packet.Header.GetPacketType()); pt {
	case DeliveryState:
		payload := &StatePacket{}
		if err = readFn(payload.DeserializePayload); err == nil {
			err = p.ctl.receiveState(packet, payload)
		}
	case DeliveryParcelHead, DeliveryParcelBody:
		payload := &ParcelPacket{}
		if err = readFn(func(pkt *uniproto.Packet, r *iokit.LimitedReader) error {
			return payload.DeserializePayload(p.ctl.factory, pkt, r)
		}); err == nil {
			err = p.ctl.receiveParcel(packet, payload)
		}
	default:
		err = throw.Impossible()
	}
	p.ctl.reportError(throw.WithDetails(err, l2.PacketErrDetails{packet.Header, packet.PulseNumber}))
	return err
}
