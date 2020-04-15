// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
)

type RegisterProtocolFunc func(ProtocolType, Descriptor, ProtocolController)
type ProtocolRegistrationFunc func(RegisterProtocolFunc)
type ProtocolController interface {
	Start(ProtocolSender)
	Stop()
}

type PayloadSerializeFunc func(*SenderPacket, *iokit.LimitedWriter) error
type PacketSerializeFunc func(*Packet) (int, PayloadSerializeFunc)

type ProtocolSender interface {
	SendTo(apinetwork.Address, func(*Packet) PayloadSerializeFunc) error
	ConnectTo(apinetwork.Address, func(*SenderPacket)) error
}
