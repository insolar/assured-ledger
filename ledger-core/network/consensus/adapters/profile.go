// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"crypto"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type StaticProfileExtension struct {
	shortID   node.ShortNodeID
	ref       reference.Global
	signature cryptkit.Signature
}

func NewStaticProfileExtension(networkNode nodeinfo.NetworkNode) *StaticProfileExtension {
	return NewStaticProfileExtensionExt(
		networkNode.GetNodeID(),
		networkNode.GetReference(),
		networkNode.GetSignature().GetSignatureHolder().CopyOfSignature(),
	)
}

func NewStaticProfileExtensionExt(shortID node.ShortNodeID, ref reference.Global, signature cryptkit.Signature) *StaticProfileExtension {
	return &StaticProfileExtension{
		shortID:   shortID,
		ref:       ref,
		signature: signature,
	}
}

func (ni *StaticProfileExtension) GetPowerLevels() member.PowerSet {
	return member.PowerSet{0, 0, 0, 0xff}
}

func (ni *StaticProfileExtension) GetIntroducedNodeID() node.ShortNodeID {
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

func (ni *StaticProfileExtension) GetIssuerID() node.ShortNodeID {
	return ni.shortID
}

func (ni *StaticProfileExtension) GetIssuerSignature() cryptkit.SignatureHolder {
	return ni.signature
}

func (ni *StaticProfileExtension) GetReference() reference.Global {
	return ni.ref
}

type StaticProfile struct {
	shortID     node.ShortNodeID
	primaryRole member.PrimaryRole
	specialRole member.SpecialRole
	intro       profiles.StaticProfileExtension
	endpoint    endpoints.Outbound
	store       cryptkit.PublicKeyStore
	keyHolder   cryptkit.SignatureKeyHolder

	signature cryptkit.SignedDigestHolder
}

func NewStaticProfile(networkNode nodeinfo.NetworkNode, certificate nodeinfo.Certificate, keyProcessor cryptography.KeyProcessor) *StaticProfile {
	return NewStaticProfileExt(networkNode, networkNode.Address(), certificate, keyProcessor, networkNode.GetSignature())
}

func NewStaticProfileExt(networkNode nodeinfo.NetworkNode, addr string, certificate nodeinfo.Certificate,
	keyProcessor cryptography.KeyProcessor, signature cryptkit.SignedDigestHolder,
) *StaticProfile {

	specialRole := member.SpecialRoleNone
	if network.IsDiscovery(networkNode.GetReference(), certificate) {
		specialRole = member.SpecialRoleDiscovery
	}

	publicKey := networkNode.PublicKey().(*ecdsa.PublicKey)

	return NewStaticProfileExt2(
		networkNode.GetNodeID(),
		networkNode.GetPrimaryRole(),
		specialRole,
		NewStaticProfileExtension(networkNode),
		NewOutbound(addr),
		NewECDSAPublicKeyStore(publicKey),
		NewECDSASignatureKeyHolder(publicKey, keyProcessor),
		signature,
	)
}

func ECDSAPublicKeyAsPublicKeyStore(pk crypto.PublicKey) cryptkit.PublicKeyStore {
	if pk == nil {
		return nil
	}
	return NewECDSAPublicKeyStore(pk.(*ecdsa.PublicKey))
}

func NewStaticProfileExt2(
	shortID node.ShortNodeID,
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

func (sp *StaticProfile) GetStaticNodeID() node.ShortNodeID {
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

func NewOutboundNoPort(address string) *Outbound {
	addr, err := endpoints.NewIPAddressZeroPort(address)
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

func (*Outbound) GetRelayID() node.ShortNodeID {
	return 0
}

func (p *Outbound) GetNameAddress() endpoints.Name {
	return p.name
}

func (p *Outbound) GetIPAddress() endpoints.IPAddress {
	return p.addr
}

func NewStaticProfileList(nodes []nodeinfo.NetworkNode, certificate nodeinfo.Certificate, keyProcessor cryptography.KeyProcessor) []profiles.StaticProfile {
	intros := make([]profiles.StaticProfile, len(nodes))
	for i, n := range nodes {
		intros[i] = NewStaticProfile(n, certificate, keyProcessor)
	}

	profiles.SortStaticProfiles(intros, false)

	return intros
}

func NewNetworkNode(profile profiles.ActiveNode) nodeinfo.NetworkNode {
	if profile == nil {
		panic(throw.IllegalValue())
	}
	return profileNode{profile}
}

func NewNetworkNodeList(profiles []profiles.ActiveNode) []nodeinfo.NetworkNode {
	networkNodes := make([]nodeinfo.NetworkNode, len(profiles))
	for i, p := range profiles {
		networkNodes[i] = NewNetworkNode(p)
	}

	return networkNodes
}
