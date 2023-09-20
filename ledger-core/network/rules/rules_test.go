package rules

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"

	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func TestRules_CheckMinRole(t *testing.T) {
	role := census.NewRolePopulationMock(t)
	role.GetWorkingCountMock.Return(1)
	role.GetIdleCountMock.Return(1)

	pop := census.NewOnlinePopulationMock(t)
	pop.GetRolePopulationMock.Return(role)

	cert := testutils.NewCertificateMock(t)
	cert.GetMinRolesMock.Return(2, 0, 0)
	err := CheckMinRole(cert, pop)
	require.NoError(t, err)

	cert.GetMinRolesMock.Return(3, 2, 4)
	err = CheckMinRole(cert, pop)
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

	pop := census.NewOnlinePopulationMock(t)
	pop.GetProfilesMock.Return(netNodes)

	count, err := CheckMajorityRule(cert, pop)
	require.NoError(t, err)

	require.Equal(t, discNodesCount, count)

	netNodes = netNodes[:len(netNodes)-len(netNodes)/2]
	pop = census.NewOnlinePopulationMock(t)
	pop.GetProfilesMock.Return(netNodes)

	count, err = CheckMajorityRule(cert, pop)
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
