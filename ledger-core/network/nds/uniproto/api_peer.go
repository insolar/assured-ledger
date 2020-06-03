// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type Peer interface {
	GetPrimary() nwapi.Address
	GetLocalUID() nwapi.Address
	GetNodeID() nwapi.ShortNodeID

	GetSignatureKey() cryptkit.SignatureKey

	SetProtoInfo(pt ProtocolType, info io.Closer)
	GetProtoInfo(pt ProtocolType) io.Closer
	GetOrCreateProtoInfo(pt ProtocolType, factoryFn func(Peer) io.Closer) io.Closer

	Transport() OutTransport
	SendPacket(tp OutType, packet PacketPreparer) error
	SendPreparedPacket(tp OutType, packet *Packet, dataSize uint, fn PayloadSerializerFunc, checkFn func() bool) error
}

type OutType uint8

const (
	Any OutType = iota
	SessionfulAny
	SessionfulLarge
	SessionfulSmall
	SmallAny
	Sessionless
	SessionlessNoQuota
)
