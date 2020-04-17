// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
)

type PacketTemplate struct {
	Packet
}

type DataSerializerFunc func(*SendingPacket, *iokit.LimitedWriter) error
type PacketSerializerFunc func() (template PacketTemplate, dataSize uint, dataFn DataSerializerFunc)

//var _ PacketSerializerFunc = PacketSerializer(nil).SerializePacket

type PacketSerializer interface {
	SerializePacket() (template PacketTemplate, dataSize uint, dataFn DataSerializerFunc)
}

type SerializationHelper struct {
	// TODO signature size
	// TODO size limit
}

func NewSendingPacket(signer cryptkit.DataSigner, encrypter cryptkit.Encrypter) *SendingPacket {
	p := &SendingPacket{}
	p.signer.signer = signer
	p.encrypter = encrypter
	return p
}

type SendingPacket struct {
	Packet
	Peer            Peer
	signer          PacketDataSigner
	encrypter       cryptkit.Encrypter
	packetSizeLimit uint64
}

func (p SendingPacket) GetContext() nwapi.SerializationContext {
	return nil
}

type PacketDataSigner struct {
	signer cryptkit.DataSigner
}
