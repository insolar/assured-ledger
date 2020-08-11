// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package serialization

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/phases"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/proofs"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type packetData struct {
	data   []byte // nolint
	packet *Packet
}

func (p *packetData) GetPulseNumber() pulse.Number {
	return p.packet.getPulseNumber()
}

type PacketParser struct {
	packetData
	digester     cryptkit.DataDigester
	signMethod   cryptkit.SigningMethod
	keyProcessor cryptography.KeyProcessor
}

func newPacketParser(
	ctx context.Context,
	reader io.Reader,
	digester cryptkit.DataDigester,
	signMethod cryptkit.SigningMethod,
	keyProcessor cryptography.KeyProcessor,
) (*PacketParser, error) {

	capture := network.NewCapturingReader(reader)
	parser := &PacketParser{
		packetData: packetData{
			packet: new(Packet),
		},
		digester:     digester,
		signMethod:   signMethod,
		keyProcessor: keyProcessor,
	}

	_, err := parser.packet.DeserializeFrom(ctx, capture)
	if err != nil {
		return nil, err
	}

	parser.data = capture.Captured()

	return parser, nil
}

func (p PacketParser) String() string {
	return p.packet.String()
}

func (p *PacketParser) ParsePacketBody() (transport.PacketParser, error) {
	return nil, nil
}

type PacketParserFactory struct {
	digester     cryptkit.DataDigester
	signMethod   cryptkit.SigningMethod
	keyProcessor cryptography.KeyProcessor
}

func NewPacketParserFactory(
	digester cryptkit.DataDigester,
	signMethod cryptkit.SigningMethod,
	keyProcessor cryptography.KeyProcessor,
) *PacketParserFactory {

	return &PacketParserFactory{
		digester:     digester,
		signMethod:   signMethod,
		keyProcessor: keyProcessor,
	}
}

func (f *PacketParserFactory) ParsePacket(ctx context.Context, reader io.Reader) (transport.PacketParser, error) {
	return newPacketParser(ctx, reader, f.digester, f.signMethod, f.keyProcessor)
}

func (p *PacketParser) GetPulsePacket() transport.PulsePacketReader {
	pulsarBody := p.packet.EncryptableBody.(*PulsarPacketBody)
	return adapters.NewPulsePacketParser(pulsarBody.getPulseData())
}

func (p *PacketParser) GetMemberPacket() transport.MemberPacketReader {
	return &MemberPacketReader{
		PacketParser: *p,
		body:         p.packet.EncryptableBody.(*GlobulaConsensusPacketBody),
	}
}

func (p *PacketParser) GetSourceID() node.ShortNodeID {
	return p.packet.Header.GetSourceID()
}

func (p *PacketParser) GetReceiverID() node.ShortNodeID {
	return node.ShortNodeID(p.packet.Header.ReceiverID)
}

func (p *PacketParser) GetTargetID() node.ShortNodeID {
	return node.ShortNodeID(p.packet.Header.TargetID)
}

func (p *PacketParser) GetPacketType() phases.PacketType {
	return p.packet.Header.GetPacketType()
}

func (p *PacketParser) IsRelayForbidden() bool {
	return p.packet.Header.IsRelayRestricted()
}

func (p *PacketParser) GetPacketSignature() cryptkit.SignedDigest {
	payloadReader := bytes.NewReader(p.data[:len(p.data)-signatureSize])

	signature := cryptkit.NewSignature(&p.packet.PacketSignature, p.digester.GetDigestMethod().SignedBy(p.signMethod))
	digest := p.digester.DigestData(payloadReader)
	return cryptkit.NewSignedDigest(digest, signature)
}

type MemberPacketReader struct {
	PacketParser
	body *GlobulaConsensusPacketBody
}

func (r *MemberPacketReader) AsPhase0Packet() transport.Phase0PacketReader {
	return &Phase0PacketReader{
		MemberPacketReader: *r,
		EmbeddedPulseReader: EmbeddedPulseReader{
			MemberPacketReader: *r,
		},
	}
}

func (r *MemberPacketReader) AsPhase1Packet() transport.Phase1PacketReader {
	return &Phase1PacketReader{
		MemberPacketReader: *r,
		ExtendedIntroReader: ExtendedIntroReader{
			MemberPacketReader: *r,
		},
		EmbeddedPulseReader: EmbeddedPulseReader{
			MemberPacketReader: *r,
		},
	}
}

