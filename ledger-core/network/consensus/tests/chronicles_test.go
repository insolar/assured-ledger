// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"fmt"
	"math"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/misbehavior"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/power"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/proofs"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func NewEmuChronicles(intros []profiles.StaticProfile, localNodeIndex int, asJoiner bool,
	primingCloudStateHash proofs.CloudStateHash) api.ConsensusChronicles {

	var localCensus *censusimpl.PrimingCensusTemplate
	registries := &EmuVersionedRegistries{primingCloudStateHash: primingCloudStateHash}

	if asJoiner {
		if len(intros) != 1 && localNodeIndex != 0 {
			panic("illegal state")
		}
		localCensus = censusimpl.NewPrimingCensusForJoiner(intros[localNodeIndex], registries, EmuDefaultCryptography, true)
	} else {
		localCensus = censusimpl.NewPrimingCensus(intros, intros[localNodeIndex], registries, EmuDefaultCryptography, true)
	}

	chronicles := censusimpl.NewLocalChronicles(profiles.NewSimpleProfileIntroFactory(EmuDefaultCryptography))
	localCensus.SetAsActiveTo(chronicles)
	return chronicles
}

func NewEmuNodeIntros(names ...string) []profiles.StaticProfile {
	r := make([]profiles.StaticProfile, len(names))
	for i, n := range names {
		r[i] = NewEmuNodeIntroByName(i, n)
	}
	return r
}

func NewEmuNodeIntroByName(id int, name string) *EmuNodeIntro {

	var sr member.SpecialRole
	var pr member.PrimaryRole
	switch name[0] {
	case 'H':
		pr = member.PrimaryRoleHeavyMaterial
		sr = member.SpecialRoleDiscovery
	case 'L':
		pr = member.PrimaryRoleLightMaterial
	case 'V':
		pr = member.PrimaryRoleVirtual
	default:
		pr = member.PrimaryRoleNeutral
		sr = member.SpecialRoleDiscovery
	}
	return NewEmuNodeIntro(id, endpoints.Name(name), pr, sr)
}

type EmuVersionedRegistries struct {
	pd                    pulse.Data
	primingCloudStateHash proofs.CloudStateHash
}

func (c *EmuVersionedRegistries) GetNearestValidPulseData() pulse.Data {
	panic("implement me")
}

func (c *EmuVersionedRegistries) GetCloudIdentity() cryptkit.DigestHolder {
	return c.primingCloudStateHash
}

func (c *EmuVersionedRegistries) GetConsensusConfiguration() census.ConsensusConfiguration {
	return c
}

func (c *EmuVersionedRegistries) GetPrimingCloudHash() proofs.CloudStateHash {
	return c.primingCloudStateHash
}

func (c *EmuVersionedRegistries) FindRegisteredProfile(identity endpoints.Inbound) profiles.Host {
	return NewEmuNodeIntro(-1, identity.GetNameAddress(),
		/* unused by HostProfile */ member.PrimaryRole(math.MaxUint8), 0)
}

func (c *EmuVersionedRegistries) AddReport(report misbehavior.Report) {
}

func (c *EmuVersionedRegistries) CommitNextPulse(pd pulse.Data, population census.OnlinePopulation) census.VersionedRegistries {
	pd.EnsurePulseData()
	cp := *c
	cp.pd = pd
	return &cp
}

func (c *EmuVersionedRegistries) GetMisbehaviorRegistry() census.MisbehaviorRegistry {
	return c
}

func (c *EmuVersionedRegistries) GetMandateRegistry() census.MandateRegistry {
	return c
}

func (c *EmuVersionedRegistries) GetOfflinePopulation() census.OfflinePopulation {
	return c
}

func (c *EmuVersionedRegistries) GetVersionPulseData() pulse.Data {
	return c.pd
}

const ShortNodeIdOffset = 1000

func NewEmuNodeIntro(id int, s endpoints.Name, pr member.PrimaryRole, sr member.SpecialRole) *EmuNodeIntro {
	return &EmuNodeIntro{
		id: node.ShortNodeID(ShortNodeIdOffset + id),
		n:  &emuEndpoint{name: s},
		pr: pr,
		sr: sr,
	}
}

var _ endpoints.Outbound = &emuEndpoint{}

type emuEndpoint struct {
	name endpoints.Name
}

