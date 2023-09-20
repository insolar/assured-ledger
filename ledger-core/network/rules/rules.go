package rules

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// CheckMajorityRule returns error if MajorityRule check not passed, also returns active discovery nodes count
func CheckMajorityRule(cert nodeinfo.Certificate, pop census.OnlinePopulation) (int, error) {

	nodes := pop.GetProfiles()

	majorityRule := cert.GetMajorityRule()
	discoveriesInList := network.FindDiscoveriesInNodeList(nodes, cert)
	activeDiscoveryNodesLen := len(discoveriesInList)
	if activeDiscoveryNodesLen >= majorityRule {
		return activeDiscoveryNodesLen, nil
	}
	strErr := fmt.Sprintf("Active discovery nodes len actual %d, expected %d. Not active ",
		activeDiscoveryNodesLen, majorityRule)
	discoveries := cert.GetDiscoveryNodes()
	for _, d := range discoveries {
		var found bool
		for _, n := range nodes {
			if d.GetNodeRef().Equal(nodeinfo.NodeRef(n)) {
				found = true
				break
			}
		}
		if !found {
			strErr += d.GetHost()
		}
	}
	return activeDiscoveryNodesLen, errors.W(errors.New(strErr), "MajorityRule failed")
}

// CheckMinRole returns true if MinRole check passed
func CheckMinRole(cert nodeinfo.Certificate, nodes census.OnlinePopulation) error {
	var virtualCount, heavyCount, lightCount uint
	if rp := nodes.GetRolePopulation(member.PrimaryRoleVirtual); rp != nil {
		virtualCount = uint(rp.GetWorkingCount() + rp.GetIdleCount())
	}
	if rp := nodes.GetRolePopulation(member.PrimaryRoleLightMaterial); rp != nil {
		lightCount = uint(rp.GetWorkingCount() + rp.GetIdleCount())
	}
	if rp := nodes.GetRolePopulation(member.PrimaryRoleHeavyMaterial); rp != nil {
		heavyCount = uint(rp.GetWorkingCount() + rp.GetIdleCount())
	}

	v, h, l := cert.GetMinRoles()
	if virtualCount >= v &&
		heavyCount >= h &&
		lightCount >= l {
		return nil
	}

	err := errors.New(fmt.Sprintf("%s actual %d expected %d, %s actual %d expected %d, %s actual %d expected %d",
		member.PrimaryRoleVirtual.String(), virtualCount, v,
		member.PrimaryRoleHeavyMaterial.String(), heavyCount, h,
		member.PrimaryRoleLightMaterial.String(), lightCount, l))
	return errors.W(err, "MinRoles failed")
}
