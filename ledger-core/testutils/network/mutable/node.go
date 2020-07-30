// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mutable

import (
	"crypto"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewTestNode(id reference.Global, role member.PrimaryRole, address string) *Node {
	return newMutableNode(id, role, nil, nodeinfo.Ready, address)
}

func newMutableNode(
	id reference.Global,
	role member.PrimaryRole,
	publicKey crypto.PublicKey,
	state nodeinfo.State,
	address string) *Node {

	n := &Node{
		nodeID:        id,
		nodeShortID:   node.GenerateUintShortID(id),
		nodeRole:      role,
		nodePublicKey: publicKey,
		state:         state,
	}

	if address != "" {
		var err error
		n.nodeAddress, err = endpoints.NewIPAddressZeroPort(address)
		if err != nil {
			panic(err)
		}
	}
	return n
}

var _ nodeinfo.NetworkNode = &Node{}
type Node struct {
	nodeID        reference.Global
	nodeShortID   node.ShortNodeID
	nodePublicKey crypto.PublicKey
	nodeRole      member.PrimaryRole
	nodeAddress   endpoints.IPAddress
	nodePower     member.Power
	state         nodeinfo.State
	digest        cryptkit.SignedDigest
}

func (n *Node) GetReference() reference.Global {
	return n.nodeID
}

func (n *Node) GetEndpointType() endpoints.NodeEndpointType {
	return endpoints.IPEndpoint
}

func (n *Node) GetRelayID() node.ShortNodeID {
	return 0
}

func (n *Node) GetNameAddress() endpoints.Name {
	return endpoints.Name(n.nodeAddress.String())
}

func (n *Node) GetIPAddress() endpoints.IPAddress {
	return n.nodeAddress
}

func (n *Node) CanAccept(endpoints.Inbound) bool {
	return true
}

func (n *Node) GetIntroducedNodeID() node.ShortNodeID {
	return n.nodeShortID
}

func (n *Node) GetPowerLevels() member.PowerSet {
	return member.PowerSet{0, 0, 0, n.nodePower}
}

func (n *Node) GetExtraEndpoints() []endpoints.Outbound {
	return nil
}

func (n *Node) GetIssuedAtPulse() pulse.Number {
	panic(throw.NotImplemented())
}

func (n *Node) GetIssuedAtTime() time.Time {
	panic(throw.NotImplemented())
}

func (n *Node) GetIssuerID() node.ShortNodeID {
	panic(throw.NotImplemented())
}

func (n *Node) GetIssuerSignature() cryptkit.SignatureHolder {
	panic(throw.NotImplemented())
}

func (n *Node) GetDefaultEndpoint() endpoints.Outbound {
	return n
}

func (n *Node) GetPublicKeyStore() cryptkit.PublicKeyStore {
	panic("implement me")
}

func (n *Node) IsAcceptableHost(from endpoints.Inbound) bool {
	return true
}

func (n *Node) GetStaticNodeID() node.ShortNodeID {
	return n.nodeShortID
}

func (n *Node) GetSpecialRoles() member.SpecialRole {
	return 0
}

func (n *Node) GetNodePublicKey() cryptkit.SignatureKeyHolder {
	panic("implement me")
}

func (n *Node) GetStartPower() member.Power {
	return n.nodePower
}

func (n *Node) GetBriefIntroSignedDigest() cryptkit.SignedDigestHolder {
	return n.digest
}

func (n *Node) GetExtension() profiles.StaticProfileExtension {
	return n
}

func (n *Node) GetStatic() profiles.StaticProfile {
	return n
}

func (n *Node) GetState() nodeinfo.State {
	return n.state
}

func (n *Node) GetNodeID() node.ShortNodeID {
	return n.nodeShortID
}

func (n *Node) GetPrimaryRole() member.PrimaryRole {
	return n.nodeRole
}

func (n *Node) PublicKey() crypto.PublicKey {
	return n.nodePublicKey
}

func (n *Node) GetDeclaredPower() member.Power {
	return n.nodePower
}

func (n *Node) GetSignature() cryptkit.SignedDigestHolder {
	return n.digest
}

func (n *Node) SetSignature(digest cryptkit.SignedDigest) {
	// cryptkit.NewSignedDigest(
	// 	cryptkit.NewDigest(longbits.NewBits512FromBytes(digest), SHA3512Digest),
	// 	cryptkit.NewSignature(longbits.NewBits512FromBytes(signature.Bytes()), SHA3512Digest.SignedBy(SECP256r1Sign)),
	// )

	n.digest = digest
}

func (n *Node) SetShortID(id node.ShortNodeID) {
	n.nodeShortID = id
}

func (n *Node) IsJoiner() bool {
	return n.GetState() == nodeinfo.Joining
}

func (n *Node) IsPowered() bool {
	return n.GetState() == nodeinfo.Ready && n.GetDeclaredPower() > 0
}
