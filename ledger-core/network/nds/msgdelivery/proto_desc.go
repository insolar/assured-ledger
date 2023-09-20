package msgdelivery

import "github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"

const Protocol = uniproto.ProtocolTypeMessageDelivery

var protoDescriptor = uniproto.Descriptor{
	SupportedPackets: [16]uniproto.PacketDescriptor{
		DeliveryState:          {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.SmallLengthBits},
		DeliveryParcelHead:     {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.SmallLengthBits},
		DeliveryParcelComplete: {Flags: uniproto.DatagramAllowed, LengthBits: uniproto.MaxLengthBits},
	},
}

const maxStatePacketDataSize = uniproto.MaxNonExcessiveLength - uniproto.PacketByteSizeMin - 512 /* reserved for signature */

type PacketType uint8

const (
	DeliveryState PacketType = iota
	DeliveryParcelHead
	DeliveryParcelComplete
)

func initPacket(tp PacketType, pt *uniproto.Packet) {
	pt.Header.SetProtocolType(Protocol)
	pt.Header.SetPacketType(uint8(tp))
}
