package transport

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

func NewBriefJoinerAnnouncement(np profiles.StaticProfile, announcerID node.ShortNodeID, joinerSecret cryptkit.DigestHolder) *JoinerAnnouncement {

	return &JoinerAnnouncement{
		privStaticProfile: np,
		announcerID:       announcerID,
		joinerSecret:      joinerSecret,
		joinerSignature:   np.GetBriefIntroSignedDigest().GetSignatureHolder(),
		disableFull:       true,
	}
}

func NewBriefJoinerAnnouncementByFull(fp JoinerAnnouncementReader) JoinerAnnouncementReader {
	return &fullIntroduction{
		fp.GetBriefIntroduction(),
		nil,
		fp.GetJoinerIntroducedByID(),
	}
}

func NewFullJoinerAnnouncement(np profiles.StaticProfile, announcerID node.ShortNodeID, joinerSecret cryptkit.DigestHolder) *JoinerAnnouncement {

	if np.GetExtension() == nil {
		panic("illegal value")
	}
	return NewAnyJoinerAnnouncement(np, announcerID, joinerSecret)
}

func NewAnyJoinerAnnouncement(np profiles.StaticProfile, announcerID node.ShortNodeID, joinerSecret cryptkit.DigestHolder) *JoinerAnnouncement {
	return &JoinerAnnouncement{
		privStaticProfile: np,
		announcerID:       announcerID,
		joinerSecret:      joinerSecret,
		joinerSignature:   np.GetBriefIntroSignedDigest().GetSignatureHolder(),
	}
}

var _ JoinerAnnouncementReader = &JoinerAnnouncement{}

type privStaticProfile profiles.StaticProfile

type JoinerAnnouncement struct {
	privStaticProfile
	disableFull     bool
	announcerID     node.ShortNodeID
	joinerSecret    cryptkit.DigestHolder
	joinerSignature cryptkit.SignatureHolder
}

func (p *JoinerAnnouncement) GetJoinerIntroducedByID() node.ShortNodeID {
	return p.announcerID
}

func (p *JoinerAnnouncement) HasFullIntro() bool {
	return !p.disableFull && p.privStaticProfile.GetExtension() != nil
}

func (p *JoinerAnnouncement) GetFullIntroduction() FullIntroductionReader {
	if !p.HasFullIntro() {
		return nil
	}
	return &fullIntroduction{
		p.privStaticProfile,
		p.privStaticProfile.GetExtension(),
		p.announcerID,
	}
}

func (p *JoinerAnnouncement) GetBriefIntroduction() BriefIntroductionReader {
	return p
}

func (p *JoinerAnnouncement) GetDecryptedSecret() cryptkit.DigestHolder {
	return p.joinerSecret
}

type fullIntroduction struct {
	profiles.BriefCandidateProfile
	profiles.StaticProfileExtension
	announcerID node.ShortNodeID
}

func (p *fullIntroduction) GetJoinerIntroducedByID() node.ShortNodeID {
	return p.announcerID
}

func (p *fullIntroduction) GetBriefIntroduction() BriefIntroductionReader {
	return p
}

func (p *fullIntroduction) HasFullIntro() bool {
	return p.StaticProfileExtension != nil
}

func (p *fullIntroduction) GetFullIntroduction() FullIntroductionReader {
	if p.HasFullIntro() {
		return p
	}
	return nil
}