func (r *MemberPacketReader) AsPhase2Packet() transport.Phase2PacketReader {
	return &Phase2PacketReader{
		MemberPacketReader: *r,
		ExtendedIntroReader: ExtendedIntroReader{
			MemberPacketReader: *r,
		},
	}
}

func (r *MemberPacketReader) AsPhase3Packet() transport.Phase3PacketReader {
	return &Phase3PacketReader{*r}
}

type EmbeddedPulseReader struct {
	MemberPacketReader
}

func (r *EmbeddedPulseReader) HasPulseData() bool {
	return r.packet.Header.HasFlag(FlagHasPulsePacket)
}

func (r *EmbeddedPulseReader) GetEmbeddedPulsePacket() transport.PulsePacketReader {
	if !r.HasPulseData() {
		return nil
	}

	return adapters.NewPulsePacketParser(r.body.PulsarPacket.PulsarPacketBody.getPulseData())
}

type Phase0PacketReader struct {
	MemberPacketReader
	EmbeddedPulseReader
}

func (r *Phase0PacketReader) GetNodeRank() member.Rank {
	return r.body.CurrentRank
}

type ExtendedIntroReader struct {
	MemberPacketReader
}

func (r *ExtendedIntroReader) HasFullIntro() bool {
	flags := r.packet.Header.GetFlagRangeInt(1, 2)
	return flags == 2 || flags == 3
}

func (r *ExtendedIntroReader) HasCloudIntro() bool {
	flags := r.packet.Header.GetFlagRangeInt(1, 2)
	return flags == 2 || flags == 3
}

func (r *ExtendedIntroReader) HasJoinerSecret() bool {
	return r.packet.Header.GetFlagRangeInt(1, 2) == 3
}

func (r *ExtendedIntroReader) GetFullIntroduction() transport.FullIntroductionReader {
	if !r.HasFullIntro() {
		return nil
	}

	return &FullIntroductionReader{
		MemberPacketReader: r.MemberPacketReader,
		intro:              r.body.FullSelfIntro,
		nodeID:             node.ShortNodeID(r.packet.Header.SourceID),
	}
}

func (r *ExtendedIntroReader) GetCloudIntroduction() transport.CloudIntroductionReader {
	if !r.HasCloudIntro() {
		return nil
	}

	return &CloudIntroductionReader{
		MemberPacketReader: r.MemberPacketReader,
	}
}

func (r *ExtendedIntroReader) GetJoinerSecret() cryptkit.DigestHolder {
	if !r.HasJoinerSecret() {
		return nil
	}

	return cryptkit.NewDigest(
		&r.body.JoinerSecret,
		r.digester.GetDigestMethod(),
	).AsDigestHolder()
}

type Phase1PacketReader struct {
	MemberPacketReader
	ExtendedIntroReader
	EmbeddedPulseReader
}

func (r *Phase1PacketReader) GetAnnouncementReader() transport.MembershipAnnouncementReader {
	return &MembershipAnnouncementReader{
		MemberPacketReader: r.MemberPacketReader,
	}
}

type Phase2PacketReader struct {
	MemberPacketReader
	ExtendedIntroReader
}

func (r *Phase2PacketReader) GetBriefIntroduction() transport.BriefIntroductionReader {
	flags := r.packet.Header.GetFlagRangeInt(1, 2)
	if flags != 1 {
		return nil
	}

	return &FullIntroductionReader{
		MemberPacketReader: r.MemberPacketReader,
		intro: NodeFullIntro{
			NodeBriefIntro: r.body.BriefSelfIntro,
		},
		nodeID: node.ShortNodeID(r.packet.Header.SourceID),
	}
}

func (r *Phase2PacketReader) GetAnnouncementReader() transport.MembershipAnnouncementReader {
	return &MembershipAnnouncementReader{
		MemberPacketReader: r.MemberPacketReader,
	}
}

func (r *Phase2PacketReader) GetNeighbourhood() []transport.MembershipAnnouncementReader {
	readers := make([]transport.MembershipAnnouncementReader, r.body.Neighbourhood.NeighbourCount)
	for i := 0; i < int(r.body.Neighbourhood.NeighbourCount); i++ {
		readers[i] = &NeighbourAnnouncementReader{
			MemberPacketReader: r.MemberPacketReader,
			neighbour:          r.body.Neighbourhood.Neighbours[i],
		}
	}

	return readers
}

type Phase3PacketReader struct {
	MemberPacketReader
}

