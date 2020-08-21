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
	// GetPrimary returns a peer's primary identity
	GetPrimary() nwapi.Address
	// GetLocalUID returns a locally unique peer's address.
	// This address can only be used locally and it will change after each disconnection of the peer.
	GetLocalUID() nwapi.Address
	// GetNodeID returns ShortNodeID of the peer. Result can be zero. ShortNodeID is related to consensus operations.
	GetNodeID() nwapi.ShortNodeID

	// GetSignatureKey returns peer's signature key. Key can be zero.
	GetSignatureKey() cryptkit.SignatureKey

	// SetProtoInfo sets/updated per-protocol peer's projection (info). It will be closed when the peer is unregistered.
	SetProtoInfo(pt ProtocolType, info io.Closer)
	// GetOrCreateProtoInfo performs "atomic" get or create of peer's projection for the given protocol.
	GetProtoInfo(pt ProtocolType) io.Closer
	// GetOrCreateProtoInfo performs "atomic" get or create of peer's projection for the given protocol.
	GetOrCreateProtoInfo(pt ProtocolType, factoryFn func(Peer) io.Closer) io.Closer

	// Transport returns a multiplexed transport over available peer's transports.
	// This transport does internal connection management (retry, reconnect, etc).
	Transport() OutTransport
	// SendPacket is a convenience handler to send a packet provided by uniproto.PacketPreparer.
	// See SendPreparedPacket for details.
	SendPacket(tp OutType, packet PacketPreparer) error
	// SendPreparedPacket prepares uniproto packet and checks its eligibility for the given (uniproto.OutType).
	// Can also choose a specific out type by properties of packet and value of (uniproto.OutType).
	// See SendingPacket.NewTransportFunc for details about serialization.
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
