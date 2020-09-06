// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mutable

import (
	"crypto"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewTestNode(id reference.Global, role member.PrimaryRole, address string) *Node {
	return newMutableNode(id, role, nil, address)
}

func newMutableNode(
	id reference.Global,
	role member.PrimaryRole,
	publicKey crypto.PublicKey,
	address string) *Node {

	n := &Node{
		nodeID:        id,
		nodeShortID:   node.GenerateShortID(id),
		nodeRole:      role,
		nodePublicKey: publicKey,
	}

	if address != "" {
		var err error
		n.nodeAddress, err = endpoints.NewIPAddress(address)
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
	digest        cryptkit.SignedDigest
	pks           cryptkit.PublicKeyStore
	npk           cryptkit.SigningKeyHolder
}

func (n *Node) LocalNodeProfile() {}

func (n *Node) GetSignatureVerifier() cryptkit.SignatureVerifier {
	panic(throw.NotImplemented())
}

func (n *Node) GetOpMode() member.OpMode {
	return member.ModeNormal
}

func (n *Node) GetIndex() member.Index {
	panic(throw.NotImplemented())
}

func (n *Node) IsVoter() bool {
	return true
}

func (n *Node) IsStateful() bool {
	return true
}

func (n *Node) CanIntroduceJoiner() bool {
	return true
}

func (n *Node) HasFullProfile() bool {
	return true
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
	return pulse.MinTimePulse
}

func (n *Node) GetIssuedAtTime() time.Time {
	return time.Now()
}

func (n *Node) GetIssuerID() node.ShortNodeID {
	return n.nodeShortID
}

func (n *Node) GetIssuerSignature() cryptkit.SignatureHolder {
	return n.digest.GetSignatureHolder()
}

func (n *Node) GetDefaultEndpoint() endpoints.Outbound {
	return n
}

func (n *Node) GetPublicKeyStore() cryptkit.PublicKeyStore {
	return n.pks
}

func (n *Node) SetPublicKeyStore(pks cryptkit.PublicKeyStore) {
	n.pks = pks
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

func (n *Node) GetNodePublicKey() cryptkit.SigningKeyHolder {
	return n.npk
}

func (n *Node) SetNodePublicKey(npk cryptkit.SigningKeyHolder) {
	n.npk = npk
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

func (n *Node) GetNodeID() node.ShortNodeID {
	return n.nodeShortID
}

func (n *Node) GetPrimaryRole() member.PrimaryRole {
	return n.nodeRole
}

func (n *Node) GetDeclaredPower() member.Power {
	return n.nodePower
}

func (n *Node) SetSignature(digest cryptkit.SignedDigest) {
	n.digest = digest
}

func (n *Node) SetShortID(id node.ShortNodeID) {
	n.nodeShortID = id
}

func (n *Node) IsJoiner() bool {
	return false
}

func (n *Node) IsPowered() bool {
	return n.GetDeclaredPower() > 0
}