func (r *Phase3PacketReader) hasDoubtedVector() bool {
	return r.packet.Header.GetFlagRangeInt(1, 2) > 0
}

func (r *Phase3PacketReader) GetTrustedGlobulaAnnouncementHash() proofs.GlobulaAnnouncementHash {
	return cryptkit.NewDigest(&r.body.Vectors.MainStateVector.VectorHash, r.digester.GetDigestMethod()).AsDigestHolder()
}

func (r *Phase3PacketReader) GetTrustedExpectedRank() member.Rank {
	return r.body.Vectors.MainStateVector.ExpectedRank
}

func (r *Phase3PacketReader) GetTrustedGlobulaStateSignature() proofs.GlobulaStateSignature {
	return cryptkit.NewSignature(
		&r.body.Vectors.MainStateVector.SignedGlobulaStateHash,
		r.digester.GetDigestMethod().SignedBy(r.signMethod),
	).AsSignatureHolder()
}

func (r *Phase3PacketReader) GetDoubtedGlobulaAnnouncementHash() proofs.GlobulaAnnouncementHash {
	if !r.hasDoubtedVector() {
		return nil
	}

	return cryptkit.NewDigest(&r.body.Vectors.AdditionalStateVectors[0].VectorHash, r.digester.GetDigestMethod()).AsDigestHolder()
}

func (r *Phase3PacketReader) GetDoubtedExpectedRank() member.Rank {
	if !r.hasDoubtedVector() {
		return 0
	}

	return r.body.Vectors.AdditionalStateVectors[0].ExpectedRank
}

func (r *Phase3PacketReader) GetDoubtedGlobulaStateSignature() proofs.GlobulaStateSignature {
	if !r.hasDoubtedVector() {
		return nil
	}

	return cryptkit.NewSignature(
		&r.body.Vectors.AdditionalStateVectors[0].SignedGlobulaStateHash,
		r.digester.GetDigestMethod().SignedBy(r.signMethod),
	).AsSignatureHolder()
}

func (r *Phase3PacketReader) GetBitset() member.StateBitset {
	return r.body.Vectors.StateVectorMask.GetBitset()
}

type CloudIntroductionReader struct {
	MemberPacketReader
}

func (r *CloudIntroductionReader) GetLastCloudStateHash() cryptkit.DigestHolder {
	digest := cryptkit.NewDigest(&r.body.CloudIntro.LastCloudStateHash, r.digester.GetDigestMethod())
	return digest.AsDigestHolder()
}

func (r *CloudIntroductionReader) hasJoinerSecret() bool {
	return r.packet.Header.GetFlagRangeInt(1, 2) == 3
}

func (r *CloudIntroductionReader) GetJoinerSecret() cryptkit.DigestHolder {
	if !r.hasJoinerSecret() {
		return nil
	}

	return cryptkit.NewDigest(&r.body.JoinerSecret, r.digester.GetDigestMethod()).AsDigestHolder()
}

func (r *CloudIntroductionReader) GetCloudIdentity() cryptkit.DigestHolder {
	digest := cryptkit.NewDigest(&r.body.CloudIntro.CloudIdentity, r.digester.GetDigestMethod())
	return digest.AsDigestHolder()
}

type FullIntroductionReader struct {
	MemberPacketReader
	intro  NodeFullIntro
	nodeID node.ShortNodeID
}

func (r *FullIntroductionReader) GetBriefIntroSignedDigest() cryptkit.SignedDigestHolder {
	return cryptkit.NewSignedDigest(
		r.digester.DigestData(bytes.NewReader(r.intro.JoinerData)),
		cryptkit.NewSignature(&r.intro.JoinerSignature, r.digester.GetDigestMethod().SignedBy(r.signMethod)),
	).AsSignedDigestHolder()
}

func (r *FullIntroductionReader) GetStaticNodeID() node.ShortNodeID {
	return r.nodeID
}

func (r *FullIntroductionReader) GetPrimaryRole() member.PrimaryRole {
	return r.intro.GetPrimaryRole()
}

func (r *FullIntroductionReader) GetSpecialRoles() member.SpecialRole {
	return r.intro.SpecialRoles
}

func (r *FullIntroductionReader) GetStartPower() member.Power {
	return r.intro.StartPower
}

func (r *FullIntroductionReader) GetNodePublicKey() cryptkit.SignatureKeyHolder {
	return legacyadapter.NewECDSASignatureKeyHolderFromBits(r.intro.NodePK, r.keyProcessor)
}

