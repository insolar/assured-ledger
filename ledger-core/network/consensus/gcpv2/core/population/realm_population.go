package population

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/phases"
)

type RealmPopulation interface {
	GetIndexedCount() int
	GetJoinersCount() int
	// GetVotersCount() int

	GetSealedCapacity() (int, bool)
	SealIndexed(indexedCapacity int) bool

	GetNodeAppearance(id node.ShortNodeID) *NodeAppearance
	GetActiveNodeAppearance(id node.ShortNodeID) *NodeAppearance
	GetJoinerNodeAppearance(id node.ShortNodeID) *NodeAppearance
	GetNodeAppearanceByIndex(idx int) *NodeAppearance

	GetShuffledOtherNodes() []*NodeAppearance /* excludes joiners and self */
	GetIndexedNodes() []*NodeAppearance       /* no joiners included */
	GetIndexedNodesAndHasNil() ([]*NodeAppearance, bool)
	GetIndexedCountAndCompleteness() (int, bool)

	GetSelf() *NodeAppearance

	// CreateNodeAppearance(ctx context.Context, inp profiles.ActiveNode) *NodeAppearance
	AddReservation(id node.ShortNodeID) (bool, *NodeAppearance)
	FindReservation(id node.ShortNodeID) (bool, *NodeAppearance)

	AddToDynamics(ctx context.Context, n *NodeAppearance) (*NodeAppearance, error)
	GetAnyNodes(includeIndexed bool, shuffle bool) []*NodeAppearance

	CreateVectorHelper() *RealmVectorHelper
	CreatePacketLimiter(isJoiner bool) phases.PacketLimiter

	GetTrustCounts() (fraudCount, bySelfCount, bySomeCount, byNeighborsCount uint16)
	GetDynamicCounts() (briefCount, fullCount uint16)
	GetPurgatoryCounts() (addedCount, ascentCount uint16)
}
