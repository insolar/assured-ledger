// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"crypto"
	"sync"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type MutableNode interface {
	nodeinfo.NetworkNode

	SetShortID(shortID node.ShortNodeID)
	SetSignature(cryptkit.SignedDigest)
	SetAddress(address string)
}

func newMutableNode(
	id reference.Global,
	role member.PrimaryRole,
	publicKey crypto.PublicKey,
	state nodeinfo.State,
	address string) *nodeInfo {

	return &nodeInfo{
		NodeID:        id,
		NodeShortID:   GenerateUintShortID(id),
		NodeRole:      role,
		NodePublicKey: publicKey,
		NodeAddress:   address,
		state:         state,
	}
}

func NewTestNode(id reference.Global, role member.PrimaryRole, publicKey crypto.PublicKey, address string) MutableNode {
	return newMutableNode(id, role, publicKey, nodeinfo.Ready, address)
}

type nodeInfo struct {
	NodeID        reference.Global
	NodeShortID   uint32
	NodeRole      member.PrimaryRole
	NodePublicKey crypto.PublicKey
	NodePower     member.Power

	NodeAddress string

	mutex          sync.RWMutex
	digest         cryptkit.SignedDigest
	state          nodeinfo.State
}

func (n *nodeInfo) GetState() nodeinfo.State {
	return n.state
}

func (n *nodeInfo) GetReference() reference.Global {
	return n.NodeID
}

func (n *nodeInfo) GetNodeID() node.ShortNodeID {
	return node.ShortNodeID(atomic.LoadUint32(&n.NodeShortID))
}

func (n *nodeInfo) GetPrimaryRole() member.PrimaryRole {
	return n.NodeRole
}

func (n *nodeInfo) PublicKey() crypto.PublicKey {
	return n.NodePublicKey
}

func (n *nodeInfo) Address() string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.NodeAddress
}

func (n *nodeInfo) GetPower() member.Power {
	return n.NodePower
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
	// ).AsSignedDigestHolder(),

	n.digest = digest
}

func (n *nodeInfo) SetShortID(id node.ShortNodeID) {
	atomic.StoreUint32(&n.NodeShortID, uint32(id))
}

func (n *nodeInfo) SetAddress(address string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.NodeAddress = address
}

func (n *nodeInfo) IsJoiner() bool {
	return n.GetState() == nodeinfo.Joining // TODO use correct impl
}

func (n *nodeInfo) IsPowered() bool {
	return n.GetState() == nodeinfo.Ready && n.GetPower() > 0 // TODO use correct impl
}
