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
	GetSignatureKey() cryptkit.SigningKey

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

// OutType defines a type of transport allowed for a send operation.
type OutType uint8

const (
	// Any enables use of any transport based on packet size. Sessionless will be preferred over SessionfulSmall.
	Any OutType = iota
	// SessionfulAny allows use of either SessionfulSmall or SessionfulLarge based on packet size.
	SessionfulAny
	// SessionfulLarge allows ONLY use of sessionful large-packet transport, irrelevant of packet size.
	SessionfulLarge
	// SessionfulLarge allows ONLY use of sessionful small-packet transport, irrelevant of packet size.
	// Packets larger than uniproto.MaxNonExcessiveLength will be rejected locally with an error.
	SessionfulSmall
	// SmallAny allows use of either Sessionless or SessionfulSmall. Sessionless is preferred when size allows.
	// Packets larger than uniproto.MaxNonExcessiveLength will be rejected locally with an error.
	SmallAny
	// Sessionless allows ONLY use of sessionless transport.
	// Packets larger than transport's size limit will be rejected locally with an error.
	Sessionless
	// SessionlessNoQuota allows ONLY use of sessionless transport. Quota limits will be ignored.
	// Packets larger than transport's size limit will be rejected locally with an error.
	// NB! Only for use by consensus / low latency packets.
	SessionlessNoQuota
)
