package serialization

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/proofs"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/statevector"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func fillPulsarPacket(p *EmbeddedPulsarData, pulsarPacket proofs.OriginalPulsarPacket) {
	p.setData(longbits.AsBytes(pulsarPacket))
}

func fillNodeState(s *CompactGlobulaNodeState, nodeStateHash proofs.NodeStateHashEvidence) {
	nodeStateHash.GetDigestHolder().CopyTo(s.NodeStateHash[:])
	nodeStateHash.GetSignatureHolder().CopyTo(s.NodeStateHashSignature[:])
}

func fillMembershipAnnouncement(a *MembershipAnnouncement, sender transport.MembershipAnnouncementReader) {
	a.ShortID = sender.GetNodeID()
	a.CurrentRank = sender.GetNodeRank()
	a.RequestedPower = sender.GetRequestedPower()

	if sender.GetNodeRank().IsJoiner() {
		return
	}

	sender.GetAnnouncementSignature().CopyTo(a.AnnounceSignature[:])

	fillNodeState(&a.Member.NodeState, sender.GetNodeStateHashEvidence())

	if sender.IsLeaving() {
		a.Member.AnnounceID = sender.GetNodeID()
		a.Member.Leaver.LeaveReason = sender.GetLeaveReason()
	} else if joinerAnnouncement := sender.GetJoinerAnnouncement(); joinerAnnouncement != nil {
		a.Member.AnnounceID = sender.GetJoinerID()
		fillBriefInto(&a.Member.Joiner.NodeBriefIntro, joinerAnnouncement.GetBriefIntroduction())
	}
}

func fillBriefInto(i *NodeBriefIntro, intro transport.BriefIntroductionReader) {
	i.ShortID = intro.GetStaticNodeID()
	i.SetPrimaryRole(intro.GetPrimaryRole())
	i.SetAddrMode(endpoints.IPEndpoint)
	i.SpecialRoles = intro.GetSpecialRoles()
	i.StartPower = intro.GetStartPower()
	intro.GetNodePublicKey().CopyTo(i.NodePK[:])
	i.Endpoint = intro.GetDefaultEndpoint().GetIPAddress()
	intro.GetBriefIntroSignedDigest().GetSignatureHolder().CopyTo(i.JoinerSignature[:])
}

func fillExtendedIntro(i *NodeExtendedIntro, intro transport.FullIntroductionReader) {
	i.IssuedAtPulse = intro.GetIssuedAtPulse()
	i.IssuedAtTime = uint64(intro.GetIssuedAtTime().UnixNano())
	i.PowerLevels = intro.GetPowerLevels()

	// TODO: no extra endpoints

	i.ProofLen = 1
	i.NodeRefProof = make([]longbits.Bits512, 1)
	copy(i.NodeRefProof[0][:], intro.GetReference().AsBytes())

	i.DiscoveryIssuerNodeID = intro.GetIssuerID()
	intro.GetIssuerSignature().CopyTo(i.IssuerSignature[:])
}

func fillFullInto(i *NodeFullIntro, intro transport.FullIntroductionReader) {
	fillBriefInto(&i.NodeBriefIntro, intro)
	fillExtendedIntro(&i.NodeExtendedIntro, intro)
}

func fillWelcome(b *GlobulaConsensusPacketBody, welcome *proofs.NodeWelcomePackage) {
	welcome.CloudIdentity.CopyTo(b.CloudIntro.CloudIdentity[:])
	welcome.LastCloudStateHash.CopyTo(b.CloudIntro.LastCloudStateHash[:])
	if welcome.JoinerSecret != nil {
		welcome.JoinerSecret.CopyTo(b.JoinerSecret[:])
	}
}

func fillNeighbourhood(n *Neighbourhood, neighbourhood []transport.MembershipAnnouncementReader) {
	n.NeighbourCount = uint8(len(neighbourhood))
	n.Neighbours = make([]NeighbourAnnouncement, len(neighbourhood))
	for i, neighbour := range neighbourhood {
		fillNeighbourAnnouncement(&n.Neighbours[i], neighbour)
	}
}

