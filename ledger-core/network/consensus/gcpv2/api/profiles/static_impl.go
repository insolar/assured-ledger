// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package profiles

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

func newStaticProfile(v BriefCandidateProfile, pks cryptkit.PublicKeyStore, endpoints []endpoints.Outbound) FixedStaticProfile {
	return FixedStaticProfile{
		endpoints:        endpoints,
		nodeID:           v.GetStaticNodeID(),
		primaryRole:      v.GetPrimaryRole(),
		specialRoles:     v.GetSpecialRoles(),
		pk:               v.GetNodePublicKey(),
		pks:              pks,
		startPower:       v.GetStartPower(),
		briefIntroSigned: v.GetBriefIntroSignedDigest(),
		isFull:           false,
	}
}

func NewStaticProfileByBrief(v BriefCandidateProfile, pks cryptkit.PublicKeyStore) StaticProfile {
	r := newStaticProfile(v, pks, []endpoints.Outbound{v.GetDefaultEndpoint()})
	return &r
}

func NewStaticProfileByFull(v CandidateProfile, pks cryptkit.PublicKeyStore) StaticProfile {
	r := newStaticProfile(v, pks, []endpoints.Outbound{v.GetDefaultEndpoint()})
	r.setFull(v)
	return &r
}

func NewStaticProfileByExt(v BriefCandidateProfile, ext CandidateProfileExtension, pks cryptkit.PublicKeyStore) StaticProfile {
	r := newStaticProfile(v, pks, []endpoints.Outbound{v.GetDefaultEndpoint()})
	r.setFull(ext)
	return &r
}

type FixedStaticProfile struct {
	endpoints        []endpoints.Outbound
	nodeID           node.ShortNodeID
	primaryRole      member.PrimaryRole
	specialRoles     member.SpecialRole
	startPower       member.Power
	briefIntroSigned cryptkit.SignedDigestHolder
	pk               cryptkit.SigningKeyHolder
	pks              cryptkit.PublicKeyStore

	isFull   bool
	powerSet member.PowerSet
	nodeRef  reference.Global

	issuedAtPulse   pulse.Number // =0 when a node was connected during zeronet
	issuedAtTime    time.Time
	issuerID        node.ShortNodeID
	issuerSignature cryptkit.SignatureHolder
}

func (p *FixedStaticProfile) GetExtraEndpoints() []endpoints.Outbound {
	p.ensureFull()
	return p.endpoints[1:]
}

func (p *FixedStaticProfile) GetBriefIntroSignedDigest() cryptkit.SignedDigestHolder {
	return p.briefIntroSigned
}

func (p *FixedStaticProfile) ensureFull() {
	if p.isFull {
		return
	}
	panic("illegal state")
}

func (p *FixedStaticProfile) GetPowerLevels() member.PowerSet {
	p.ensureFull()
	return p.powerSet
}

func (p *FixedStaticProfile) GetIssuedAtPulse() pulse.Number {
	p.ensureFull()
	return p.issuedAtPulse
}

func (p *FixedStaticProfile) GetIssuedAtTime() time.Time {
	p.ensureFull()
	return p.issuedAtTime
}

func (p *FixedStaticProfile) GetIssuerID() node.ShortNodeID {
	p.ensureFull()
	return p.issuerID
}

func (p *FixedStaticProfile) GetIssuerSignature() cryptkit.SignatureHolder {
	p.ensureFull()
	return p.issuerSignature
}

func (p *FixedStaticProfile) GetReference() reference.Global {
	p.ensureFull()
	return p.nodeRef
}

func (p *FixedStaticProfile) GetIntroducedNodeID() node.ShortNodeID {
	p.ensureFull()
	return p.nodeID
}

func (p *FixedStaticProfile) GetDefaultEndpoint() endpoints.Outbound {
	return p.endpoints[0]
}

func (p *FixedStaticProfile) GetPublicKeyStore() cryptkit.PublicKeyStore {
	return p.pks
}

func (p *FixedStaticProfile) IsAcceptableHost(from endpoints.Inbound) bool {
	for _, ep := range p.endpoints {
		if ep.CanAccept(from) {
			return true
		}
	}
	return false
}

func (p *FixedStaticProfile) GetStaticNodeID() node.ShortNodeID {
	return p.nodeID
}

func (p *FixedStaticProfile) GetPrimaryRole() member.PrimaryRole {
	return p.primaryRole
}

func (p *FixedStaticProfile) GetSpecialRoles() member.SpecialRole {
	return p.specialRoles
}

func (p *FixedStaticProfile) GetNodePublicKey() cryptkit.SigningKeyHolder {
	return p.pk
}

func (p *FixedStaticProfile) GetStartPower() member.Power {
	return p.startPower
}

func (p *FixedStaticProfile) GetExtension() StaticProfileExtension {
	if p.isFull {
		return p
	}
	return nil
}

func (p *FixedStaticProfile) setFull(v CandidateProfileExtension) {

	p.powerSet = v.GetPowerLevels()
	p.nodeRef = v.GetReference()
	p.issuedAtPulse = v.GetIssuedAtPulse()
	p.issuedAtTime = v.GetIssuedAtTime()
	p.issuerID = v.GetIssuerID()
	p.issuerSignature = v.GetIssuerSignature()

	extraEndpoints := v.GetExtraEndpoints()
	p.endpoints = append(append(make([]endpoints.Outbound, 0, len(extraEndpoints)+1),
		p.GetDefaultEndpoint()), extraEndpoints...)

	p.isFull = true
}

func NewUpgradableProfileByBrief(v BriefCandidateProfile, pks cryptkit.PublicKeyStore) StaticProfile {
	return &UpgradableStaticProfile{FixedStaticProfile: newStaticProfile(v, pks, []endpoints.Outbound{v.GetDefaultEndpoint()})}
}

var _ Upgradable = &UpgradableStaticProfile{}

type UpgradableStaticProfile struct {
	mutex sync.RWMutex
	FixedStaticProfile
}

func (p *UpgradableStaticProfile) GetExtension() StaticProfileExtension {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.isFull {
		return p
	}
	return nil
}

func (p *UpgradableStaticProfile) UpgradeProfile(v CandidateProfileExtension) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isFull {
		return false
	}

	p.setFull(v)
	return true
}
