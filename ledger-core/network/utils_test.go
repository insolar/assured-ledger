package network

import (
	"crypto"
	"math"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"
)

func newTestNodeWithShortID(id node.ShortNodeID) *mutable.Node {
	n := mutable.NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleUnknown, "127.0.0.1:5432")
	n.SetShortID(id)
	return n
}

func TestCorrectShortIDCollision(t *testing.T) {

	nodes := []nodeinfo.NetworkNode{
		newTestNodeWithShortID(0),
		newTestNodeWithShortID(1),
		newTestNodeWithShortID(30),
		newTestNodeWithShortID(32),
		newTestNodeWithShortID(33),
		newTestNodeWithShortID(34),
		newTestNodeWithShortID(64),
		newTestNodeWithShortID(1<<32 - 2),
		newTestNodeWithShortID(1<<32 - 1),
	}

	require.False(t, CheckShortIDCollision(nodes, node.ShortNodeID(2)))
	require.False(t, CheckShortIDCollision(nodes, node.ShortNodeID(31)))
	require.False(t, CheckShortIDCollision(nodes, node.ShortNodeID(35)))
	require.False(t, CheckShortIDCollision(nodes, node.ShortNodeID(65)))

	require.True(t, CheckShortIDCollision(nodes, node.ShortNodeID(30)))
	require.Equal(t, node.ShortNodeID(31), regenerateShortID(nodes, node.ShortNodeID(30)))

	require.True(t, CheckShortIDCollision(nodes, node.ShortNodeID(32)))
	require.Equal(t, node.ShortNodeID(35), regenerateShortID(nodes, node.ShortNodeID(32)))

	require.True(t, CheckShortIDCollision(nodes, node.ShortNodeID(64)))
	require.Equal(t, node.ShortNodeID(65), regenerateShortID(nodes, node.ShortNodeID(64)))

	require.True(t, CheckShortIDCollision(nodes, node.ShortNodeID(1<<32-2)))
	require.Equal(t, node.ShortNodeID(2), regenerateShortID(nodes, node.ShortNodeID(1<<32-2)))
}

var _ nodeinfo.DiscoveryNode = testNode{}

type testNode struct {
	ref reference.Global
}

func (t testNode) GetNodeRef() reference.Global {
	return t.ref
}

func (t testNode) GetPublicKey() crypto.PublicKey {
	return nil
}

func (t testNode) GetHost() string {
	return ""
}

func (t testNode) GetBriefDigest() []byte {
	return nil
}

func (t testNode) GetBriefSign() []byte {
	return nil
}

func TestExcludeOrigin(t *testing.T) {
	origin := gen.UniqueGlobalRef()
	originNode := testNode{origin}
	first := testNode{gen.UniqueGlobalRef()}
	second := testNode{gen.UniqueGlobalRef()}

	discoveryNodes := []nodeinfo.DiscoveryNode{first, originNode, second}
	result := ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []nodeinfo.DiscoveryNode{first, second}, result)

	discoveryNodes = []nodeinfo.DiscoveryNode{first, second}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, discoveryNodes, result)

	discoveryNodes = []nodeinfo.DiscoveryNode{first, originNode}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []nodeinfo.DiscoveryNode{first}, result)

	discoveryNodes = []nodeinfo.DiscoveryNode{originNode, first}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []nodeinfo.DiscoveryNode{first}, result)

	discoveryNodes = []nodeinfo.DiscoveryNode{originNode}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Empty(t, result)

	discoveryNodes = []nodeinfo.DiscoveryNode{originNode, first, second}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []nodeinfo.DiscoveryNode{first, second}, result)

	discoveryNodes = []nodeinfo.DiscoveryNode{first, second, originNode}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []nodeinfo.DiscoveryNode{first, second}, result)

}

func regenerateShortID(nodes []nodeinfo.NetworkNode, shortID node.ShortNodeID) node.ShortNodeID {
	shortIDs := make([]node.ShortNodeID, len(nodes))
	for i, activeNode := range nodes {
		shortIDs[i] = activeNode.GetNodeID()
	}
	sort.Slice(shortIDs, func(i, j int) bool {
		return shortIDs[i] < shortIDs[j]
	})
	return generateNonConflictingID(shortIDs, shortID)
}


func generateNonConflictingID(sortedSlice []node.ShortNodeID, conflictingID node.ShortNodeID) node.ShortNodeID {
	index := sort.Search(len(sortedSlice), func(i int) bool {
		return sortedSlice[i] >= conflictingID
	})
	result := conflictingID
	repeated := false
	for {
		if result == math.MaxUint32 {
			if !repeated {
				repeated = true
				result = 0
				index = 0
			} else {
				panic("[ generateNonConflictingID ] shortID overflow twice")
			}
		}
		index++
		result++
		if index >= len(sortedSlice) || result != sortedSlice[index] {
			return result
		}
	}
}