func fillNeighbourAnnouncement(n *NeighbourAnnouncement, neighbour transport.MembershipAnnouncementReader) {
	n.NeighbourNodeID = neighbour.GetNodeID()
	n.CurrentRank = neighbour.GetNodeRank()
	n.RequestedPower = neighbour.GetRequestedPower()
	neighbour.GetAnnouncementSignature().CopyTo(n.AnnounceSignature[:])

	if !neighbour.GetNodeRank().IsJoiner() {
		fillNodeState(&n.Member.NodeState, neighbour.GetNodeStateHashEvidence())

		if neighbour.IsLeaving() {
			n.Member.AnnounceID = neighbour.GetNodeID()
			n.Member.Leaver.LeaveReason = neighbour.GetLeaveReason()
		} else {
			n.Member.AnnounceID = neighbour.GetJoinerID()
		}
	} else if announcement := neighbour.GetJoinerAnnouncement(); announcement != nil {
		n.JoinerIntroducedBy = announcement.GetJoinerIntroducedByID()
		fillBriefInto(&n.Joiner.NodeBriefIntro, announcement.GetBriefIntroduction())
	}
}

func fillVector(v *GlobulaStateVector, vector statevector.SubVector) {
	vector.AnnouncementHash.CopyTo(v.VectorHash[:])
	vector.StateSignature.CopyTo(v.SignedGlobulaStateHash[:])
	v.ExpectedRank = vector.ExpectedRank
}

func fillPhase0(
	body *GlobulaConsensusPacketBody,
	sender *transport.NodeAnnouncementProfile,
	pulsarPacket proofs.OriginalPulsarPacket,
) {

	body.CurrentRank = sender.GetNodeRank()
	fillPulsarPacket(&body.PulsarPacket, pulsarPacket)
}

func fillPhase1(
	body *GlobulaConsensusPacketBody,
	sender *transport.NodeAnnouncementProfile,
	pulsarPacket proofs.OriginalPulsarPacket,
	welcome *proofs.NodeWelcomePackage,
) {
	fillPulsarPacket(&body.PulsarPacket, pulsarPacket)
	fillMembershipAnnouncement(&body.Announcement, sender)

	if joiner := sender.GetJoinerAnnouncement(); joiner != nil && joiner.HasFullIntro() {
		fillExtendedIntro(&body.JoinerExt, joiner.GetFullIntroduction())
	}

	staticProfile := sender.GetStatic()
	fillBriefInto(&body.BriefSelfIntro, staticProfile)

	if staticProfileExtension := staticProfile.GetExtension(); staticProfileExtension != nil {
		fillFullInto(&body.FullSelfIntro, &fullReader{
			StaticProfile:          staticProfile,
			StaticProfileExtension: staticProfileExtension,
		})
	}

	if welcome != nil {
		fillWelcome(body, welcome)
	}

	// TODO:
	// Fill Claims
}

func fullPhase2(
	body *GlobulaConsensusPacketBody,
	sender *transport.NodeAnnouncementProfile,
	welcome *proofs.NodeWelcomePackage,
	neighbourhood []transport.MembershipAnnouncementReader,
) {
	fillMembershipAnnouncement(&body.Announcement, sender)

	staticProfile := sender.GetStatic()
	fillBriefInto(&body.BriefSelfIntro, staticProfile)

	if staticProfileExtension := staticProfile.GetExtension(); staticProfileExtension != nil {
		fillFullInto(&body.FullSelfIntro, &fullReader{
			StaticProfile:          staticProfile,
			StaticProfileExtension: staticProfileExtension,
		})
	}

	if welcome != nil {
		fillWelcome(body, welcome)
	}

	fillNeighbourhood(&body.Neighbourhood, neighbourhood)
}

func fillPhase3(body *GlobulaConsensusPacketBody, vectors statevector.Vector) {
	body.Vectors.StateVectorMask.SetBitset(vectors.Bitset)
	fillVector(&body.Vectors.MainStateVector, vectors.Trusted)
	if vectors.Doubted.AnnouncementHash != nil {
		body.Vectors.AdditionalStateVectors = make([]GlobulaStateVector, 1)
		fillVector(&body.Vectors.AdditionalStateVectors[0], vectors.Doubted)
	}

	// TODO:
	// Fill Claims
}
