package serialization

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/phases"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

var (
	defaultByteOrder = binary.BigEndian
)

type PacketHeaderAccessor interface {
	GetProtocolType() ProtocolType
	GetPacketType() phases.PacketType
	GetSourceID() node.ShortNodeID
	HasFlag(flag Flag) bool
	GetFlagRangeInt(from, to uint8) uint8
	IsRelayRestricted() bool
	IsBodyEncrypted() bool
}

type PacketHeaderModifier interface {
	SetFlag(flag Flag)
	ClearFlag(flag Flag)
}

type PacketBody interface {
	ContextSerializerTo
	ContextDeserializerFrom

	String(ctx PacketContext) string
}

type FieldContext uint

const (
	NoContext = FieldContext(iota)
	ContextMembershipAnnouncement
	ContextNeighbourAnnouncement
)

type PacketContext interface {
	PacketHeaderAccessor
	context.Context

	InContext(ctx FieldContext) bool
	SetInContext(ctx FieldContext)
	GetNeighbourNodeID() node.ShortNodeID
	SetNeighbourNodeID(nodeID node.ShortNodeID)
	GetAnnouncedJoinerNodeID() node.ShortNodeID
	SetAnnouncedJoinerNodeID(nodeID node.ShortNodeID)
}

type SerializeContext interface {
	PacketHeaderModifier
	PacketContext
}

type DeserializeContext interface {
	PacketContext
}

type SerializerTo interface {
	SerializeTo(ctx context.Context, writer io.Writer, digester cryptkit.DataDigester, signer cryptkit.DigestSigner) (int64, error)
}

type ContextSerializerTo interface {
	SerializeTo(ctx SerializeContext, writer io.Writer) error
}

type DeserializerFrom interface {
	DeserializeFrom(ctx context.Context, reader io.Reader) (int64, error)
}

type ContextDeserializerFrom interface {
	DeserializeFrom(ctx DeserializeContext, reader io.Reader) error
}
