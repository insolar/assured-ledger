package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
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
		if err = readFn(packet.GetContext(p.ctl.factory), payload.DeserializePayload); err == nil {
			err = p.ctl.receiveState(packet, payload)
		}
	case DeliveryParcelHead:
		payload := &ParcelPacket{}
		if err = readFn(packet.GetContext(p.ctl.factory), payload.DeserializePayload); err == nil {
			err = p.ctl.receiveParcel(packet, payload)
		}
	case DeliveryParcelComplete:
		payload := &ParcelPacket{}
		payload.OnDataReceiveFn = p.ctl.onReceiveLargeParcelData
		if err = readFn(packet.GetContext(p.ctl.factory), payload.DeserializePayload); err == nil {
			err = p.ctl.receiveParcel(packet, payload)
		}
	default:
		err = throw.Impossible()
	}
	if err == nil {
		return nil
	}
	p.ctl.reportError(throw.WithDetails(err, uniserver.PacketErrDetails{Header: packet.Header, Pulse: packet.PulseNumber}))
	return err
}
