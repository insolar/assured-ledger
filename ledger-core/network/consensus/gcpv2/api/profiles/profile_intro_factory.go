// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package profiles

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

func NewSimpleProfileIntroFactory(pksFactory cryptkit.KeyStoreFactory) Factory {
	return &SimpleProfileIntroFactory{pksFactory}
}

var _ Factory = &SimpleProfileIntroFactory{}

type SimpleProfileIntroFactory struct {
	pksFactory cryptkit.KeyStoreFactory
}

func (p *SimpleProfileIntroFactory) TryConvertUpgradableIntroProfile(profile StaticProfile) (StaticProfile, bool) {
	ext := profile.GetExtension()
	if ext == nil {
		return profile, false
	}
	if _, ok := profile.(Upgradable); !ok {
		return profile, false
	}

	pks := profile.GetPublicKeyStore()
	if pks == nil {
		pks = p.pksFactory.CreatePublicKeyStore(profile.GetNodePublicKey())
	}
	return NewStaticProfileByExt(profile, ext, pks), true
}

func (p *SimpleProfileIntroFactory) CreateUpgradableIntroProfile(candidate BriefCandidateProfile) StaticProfile {
	pks := p.pksFactory.CreatePublicKeyStore(candidate.GetNodePublicKey())
	return NewUpgradableProfileByBrief(candidate, pks)
}

func (p *SimpleProfileIntroFactory) CreateBriefIntroProfile(candidate BriefCandidateProfile) StaticProfile {

	pks := p.pksFactory.CreatePublicKeyStore(candidate.GetNodePublicKey())
	return NewStaticProfileByBrief(candidate, pks)
}

func (p *SimpleProfileIntroFactory) CreateFullIntroProfile(candidate CandidateProfile) StaticProfile {

	pks := p.pksFactory.CreatePublicKeyStore(candidate.GetNodePublicKey())
	return NewStaticProfileByFull(candidate, pks)
}
