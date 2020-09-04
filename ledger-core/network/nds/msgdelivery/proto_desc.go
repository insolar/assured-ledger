// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import "github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"

const Protocol = uniproto.ProtocolTypeMessageDelivery

var protoDescriptor = uniproto.Descriptor{
	SupportedPackets: [16]uniproto.PacketDescriptor{
		DeliveryState:          {Flags: 0 /* uniproto.DatagramAllowed */, LengthBits: uniproto.SmallLengthBits},
		DeliveryParcelHead:     {Flags: 0 /* uniproto.DatagramAllowed */, LengthBits: uniproto.SmallLengthBits},
		DeliveryParcelComplete: {Flags: 0 /* uniproto.DatagramAllowed */, LengthBits: uniproto.MaxLengthBits},
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