func (r *FullIntroductionReader) GetDefaultEndpoint() endpoints.Outbound {
	return adapters.NewOutbound(endpoints.IPAddress(r.intro.Endpoint).String())
}

func (r *FullIntroductionReader) GetIssuedAtPulse() pulse.Number {
	return r.intro.IssuedAtPulse
}

func (r *FullIntroductionReader) GetIssuedAtTime() time.Time {
	return time.Unix(0, int64(r.intro.IssuedAtTime))
}

func (r *FullIntroductionReader) GetPowerLevels() member.PowerSet {
	return r.intro.PowerLevels
}

func (r *FullIntroductionReader) GetExtraEndpoints() []endpoints.Outbound {
	// TODO: we have no extra endpoints atm
	return nil
}

func (r *FullIntroductionReader) GetReference() reference.Global {
	if r.body.FullSelfIntro.ProofLen > 0 {
		return reference.GlobalFromBytes(r.intro.NodeRefProof[0].AsBytes())
	}

	return reference.Global{}
}

func (r *FullIntroductionReader) GetIssuerID() node.ShortNodeID {
	return r.intro.DiscoveryIssuerNodeID
}

func (r *FullIntroductionReader) GetIssuerSignature() cryptkit.SignatureHolder {
	return cryptkit.NewSignature(
		&r.intro.IssuerSignature,
		r.digester.GetDigestMethod().SignedBy(r.signMethod),
	).AsSignatureHolder()
}

type MembershipAnnouncementReader struct {
	MemberPacketReader
}

func (r *MembershipAnnouncementReader) isJoiner() bool {
	return r.body.Announcement.CurrentRank.IsJoiner()
}

func (r *MembershipAnnouncementReader) GetNodeID() node.ShortNodeID {
	return node.ShortNodeID(r.packet.Header.SourceID)
}

func (r *MembershipAnnouncementReader) GetNodeRank() member.Rank {
	return r.body.Announcement.CurrentRank
}

func (r *MembershipAnnouncementReader) GetRequestedPower() member.Power {
	return r.body.Announcement.RequestedPower
}

func (r *MembershipAnnouncementReader) GetNodeStateHashEvidence() proofs.NodeStateHashEvidence {
	if r.isJoiner() {
		return nil
	}

	return cryptkit.NewSignedDigest(
		cryptkit.NewDigest(&r.body.Announcement.Member.NodeState.NodeStateHash, r.digester.GetDigestMethod()),
		cryptkit.NewSignature(&r.body.Announcement.Member.NodeState.NodeStateHashSignature, r.digester.GetDigestMethod().SignedBy(r.signMethod)),
	).AsSignedDigestHolder()
}

func (r *MembershipAnnouncementReader) GetAnnouncementSignature() proofs.MemberAnnouncementSignature {
	if r.isJoiner() {
		return nil
	}

	return cryptkit.NewSignature(
		&r.body.Announcement.AnnounceSignature,
		r.digester.GetDigestMethod().SignedBy(r.signMethod),
	).AsSignatureHolder()
}

func (r *MembershipAnnouncementReader) IsLeaving() bool {
	if r.isJoiner() {
		return false
	}

	return r.body.Announcement.Member.AnnounceID == node.ShortNodeID(r.packet.Header.SourceID)
}

func (r *MembershipAnnouncementReader) GetLeaveReason() uint32 {
	if !r.IsLeaving() {
		return 0
	}

	return r.body.Announcement.Member.Leaver.LeaveReason
}

func (r *MembershipAnnouncementReader) GetJoinerID() node.ShortNodeID {
	if r.isJoiner() {
		return node.AbsentShortNodeID
	}

	if r.IsLeaving() {
		return node.AbsentShortNodeID
	}

	return r.body.Announcement.Member.AnnounceID
}

func (r *MembershipAnnouncementReader) GetJoinerAnnouncement() transport.JoinerAnnouncementReader {
	if r.isJoiner() {
		return nil
	}

	if r.body.Announcement.Member.AnnounceID == node.ShortNodeID(r.packet.Header.SourceID) ||
		r.body.Announcement.Member.AnnounceID.IsAbsent() {
		return nil
	}

	var ext *NodeExtendedIntro
	if r.packet.Header.HasFlag(FlagHasJoinerExt) {
		ext = &r.body.JoinerExt
	}

	return &JoinerAnnouncementReader{
		MemberPacketReader: r.MemberPacketReader,
		joiner:             r.body.Announcement.Member.Joiner,
		introducedBy:       node.ShortNodeID(r.packet.Header.SourceID),
		nodeID:             r.body.Announcement.Member.AnnounceID,
		extIntro:           ext,
	}
}

