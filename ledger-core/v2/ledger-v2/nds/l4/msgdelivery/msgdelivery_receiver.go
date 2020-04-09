// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l2"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type ProtocolController struct {
	factory  apinetwork.DeserializationFactory
	receiver PacketReceiver
}

func (p *ProtocolController) reportError(err error) {

}

func (p *ProtocolController) receiveState(payload *DeliveryStatePacket) error {

}

func (p *ProtocolController) receiveComplete(payload *DeliveryParcelPacket) error {

}

func (p *ProtocolController) receiveBody(payload *DeliveryParcelPacket) error {

}

func (p *ProtocolController) receiveHead(payload *DeliveryParcelPacket) error {

}

var _ apinetwork.ProtocolReceiver = &PacketReceiver{}

type PacketReceiver struct {
	ctl *ProtocolController
}

func (p *PacketReceiver) ReceiveSmallPacket(packet *apinetwork.ReceiverPacket, b []byte) {
	readFn := packet.NewSmallPayloadDeserializer(b)
	_ = p.receiveDispatcher(packet, readFn)
}

func (p *PacketReceiver) ReceiveLargePacket(packet *apinetwork.ReceiverPacket, preRead []byte, r io.LimitedReader) error {
	readFn := packet.NewLargePayloadDeserializer(preRead, r)
	err := p.receiveDispatcher(packet, readFn)
	return packet.DownstreamError(err)
}

func (p *PacketReceiver) receiveDispatcher(packet *apinetwork.ReceiverPacket, readFn apinetwork.PacketDeserializerFunc) error {
	var err error
	switch pt := PacketType(packet.Header.GetPacketType()); pt {
	case DeliveryState:
		payload := &DeliveryStatePacket{}
		if err = readFn(payload.DeserializePayload); err == nil {
			err = p.ctl.receiveState(payload)
		}
	case DeliveryParcelHead, DeliveryParcelBody:
		payload := &DeliveryParcelPacket{}
		if err = readFn(func(pkt *apinetwork.Packet, r *iokit.LimitedReader) error {
			return payload.DeserializePayload(p.ctl.factory, pkt, r)
		}); err == nil {
			switch payload.ParcelType {
			case apinetwork.CompletePayload:
				err = p.ctl.receiveComplete(payload)
			case apinetwork.BodyPayload:
				err = p.ctl.receiveBody(payload)
			case apinetwork.HeadPayload:
				err = p.ctl.receiveHead(payload)
			}
		}
	default:
		err = throw.Impossible()
	}
	p.ctl.reportError(throw.WithDetails(err, l2.PacketErrDetails{packet.Header, packet.PulseNumber}))
	return err
}
