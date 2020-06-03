// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type StaticProfileExtension struct {
	shortID   node2.ShortNodeID
	ref       reference.Global
	signature cryptkit.SignatureHolder
}

func NewStaticProfileExtension(networkNode node2.NetworkNode) *StaticProfileExtension {
	_, signature := networkNode.(node.MutableNode).GetSignature()

	return newStaticProfileExtension(
		networkNode.ShortID(),
		networkNode.ID(),
		cryptkit.NewSignature(
			longbits.NewBits512FromBytes(signature.Bytes()),
			SHA3512Digest.SignedBy(SECP256r1Sign),
		).AsSignatureHolder(),
	)
}

func newStaticProfileExtension(shortID node2.ShortNodeID, ref reference.Global, signature cryptkit.SignatureHolder) *StaticProfileExtension {
	return &StaticProfileExtension{
		shortID:   shortID,
		ref:       ref,
		signature: signature,
	}
}

func (ni *StaticProfileExtension) GetPowerLevels() member.PowerSet {
	return member.PowerSet{0, 0, 0, 0xff}
}

func (ni *StaticProfileExtension) GetIntroducedNodeID() node2.ShortNodeID {
	return ni.shortID
}

func (ni *StaticProfileExtension) GetExtraEndpoints() []endpoints.Outbound {
	return nil
}

func (ni *StaticProfileExtension) GetIssuedAtPulse() pulse.Number {
	return pulse.NewFirstEphemeralData().PulseNumber
}

func (ni *StaticProfileExtension) GetIssuedAtTime() time.Time {
	return time.Unix(int64(pulse.NewFirstEphemeralData().Timestamp), 0)
}

func (ni *StaticProfileExtension) GetIssuerID() node2.ShortNodeID {
	return ni.shortID
}

func (ni *StaticProfileExtension) GetIssuerSignature() cryptkit.SignatureHolder {
	return ni.signature
}

func (ni *StaticProfileExtension) GetReference() reference.Global {
	return ni.ref
}

type StaticProfile struct {
	shortID     node2.ShortNodeID
	primaryRole member.PrimaryRole
	specialRole member.SpecialRole
	intro       profiles.StaticProfileExtension
	endpoint    endpoints.Outbound
	store       cryptkit.PublicKeyStore
	keyHolder   cryptkit.SignatureKeyHolder

	signature cryptkit.SignedDigestHolder
}

func NewStaticProfile(networkNode node2.NetworkNode, certificate node2.Certificate, keyProcessor cryptography.KeyProcessor) *StaticProfile {

	specialRole := member.SpecialRoleNone
	if network.IsDiscovery(networkNode.ID(), certificate) {
		specialRole = member.SpecialRoleDiscovery
	}

	publicKey := networkNode.PublicKey().(*ecdsa.PublicKey)
	mutableNode := networkNode.(node.MutableNode)
	digest, signature := mutableNode.GetSignature()

	return newStaticProfile(
		networkNode.ShortID(),
		StaticRoleToPrimaryRole(networkNode.Role()),
		specialRole,
		NewStaticProfileExtension(networkNode),
		NewOutbound(networkNode.Address()),
		NewECDSAPublicKeyStore(publicKey),
		NewECDSASignatureKeyHolder(publicKey, keyProcessor),
		cryptkit.NewSignedDigest(
			cryptkit.NewDigest(longbits.NewBits512FromBytes(digest), SHA3512Digest),
			cryptkit.NewSignature(longbits.NewBits512FromBytes(signature.Bytes()), SHA3512Digest.SignedBy(SECP256r1Sign)),
		).AsSignedDigestHolder(),
	)
}

func newStaticProfile(
	shortID node2.ShortNodeID,
	primaryRole member.PrimaryRole,
	specialRole member.SpecialRole,
	intro profiles.StaticProfileExtension,
	endpoint endpoints.Outbound,
	store cryptkit.PublicKeyStore,
	keyHolder cryptkit.SignatureKeyHolder,
	signature cryptkit.SignedDigestHolder,
) *StaticProfile {
	return &StaticProfile{
		shortID:     shortID,
		primaryRole: primaryRole,
		specialRole: specialRole,
		intro:       intro,
		endpoint:    endpoint,
		store:       store,
		keyHolder:   keyHolder,
		signature:   signature,
	}
}