type JoinerAnnouncementReader struct {
	MemberPacketReader
	joiner       JoinAnnouncement
	introducedBy node.ShortNodeID
	nodeID       node.ShortNodeID
	extIntro     *NodeExtendedIntro
}

func (r *JoinerAnnouncementReader) GetJoinerIntroducedByID() node.ShortNodeID {
	return r.introducedBy
}

func (r *JoinerAnnouncementReader) HasFullIntro() bool {
	return r.extIntro != nil
}

func (r *JoinerAnnouncementReader) GetFullIntroduction() transport.FullIntroductionReader {
	if !r.HasFullIntro() {
		return nil
	}

	return &FullIntroductionReader{
		MemberPacketReader: r.MemberPacketReader,
		intro: NodeFullIntro{
			NodeBriefIntro:    r.joiner.NodeBriefIntro,
			NodeExtendedIntro: *r.extIntro,
		},
		nodeID: r.nodeID,
	}
}

func (r *JoinerAnnouncementReader) GetBriefIntroduction() transport.BriefIntroductionReader {
	return &FullIntroductionReader{
		MemberPacketReader: r.MemberPacketReader,
		intro: NodeFullIntro{
			NodeBriefIntro: r.joiner.NodeBriefIntro,
		},
		nodeID: r.nodeID,
	}
}

type NeighbourAnnouncementReader struct {
	MemberPacketReader
	neighbour NeighbourAnnouncement
}

func (r *NeighbourAnnouncementReader) isJoiner() bool {
	return r.neighbour.CurrentRank.IsJoiner()
}

func (r *NeighbourAnnouncementReader) GetNodeID() node.ShortNodeID {
	return r.neighbour.NeighbourNodeID
}

func (r *NeighbourAnnouncementReader) GetNodeRank() member.Rank {
	return r.neighbour.CurrentRank
}

func (r *NeighbourAnnouncementReader) GetRequestedPower() member.Power {
	return r.neighbour.RequestedPower
}

func (r *NeighbourAnnouncementReader) GetNodeStateHashEvidence() proofs.NodeStateHashEvidence {
	return cryptkit.NewSignedDigest(
		cryptkit.NewDigest(&r.neighbour.Member.NodeState.NodeStateHash, r.digester.GetDigestMethod()),
		cryptkit.NewSignature(&r.neighbour.Member.NodeState.NodeStateHashSignature, r.digester.GetDigestMethod().SignedBy(r.signMethod)),
	).AsSignedDigestHolder()
}

func (r *NeighbourAnnouncementReader) GetAnnouncementSignature() proofs.MemberAnnouncementSignature {
	return cryptkit.NewSignature(
		&r.neighbour.AnnounceSignature,
		r.digester.GetDigestMethod().SignedBy(r.signMethod),
	).AsSignatureHolder()
}

func (r *NeighbourAnnouncementReader) IsLeaving() bool {
	if r.isJoiner() {
		return false
	}

	return r.neighbour.Member.AnnounceID == r.neighbour.NeighbourNodeID
}

func (r *NeighbourAnnouncementReader) GetLeaveReason() uint32 {
	if r.IsLeaving() {
		return 0
	}

	return r.neighbour.Member.Leaver.LeaveReason
}

func (r *NeighbourAnnouncementReader) GetJoinerID() node.ShortNodeID {
	if r.isJoiner() {
		return node.AbsentShortNodeID
	}

	if r.IsLeaving() {
		return node.AbsentShortNodeID
	}

	return r.neighbour.Member.AnnounceID
}

func (r *NeighbourAnnouncementReader) GetJoinerAnnouncement() transport.JoinerAnnouncementReader {
	if !r.isJoiner() {
		return nil
	}

	if r.IsLeaving() {
		return nil
	}

	if r.neighbour.NeighbourNodeID == r.body.Announcement.Member.AnnounceID {
		return nil
	}

	return &JoinerAnnouncementReader{
		MemberPacketReader: r.MemberPacketReader,
		joiner:             r.neighbour.Joiner,
		introducedBy:       r.neighbour.JoinerIntroducedBy,
		nodeID:             r.neighbour.NeighbourNodeID,
	}
}
