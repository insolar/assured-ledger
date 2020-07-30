// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rules

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"

	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func TestRules_CheckMinRole(t *testing.T) {
	cert := testutils.NewCertificateMock(t)
	nodes := []nodeinfo.NetworkNode{
		mutable.NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleHeavyMaterial, ""),
		mutable.NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleLightMaterial, ""),
		mutable.NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleLightMaterial, ""),
		mutable.NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, ""),
		mutable.NewTestNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, ""),
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
	cert.GetDiscoveryNodesMock.Set(func() (r []nodeinfo.DiscoveryNode) {
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

func getDiscoveryNodes(count int) ([]nodeinfo.NetworkNode, []nodeinfo.DiscoveryNode) {
	netNodes := make([]nodeinfo.NetworkNode, count)
	discoveryNodes := make([]nodeinfo.DiscoveryNode, count)
	for i := 0; i < count; i++ {
		n := newNode(gen.UniqueGlobalRef(), i)
		d := mandates.NewBootstrapNode(nil, "",
			nodeinfo.NodeAddr(n),
			n.GetStatic().GetExtension().GetReference().String(),
			n.GetPrimaryRole().String())
		netNodes[i] = n
		discoveryNodes[i] = d
	}
	return netNodes, discoveryNodes
}

// AllStaticRoles is an array of all possible StaticRoles.
var AllStaticRoles = []member.PrimaryRole{
	member.PrimaryRoleVirtual,
	member.PrimaryRoleLightMaterial,
	member.PrimaryRoleHeavyMaterial,
}

func newNode(ref reference.Global, i int) *mutable.Node {
	return mutable.NewTestNode(ref, AllStaticRoles[i%len(AllStaticRoles)], "127.0.0.1:"+strconv.Itoa(30000+i))
}
