package nodeinfo

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type NetworkNode = profiles.ActiveNode

func NodeAddr(n NetworkNode) string {
	return n.GetStatic().GetDefaultEndpoint().GetIPAddress().String()
}

func NodeRef(n NetworkNode) reference.Global {
	return n.GetStatic().GetExtension().GetReference()
}

func NodeRole(n NetworkNode) member.PrimaryRole {
	return n.GetStatic().GetPrimaryRole()
}

func NodeSignedDigest(n NetworkNode) cryptkit.SignedDigestHolder {
	return n.GetStatic().GetBriefIntroSignedDigest()
}

func NewStaticProfileList(nodes []NetworkNode) []profiles.StaticProfile {
	intros := make([]profiles.StaticProfile, len(nodes))
	for i, n := range nodes {
		intros[i] = n.GetStatic()
	}

	profiles.SortStaticProfiles(intros, false)

	return intros
}

func NewNetworkNode(profile profiles.ActiveNode) NetworkNode {
	if profile == nil {
		panic(throw.IllegalValue())
	}
	return profile
}

func NewNetworkNodeList(profiles []profiles.ActiveNode) []NetworkNode {
	networkNodes := make([]NetworkNode, len(profiles))
	for i, p := range profiles {
		networkNodes[i] = NewNetworkNode(p)
	}

	return networkNodes
}
