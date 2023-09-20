package transport

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/proofs"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func NewNodeAnnouncement(np profiles.ActiveNode, ma profiles.MembershipAnnouncement, nodeCount int,
	pn pulse.Number, joiner *JoinerAnnouncement) *NodeAnnouncementProfile {
	return &NodeAnnouncementProfile{
		static:    np.GetStatic(),
		nodeID:    np.GetNodeID(),
		nodeCount: uint16(nodeCount),
		ma:        ma,
		pn:        pn,
		joiner:    joiner,
	}
}

var _ MembershipAnnouncementReader = &NodeAnnouncementProfile{}

type NodeAnnouncementProfile struct {
	static    profiles.StaticProfile
	ma        profiles.MembershipAnnouncement
	nodeID    node.ShortNodeID
	pn        pulse.Number
	nodeCount uint16
	joiner    *JoinerAnnouncement
}

func (c *NodeAnnouncementProfile) GetRequestedPower() member.Power {
	return c.ma.Membership.RequestedPower
}

func (c *NodeAnnouncementProfile) IsLeaving() bool {
	return c.ma.IsLeaving
}

func (c *NodeAnnouncementProfile) GetLeaveReason() uint32 {
	return c.ma.LeaveReason
}

func (c *NodeAnnouncementProfile) GetJoinerID() node.ShortNodeID {
	return c.ma.JoinerID
}

func (c *NodeAnnouncementProfile) GetJoinerAnnouncement() JoinerAnnouncementReader {
	if c.joiner == nil {
		return nil
	}

	if !c.ma.JoinerID.IsAbsent() && c.joiner.GetBriefIntroduction().GetStaticNodeID() != c.ma.JoinerID {
		panic("illegal state")
	}
	return c.joiner
}

func (c *NodeAnnouncementProfile) GetNodeRank() member.Rank {
	return c.ma.Membership.AsRankUint16(c.nodeCount)
}

func (c *NodeAnnouncementProfile) GetAnnouncementSignature() proofs.MemberAnnouncementSignature {
	return c.ma.Membership.AnnounceSignature
}

func (c *NodeAnnouncementProfile) GetNodeID() node.ShortNodeID {
	return c.nodeID
}

func (c *NodeAnnouncementProfile) GetNodeCount() uint16 {
	return c.nodeCount
}

func (c *NodeAnnouncementProfile) GetNodeStateHashEvidence() proofs.NodeStateHashEvidence {
	return c.ma.Membership.StateEvidence
}

func (c NodeAnnouncementProfile) String() string {
	announcement := ""
	if c.IsLeaving() {
		announcement = fmt.Sprintf(" leave:%d", c.GetLeaveReason())
	} else if !c.GetJoinerID().IsAbsent() {
		joinerIntro := ""
		if c.joiner != nil {
			joinerIntro = "+intro"
		}
		announcement = fmt.Sprintf(" join:%d%s", c.GetJoinerID(), joinerIntro)
	}
	return fmt.Sprintf("{id:%d %03d/%d%s %s}", c.nodeID, c.ma.Membership.Index, c.nodeCount, announcement, c.ma.Membership.StringParts())
}

func (c *NodeAnnouncementProfile) GetMembershipProfile() profiles.MembershipProfile {
	return c.ma.Membership
}

func (c *NodeAnnouncementProfile) GetPulseNumber() pulse.Number {
	return c.pn
}

func (c *NodeAnnouncementProfile) GetStatic() profiles.StaticProfile {
	return c.static
}
