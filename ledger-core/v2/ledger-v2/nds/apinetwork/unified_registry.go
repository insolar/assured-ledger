// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
)

type RegisterProtocolFunc func(ProtocolType, ProtocolDescriptor, ProtocolController)
type ProtocolRegistrationFunc func(RegisterProtocolFunc)
type ProtocolController interface {
	Start(ProtocolSender)
	Stop()
}

type PayloadSerializeFunc func(*SenderPacket, *iokit.LimitedWriter) error
type PacketSerializeFunc func(*Packet) (int, PayloadSerializeFunc)

type ProtocolSender interface {
	SendTo(Address, func(*Packet) PayloadSerializeFunc) error
	ConnectTo(Address, func(*SenderPacket)) error
}

type SenderPacket struct {
	Packet
	signer    PacketDataSigner
	encrypter cryptkit.Encrypter
}

type PacketDataSigner struct {
	signer cryptkit.DataSigner
}
