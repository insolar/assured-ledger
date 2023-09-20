package profiles

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/proofs"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type MembershipProfile struct {
	Index          member.Index
	Mode           member.OpMode
	Power          member.Power
	RequestedPower member.Power
	proofs.NodeAnnouncedState
}

// TODO support joiner in MembershipProfile
// func (v MembershipProfile) IsJoiner() bool {
//
// }

func NewMembershipProfile(mode member.OpMode, power member.Power, index member.Index,
	nsh cryptkit.SignedDigestHolder, nas proofs.MemberAnnouncementSignature,
	ep member.Power) MembershipProfile {

	return MembershipProfile{
		Index:          index,
		Power:          power,
		Mode:           mode,
		RequestedPower: ep,
		NodeAnnouncedState: proofs.NodeAnnouncedState{
			StateEvidence:     nsh,
			AnnounceSignature: nas,
		},
	}
}

func NewMembershipProfileForJoiner(brief BriefCandidateProfile) MembershipProfile {

	return MembershipProfile{
		Index:          member.JoinerIndex,
		Power:          0,
		Mode:           0,
		RequestedPower: brief.GetStartPower(),
		NodeAnnouncedState: proofs.NodeAnnouncedState{
			StateEvidence:     brief.GetBriefIntroSignedDigest(),
			AnnounceSignature: brief.GetBriefIntroSignedDigest().GetSignatureHolder(),
		},
	}
}

func NewMembershipProfileByNode(np ActiveNode, nsh cryptkit.SignedDigestHolder, nas proofs.MemberAnnouncementSignature,
	ep member.Power) MembershipProfile {

	idx := member.JoinerIndex
	if !np.IsJoiner() {
		idx = np.GetIndex()
	}

	return NewMembershipProfile(np.GetOpMode(), np.GetDeclaredPower(), idx, nsh, nas, ep)
}

func (p MembershipProfile) IsEmpty() bool {
	return p.StateEvidence == nil || p.AnnounceSignature == nil
}

func (p MembershipProfile) IsJoiner() bool {
	return p.Index.IsJoiner()
}

func (p MembershipProfile) CanIntroduceJoiner() bool {
	return p.Mode.CanIntroduceJoiner(p.Index.IsJoiner())
}

func (p MembershipProfile) AsRank(nc int) member.Rank {
	if p.Index.IsJoiner() {
		return member.JoinerRank
	}
	return member.NewMembershipRank(p.Mode, p.Power, p.Index, member.AsIndex(nc))
}

func (p MembershipProfile) AsRankUint16(nc uint16) member.Rank {
	if p.Index.IsJoiner() {
		return member.JoinerRank
	}
	return member.NewMembershipRank(p.Mode, p.Power, p.Index, member.AsIndexUint16(nc))
}

func (p MembershipProfile) Equals(o MembershipProfile) bool {
	if p.Index != o.Index || p.Power != o.Power || p.IsEmpty() || o.IsEmpty() || p.RequestedPower != o.RequestedPower {
		return false
	}

	return p.NodeAnnouncedState.Equals(o.NodeAnnouncedState)
}

func (p MembershipProfile) StringParts() string {
	if p.Power == p.RequestedPower {
		return fmt.Sprintf("pw:%v se:%v cs:%v", p.Power, p.StateEvidence, p.AnnounceSignature)
	}

	return fmt.Sprintf("pw:%v->%v se:%v cs:%v", p.Power, p.RequestedPower, p.StateEvidence, p.AnnounceSignature)
}

func (p MembershipProfile) String() string {
	index := "joiner"
	if !p.Index.IsJoiner() {
		index = fmt.Sprintf("idx:%d", p.Index)
	}
	return fmt.Sprintf("%s %s", index, p.StringParts())
}

type JoinerAnnouncement struct {
	JoinerProfile  StaticProfile
	IntroducedByID node.ShortNodeID
	JoinerSecret   cryptkit.DigestHolder
}