func (sp *StaticProfile) GetPrimaryRole() member.PrimaryRole {
	return sp.primaryRole
}

func (sp *StaticProfile) GetSpecialRoles() member.SpecialRole {
	return sp.specialRole
}

func (sp *StaticProfile) GetExtension() profiles.StaticProfileExtension {
	return sp.intro
}

func (sp *StaticProfile) GetDefaultEndpoint() endpoints.Outbound {
	return sp.endpoint
}

func (sp *StaticProfile) GetPublicKeyStore() cryptkit.PublicKeyStore {
	return sp.store
}

func (sp *StaticProfile) GetNodePublicKey() cryptkit.SignatureKeyHolder {
	return sp.keyHolder
}

func (sp *StaticProfile) GetStartPower() member.Power {
	// TODO: get from certificate
	return 10
}

func (sp *StaticProfile) IsAcceptableHost(from endpoints.Inbound) bool {
	address := sp.endpoint.GetNameAddress()
	return address.Equals(from.GetNameAddress())
}

func (sp *StaticProfile) GetStaticNodeID() node2.ShortNodeID {
	return sp.shortID
}

func (sp *StaticProfile) GetBriefIntroSignedDigest() cryptkit.SignedDigestHolder {
	return sp.signature
}

func (sp *StaticProfile) String() string {
	return fmt.Sprintf("{sid:%d, node:%s}", sp.shortID, sp.intro.GetReference().String())
}

type Outbound struct {
	name endpoints.Name
	addr endpoints.IPAddress
}

func NewOutbound(address string) *Outbound {
	addr, err := endpoints.NewIPAddress(address)
	if err != nil {
		panic(err)
	}

	return &Outbound{
		name: endpoints.Name(address),
		addr: addr,
	}
}

func (p *Outbound) CanAccept(connection endpoints.Inbound) bool {
	return true
}

func (p *Outbound) GetEndpointType() endpoints.NodeEndpointType {
	return endpoints.IPEndpoint
}

func (*Outbound) GetRelayID() node2.ShortNodeID {
	return 0
}

func (p *Outbound) GetNameAddress() endpoints.Name {
	return p.name
}

func (p *Outbound) GetIPAddress() endpoints.IPAddress {
	return p.addr
}

func (p *Outbound) AsByteString() longbits.ByteString {
	return longbits.ByteString(p.addr.String())
}

func NewStaticProfileList(nodes []node2.NetworkNode, certificate node2.Certificate, keyProcessor cryptography.KeyProcessor) []profiles.StaticProfile {
	intros := make([]profiles.StaticProfile, len(nodes))
	for i, n := range nodes {
		intros[i] = NewStaticProfile(n, certificate, keyProcessor)
	}

	profiles.SortStaticProfiles(intros, false)

	return intros
}

func NewNetworkNode(profile profiles.ActiveNode) node2.NetworkNode {
	nip := profile.GetStatic()
	store := nip.GetPublicKeyStore()
	introduction := nip.GetExtension()

	networkNode := node.NewNode(
		introduction.GetReference(),
		PrimaryRoleToStaticRole(nip.GetPrimaryRole()),
		store.(*ECDSAPublicKeyStore).publicKey,
		nip.GetDefaultEndpoint().GetNameAddress().String(),
		"",
	)

	mutableNode := networkNode.(node.MutableNode)

	mutableNode.SetShortID(profile.GetNodeID())
	mutableNode.SetState(node2.Ready)

	mutableNode.SetPower(node2.Power(profile.GetDeclaredPower()))
	if profile.GetOpMode().IsPowerless() {
		mutableNode.SetPower(0)
	}

	sd := nip.GetBriefIntroSignedDigest()
	mutableNode.SetSignature(
		longbits.AsBytes(sd.GetDigestHolder()),
		cryptography.SignatureFromBytes(longbits.AsBytes(sd.GetSignatureHolder())),
	)

	return networkNode
}

func NewNetworkNodeList(profiles []profiles.ActiveNode) []node2.NetworkNode {
	networkNodes := make([]node2.NetworkNode, len(profiles))
	for i, p := range profiles {
		networkNodes[i] = NewNetworkNode(p)
	}

	return networkNodes
}
