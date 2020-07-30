// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package network

import (
	"crypto"
	"math"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"
)

func newTestNode() mutable.Node {
	return mutable.NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleUnknown, nil, "127.0.0.1:5432")
}

func newTestNodeWithShortID(id node2.ShortNodeID) mutable.Node {
	n := newTestNode()
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

	require.False(t, CheckShortIDCollision(nodes, node2.ShortNodeID(2)))
	require.False(t, CheckShortIDCollision(nodes, node2.ShortNodeID(31)))
	require.False(t, CheckShortIDCollision(nodes, node2.ShortNodeID(35)))
	require.False(t, CheckShortIDCollision(nodes, node2.ShortNodeID(65)))

	require.True(t, CheckShortIDCollision(nodes, node2.ShortNodeID(30)))
	require.Equal(t, node2.ShortNodeID(31), regenerateShortID(nodes, node2.ShortNodeID(30)))

	require.True(t, CheckShortIDCollision(nodes, node2.ShortNodeID(32)))
	require.Equal(t, node2.ShortNodeID(35), regenerateShortID(nodes, node2.ShortNodeID(32)))

	require.True(t, CheckShortIDCollision(nodes, node2.ShortNodeID(64)))
	require.Equal(t, node2.ShortNodeID(65), regenerateShortID(nodes, node2.ShortNodeID(64)))

	require.True(t, CheckShortIDCollision(nodes, node2.ShortNodeID(1<<32-2)))
	require.Equal(t, node2.ShortNodeID(2), regenerateShortID(nodes, node2.ShortNodeID(1<<32-2)))
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

func (t testNode) GetRole() member.PrimaryRole {
	return member.PrimaryRoleVirtual
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

func regenerateShortID(nodes []nodeinfo.NetworkNode, shortID node2.ShortNodeID) node2.ShortNodeID {
	shortIDs := make([]node2.ShortNodeID, len(nodes))
	for i, activeNode := range nodes {
		shortIDs[i] = activeNode.GetNodeID()
	}
	sort.Slice(shortIDs, func(i, j int) bool {
		return shortIDs[i] < shortIDs[j]
	})
	return generateNonConflictingID(shortIDs, shortID)
}


func generateNonConflictingID(sortedSlice []node2.ShortNodeID, conflictingID node2.ShortNodeID) node2.ShortNodeID {
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

