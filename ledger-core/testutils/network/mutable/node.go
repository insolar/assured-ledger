// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mutable

import (
	"crypto"
	"sync"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Node = *nodeInfo

func NewTestNode(id reference.Global, role member.PrimaryRole, publicKey crypto.PublicKey, address string) Node {
	return newMutableNode(nil, id, role, publicKey, nodeinfo.Ready, address)
}

func newMutableNode(
	static profiles.StaticProfile,
	id reference.Global,
	role member.PrimaryRole,
	publicKey crypto.PublicKey,
	state nodeinfo.State,
	address string) *nodeInfo {

	return &nodeInfo{
		static:        static,
		nodeID:        id,
		nodeShortID:   node.GenerateUintShortID(id),
		nodeRole:      role,
		nodePublicKey: publicKey,
		nodeAddress:   address,
		state:         state,
	}
}

var _ nodeinfo.NetworkNode = &nodeInfo{}
type nodeInfo struct {
	static        profiles.StaticProfile
	nodeID        reference.Global
	nodeShortID   uint32
	nodePublicKey crypto.PublicKey
	nodeRole      member.PrimaryRole
	nodeAddress string
	nodePower     member.Power
	state          nodeinfo.State

	mutex          sync.RWMutex
	digest         cryptkit.SignedDigest
}

func (n *nodeInfo) GetStatic() profiles.StaticProfile {
	if n.static != nil {
		return n.static
	}
	panic(throw.IllegalState())
}

func (n *nodeInfo) GetState() nodeinfo.State {
	return n.state
}

func (n *nodeInfo) GetReference() reference.Global {
	return n.nodeID
}

func (n *nodeInfo) GetNodeID() node.ShortNodeID {
	return node.ShortNodeID(atomic.LoadUint32(&n.nodeShortID))
}

func (n *nodeInfo) GetPrimaryRole() member.PrimaryRole {
	return n.nodeRole
}

func (n *nodeInfo) PublicKey() crypto.PublicKey {
	return n.nodePublicKey
}

func (n *nodeInfo) Address() string {
	return n.nodeAddress
}

func (n *nodeInfo) GetPower() member.Power {
	return n.nodePower
}

func (n *nodeInfo) GetSignature() cryptkit.SignedDigestHolder {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.digest
}

func (n *nodeInfo) SetSignature(digest cryptkit.SignedDigest) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// cryptkit.NewSignedDigest(
	// 	cryptkit.NewDigest(longbits.NewBits512FromBytes(digest), SHA3512Digest),
	// 	cryptkit.NewSignature(longbits.NewBits512FromBytes(signature.Bytes()), SHA3512Digest.SignedBy(SECP256r1Sign)),
	// )

	n.digest = digest
}

func (n *nodeInfo) SetShortID(id node.ShortNodeID) {
	atomic.StoreUint32(&n.nodeShortID, uint32(id))
}

func (n *nodeInfo) IsJoiner() bool {
	return n.GetState() == nodeinfo.Joining
}

func (n *nodeInfo) IsPowered() bool {
	return n.GetState() == nodeinfo.Ready && n.GetPower() > 0
}
