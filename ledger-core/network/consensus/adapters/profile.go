// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const DefaultStartPower = member.Power(10)

type StaticProfileExtension struct {
	shortID   node.ShortNodeID
	ref       reference.Global
	signature cryptkit.Signature
}

func NewStaticProfileExtension(networkNode nodeinfo.NetworkNode) *StaticProfileExtension {
	return NewStaticProfileExtensionExt(
		networkNode.GetNodeID(),
		nodeinfo.NodeRef(networkNode),
		nodeinfo.NodeSignedDigest(networkNode).GetSignatureHolder().CopyOfSignature(),
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
	startPower  member.Power
	intro       profiles.StaticProfileExtension
	endpoint    endpoints.Outbound
	store       cryptkit.PublicKeyStore
	keyHolder   cryptkit.SignatureKeyHolder

	signature  cryptkit.SignedDigestHolder
}

func NewStaticProfile(networkNode nodeinfo.NetworkNode, certificate nodeinfo.Certificate, keyProcessor cryptography.KeyProcessor) *StaticProfile {
	return NewStaticProfileExt(networkNode, nodeinfo.NodeAddr(networkNode),
		certificate, keyProcessor, nodeinfo.NodeSignedDigest(networkNode))
}

func NewStaticProfileExt(networkNode nodeinfo.NetworkNode, addr string, certificate nodeinfo.Certificate,
	keyProcessor cryptography.KeyProcessor, signature cryptkit.SignedDigestHolder,
) *StaticProfile {

	specialRole := member.SpecialRoleNone
	if network.IsDiscovery(nodeinfo.NodeRef(networkNode), certificate) {
		specialRole = member.SpecialRoleDiscovery
	}

	publicKey := ECDSAPublicKeyOfNode(networkNode)

	return NewStaticProfileExt2(
		networkNode.GetNodeID(),
		nodeinfo.NodeRole(networkNode),
		specialRole,
		networkNode.GetStatic().GetStartPower(),
		NewStaticProfileExtension(networkNode),
		NewOutbound(addr),
		legacyadapter.NewECDSAPublicKeyStoreFromPK(publicKey),
		legacyadapter.NewECDSASignatureKeyHolder(publicKey, keyProcessor),
		signature,
	)
}

// deprecated // for legacy code only
func ECDSAPublicKeyOfNode(n nodeinfo.NetworkNode) *ecdsa.PublicKey {
	nip := n.GetStatic()
	store := nip.GetPublicKeyStore()
	return store.(*legacyadapter.ECDSAPublicKeyStore).CryptoPublicKey().(*ecdsa.PublicKey)
}


func NewStaticProfileExt2(
	shortID node.ShortNodeID,
	primaryRole member.PrimaryRole,
	specialRole member.SpecialRole,
	startPower  member.Power,
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
		startPower:	 startPower,
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
	// TODO start power level is not passed properly - needs fix
	return DefaultStartPower // sp.startPower
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

func NewOutboundIP(address endpoints.IPAddress) *Outbound {
	return &Outbound{
		name: endpoints.Name(address.String()),
		addr: address,
	}
}

func (p *Outbound) CanAccept(endpoints.Inbound) bool {
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
	return profile
}

func NewNetworkNodeList(profiles []profiles.ActiveNode) []nodeinfo.NetworkNode {
	networkNodes := make([]nodeinfo.NetworkNode, len(profiles))
	for i, p := range profiles {
		networkNodes[i] = NewNetworkNode(p)
	}

	return networkNodes
}
