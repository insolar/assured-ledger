// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package network

import (
	"crypto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func newTestNode() node2.NetworkNode {
	return node.NewNode(gen.UniqueReference(), node2.StaticRoleUnknown, nil, "127.0.0.1:5432", "")
}

func newTestNodeWithShortID(id node2.ShortNodeID) node2.NetworkNode {
	n := newTestNode()
	n.(node.MutableNode).SetShortID(id)
	return n
}

func TestCorrectShortIDCollision(t *testing.T) {

	nodes := []node2.NetworkNode{
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

var _ node2.DiscoveryNode = testNode{}

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

func (t testNode) GetRole() node2.StaticRole {
	return node2.StaticRoleVirtual
}

func TestExcludeOrigin(t *testing.T) {
	origin := gen.UniqueReference()
	originNode := testNode{origin}
	first := testNode{gen.UniqueReference()}
	second := testNode{gen.UniqueReference()}

	discoveryNodes := []node2.DiscoveryNode{first, originNode, second}
	result := ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []node2.DiscoveryNode{first, second}, result)

	discoveryNodes = []node2.DiscoveryNode{first, second}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, discoveryNodes, result)

	discoveryNodes = []node2.DiscoveryNode{first, originNode}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []node2.DiscoveryNode{first}, result)

	discoveryNodes = []node2.DiscoveryNode{originNode, first}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []node2.DiscoveryNode{first}, result)

	discoveryNodes = []node2.DiscoveryNode{originNode}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Empty(t, result)

	discoveryNodes = []node2.DiscoveryNode{originNode, first, second}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []node2.DiscoveryNode{first, second}, result)

	discoveryNodes = []node2.DiscoveryNode{first, second, originNode}
	result = ExcludeOrigin(discoveryNodes, origin)
	assert.Equal(t, []node2.DiscoveryNode{first, second}, result)

}
