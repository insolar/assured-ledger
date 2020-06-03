// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rules

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"

	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func TestRules_CheckMinRole(t *testing.T) {
	cert := testutils.NewCertificateMock(t)
	nodes := []node2.NetworkNode{
		node.NewNode(gen.UniqueReference(), node2.StaticRoleHeavyMaterial, nil, "", ""),
		node.NewNode(gen.UniqueReference(), node2.StaticRoleLightMaterial, nil, "", ""),
		node.NewNode(gen.UniqueReference(), node2.StaticRoleLightMaterial, nil, "", ""),
		node.NewNode(gen.UniqueReference(), node2.StaticRoleVirtual, nil, "", ""),
		node.NewNode(gen.UniqueReference(), node2.StaticRoleVirtual, nil, "", ""),
	}
	cert.GetMinRolesMock.Set(func() (r uint, r1 uint, r2 uint) {
		return 1, 0, 0
	})
	err := CheckMinRole(cert, nodes)
	require.NoError(t, err)

	cert.GetMinRolesMock.Set(func() (r uint, r1 uint, r2 uint) {
		return 3, 2, 4
	})
	err = CheckMinRole(cert, nodes)
	require.Error(t, err)
}

func TestRules_CheckMajorityRule(t *testing.T) {
	discNodesCount := 5
	netNodes, discoveryNodes := getDiscoveryNodes(discNodesCount)
	cert := testutils.NewCertificateMock(t)
	cert.GetDiscoveryNodesMock.Set(func() (r []node2.DiscoveryNode) {
		return discoveryNodes
	})
	cert.GetMajorityRuleMock.Set(func() (r int) {
		return discNodesCount
	})
	count, err := CheckMajorityRule(cert, netNodes)
	require.NoError(t, err)

	require.Equal(t, discNodesCount, count)

	netNodes = netNodes[:len(netNodes)-len(netNodes)/2]
	count, err = CheckMajorityRule(cert, netNodes)
	require.Error(t, err)
	require.Equal(t, len(netNodes), count)
}

func getDiscoveryNodes(count int) ([]node2.NetworkNode, []node2.DiscoveryNode) {
	netNodes := make([]node2.NetworkNode, count)
	discoveryNodes := make([]node2.DiscoveryNode, count)
	for i := 0; i < count; i++ {
		n := newNode(gen.UniqueReference(), i)
		d := mandates.NewBootstrapNode(nil, "", n.Address(), n.ID().String(), n.Role().String())
		netNodes[i] = n
		discoveryNodes[i] = d
	}
	return netNodes, discoveryNodes
}

func newNode(ref reference.Global, i int) node2.NetworkNode {
	return node.NewNode(ref, node2.AllStaticRoles[i%len(node2.AllStaticRoles)], nil,
		"127.0.0.1:"+strconv.Itoa(30000+i), "")
}