func (p *emuEndpoint) CanAccept(connection endpoints.Inbound) bool {
	return p.name == connection.GetNameAddress()
}

func (p *emuEndpoint) AsByteString() longbits.ByteString {
	return longbits.ByteString(fmt.Sprintf("out:name:%s", p.name))
}

func (p *emuEndpoint) GetIPAddress() endpoints.IPAddress {
	panic("implement me")
}

func (p *emuEndpoint) GetEndpointType() endpoints.NodeEndpointType {
	return endpoints.NameEndpoint
}

func (*emuEndpoint) GetRelayID() node.ShortNodeID {
	return 0
}

func (p *emuEndpoint) GetNameAddress() endpoints.Name {
	return p.name
}

var _ profiles.StaticProfile = &EmuNodeIntro{}
var baseIssuedAtTime = time.Now()

type EmuNodeIntro struct {
	n  endpoints.Outbound
	id node.ShortNodeID
	pr member.PrimaryRole
	sr member.SpecialRole
}

func (c *EmuNodeIntro) GetBriefIntroSignedDigest() cryptkit.SignedDigestHolder {
	dd := longbits.NewBits64(uint64(1000000 + c.id))
	ds := longbits.NewBits64(uint64(1000000+c.id) << 32)

	return cryptkit.NewSignedDigest(
		cryptkit.NewDigest(&dd, "stubHash"),
		cryptkit.NewSignature(&ds, "stubSign")).AsSignedDigestHolder()
}

func (c *EmuNodeIntro) GetIssuedAtPulse() pulse.Number {
	return 0
}

func (c *EmuNodeIntro) GetIssuedAtTime() time.Time {
	return baseIssuedAtTime
}

func (c *EmuNodeIntro) GetPowerLevels() member.PowerSet {
	return member.PowerSet{0, 0, 0, 0xFF}
}

func (c *EmuNodeIntro) GetExtraEndpoints() []endpoints.Outbound {
	return nil
}

func (c *EmuNodeIntro) GetIssuerID() node.ShortNodeID {
	return 0
}

func (c *EmuNodeIntro) GetIssuerSignature() cryptkit.SignatureHolder {
	ds := longbits.NewBits64(uint64(5000000+c.id) << 32)

	return cryptkit.NewSignature(&ds, "stubSign").AsSignatureHolder()
}

func (c *EmuNodeIntro) GetNodePublicKey() cryptkit.SigningKeyHolder {
	v := &longbits.Bits512{}
	longbits.FillBitsWithStaticNoise(uint32(c.id), v[:])
	k := cryptkit.NewSigningKey(v, "stub/stub", cryptkit.PublicAsymmetricKey)
	return &k
}

func (c *EmuNodeIntro) GetStartPower() member.Power {
	return 10
}

func (c *EmuNodeIntro) GetReference() reference.Global {
	return reference.Global{}
}

func (c *EmuNodeIntro) ConvertPowerRequest(request power.Request) member.Power {
	if ok, cl := request.AsCapacityLevel(); ok {
		return member.PowerOf(uint16(cl.DefaultPercent()))
	}
	_, pw := request.AsMemberPower()
	return pw
}

func (c *EmuNodeIntro) GetPrimaryRole() member.PrimaryRole {
	return c.pr
}

func (c *EmuNodeIntro) GetSpecialRoles() member.SpecialRole {
	return c.sr
}

func (*EmuNodeIntro) IsAllowedPower(p member.Power) bool {
	return true
}

func (c *EmuNodeIntro) GetDefaultEndpoint() endpoints.Outbound {
	return c.n
}

func (*EmuNodeIntro) GetPublicKeyStore() cryptkit.PublicKeyStore {
	return nil
}

func (c *EmuNodeIntro) IsAcceptableHost(from endpoints.Inbound) bool {
	addr := c.n.GetNameAddress()
	return addr.Equals(from.GetNameAddress())
}

func (c *EmuNodeIntro) GetStaticNodeID() node.ShortNodeID {
	return c.id
}

func (c *EmuNodeIntro) GetIntroducedNodeID() node.ShortNodeID {
	return c.id
}

func (c *EmuNodeIntro) GetExtension() profiles.StaticProfileExtension {
	return c
}

func (c *EmuNodeIntro) String() string {
	return fmt.Sprintf("{sid:%v, n:%v}", c.id, c.n)
}