func (v JoinerAnnouncement) IsEmpty() bool {
	return v.JoinerProfile == nil
}

type MembershipAnnouncement struct {
	Membership   MembershipProfile
	IsLeaving    bool
	LeaveReason  uint32
	JoinerID     node.ShortNodeID
	JoinerSecret cryptkit.DigestHolder
}

type MemberAnnouncement struct {
	MemberID node.ShortNodeID
	MembershipAnnouncement
	Joiner        JoinerAnnouncement
	AnnouncedByID node.ShortNodeID
}

func NewMemberAnnouncement(memberID node.ShortNodeID, mp MembershipProfile,
	announcerID node.ShortNodeID) MemberAnnouncement {

	return MemberAnnouncement{
		MemberID:               memberID,
		MembershipAnnouncement: NewMembershipAnnouncement(mp),
		AnnouncedByID:          announcerID,
	}
}

func NewJoinerAnnouncement(brief StaticProfile,
	announcerID node.ShortNodeID) MemberAnnouncement {

	// TODO joiner secret
	return MemberAnnouncement{
		MemberID:               brief.GetStaticNodeID(),
		MembershipAnnouncement: NewMembershipAnnouncement(NewMembershipProfileForJoiner(brief)),
		AnnouncedByID:          announcerID,
		Joiner: JoinerAnnouncement{
			JoinerProfile:  brief,
			IntroducedByID: announcerID,
		},
	}
}

func NewJoinerIDAnnouncement(joinerID, announcerID node.ShortNodeID) MemberAnnouncement {

	return MemberAnnouncement{
		MemberID:      joinerID,
		AnnouncedByID: announcerID,
	}
}

func NewMemberAnnouncementWithLeave(memberID node.ShortNodeID, mp MembershipProfile, leaveReason uint32,
	announcerID node.ShortNodeID) MemberAnnouncement {

	return MemberAnnouncement{
		MemberID:               memberID,
		MembershipAnnouncement: NewMembershipAnnouncementWithLeave(mp, leaveReason),
		AnnouncedByID:          announcerID,
	}
}

func NewMemberAnnouncementWithJoinerID(memberID node.ShortNodeID, mp MembershipProfile,
	joinerID node.ShortNodeID, joinerSecret cryptkit.DigestHolder,
	announcerID node.ShortNodeID) MemberAnnouncement {

	return MemberAnnouncement{
		MemberID:               memberID,
		MembershipAnnouncement: NewMembershipAnnouncementWithJoinerID(mp, joinerID, joinerSecret),
		AnnouncedByID:          announcerID,
	}
}

func NewMemberAnnouncementWithJoiner(memberID node.ShortNodeID, mp MembershipProfile, joiner JoinerAnnouncement,
	announcerID node.ShortNodeID) MemberAnnouncement {

	return MemberAnnouncement{
		MemberID: memberID,
		MembershipAnnouncement: NewMembershipAnnouncementWithJoinerID(mp,
			joiner.JoinerProfile.GetStaticNodeID(), joiner.JoinerSecret),
		Joiner:        joiner,
		AnnouncedByID: announcerID,
	}
}

func NewMembershipAnnouncement(mp MembershipProfile) MembershipAnnouncement {
	return MembershipAnnouncement{
		Membership: mp,
	}
}

func NewMembershipAnnouncementWithJoinerID(mp MembershipProfile,
	joinerID node.ShortNodeID, joinerSecret cryptkit.DigestHolder) MembershipAnnouncement {

	return MembershipAnnouncement{
		Membership:   mp,
		JoinerID:     joinerID,
		JoinerSecret: joinerSecret,
	}
}

func NewMembershipAnnouncementWithLeave(mp MembershipProfile, leaveReason uint32) MembershipAnnouncement {
	return MembershipAnnouncement{
		Membership:  mp,
		IsLeaving:   true,
		LeaveReason: leaveReason,
	}
}
