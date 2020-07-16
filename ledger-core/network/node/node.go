// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"crypto"
	"hash/crc32"
	"sync"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type MutableNode interface {
	nodeinfo.NetworkNode

	SetShortID(shortID node.ShortNodeID)
	SetState(state nodeinfo.State)
	GetSignature() ([]byte, cryptography.Signature)
	SetSignature(digest []byte, signature cryptography.Signature)
	ChangeState()
	SetLeavingETA(number pulse.Number)
	SetVersion(version string)
	SetPower(power nodeinfo.Power)
	SetAddress(address string)
}

// GenerateUintShortID generate short ID for node without checking collisions
func GenerateUintShortID(ref reference.Global) uint32 {
	return crc32.ChecksumIEEE(ref.AsBytes())
}

type nodeInfo struct {
	NodeID        reference.Global
	NodeShortID   uint32
	NodeRole      member.PrimaryRole
	NodePublicKey crypto.PublicKey
	NodePower     uint32

	NodeAddress string

	mutex          sync.RWMutex
	digest         []byte
	signature      cryptography.Signature
	NodeVersion    string
	NodeLeavingETA uint32
	state          uint32
}

func (n *nodeInfo) SetVersion(version string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.NodeVersion = version
}

func (n *nodeInfo) SetState(state nodeinfo.State) {
	atomic.StoreUint32(&n.state, uint32(state))
}

func (n *nodeInfo) GetState() nodeinfo.State {
	return nodeinfo.State(atomic.LoadUint32(&n.state))
}

func (n *nodeInfo) ChangeState() {
	// we don't expect concurrent changes, so do not CAS
	currentState := atomic.LoadUint32(&n.state)
	if currentState >= uint32(nodeinfo.Ready) {
		return
	}
	atomic.StoreUint32(&n.state, currentState+1)
}

func newMutableNode(
	id reference.Global,
	role member.PrimaryRole,
	publicKey crypto.PublicKey,
	state nodeinfo.State,
	address, version string) MutableNode {

	return &nodeInfo{
		NodeID:        id,
		NodeShortID:   GenerateUintShortID(id),
		NodeRole:      role,
		NodePublicKey: publicKey,
		NodeAddress:   address,
		NodeVersion:   version,
		state:         uint32(state),
	}
}

func NewNode(
	id reference.Global,
	role member.PrimaryRole,
	publicKey crypto.PublicKey,
	address, version string) nodeinfo.NetworkNode {
	return newMutableNode(id, role, publicKey, nodeinfo.Ready, address, version)
}

func (n *nodeInfo) ID() reference.Global {
	return n.NodeID
}

func (n *nodeInfo) ShortID() node.ShortNodeID {
	return node.ShortNodeID(atomic.LoadUint32(&n.NodeShortID))
}

func (n *nodeInfo) Role() member.PrimaryRole {
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

func (n *nodeInfo) GetGlobuleID() nodeinfo.GlobuleID {
	return 0
}

func (n *nodeInfo) GetPower() nodeinfo.Power {
	return nodeinfo.Power(atomic.LoadUint32(&n.NodePower))
}

func (n *nodeInfo) SetPower(power nodeinfo.Power) {
	atomic.StoreUint32(&n.NodePower, uint32(power))
}

func (n *nodeInfo) Version() string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.NodeVersion
}

func (n *nodeInfo) GetSignature() ([]byte, cryptography.Signature) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.digest, n.signature
}

func (n *nodeInfo) SetSignature(digest []byte, signature cryptography.Signature) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.signature = signature
	n.digest = digest
}

func (n *nodeInfo) SetShortID(id node.ShortNodeID) {
	atomic.StoreUint32(&n.NodeShortID, uint32(id))
}

func (n *nodeInfo) LeavingETA() pulse.Number {
	return pulse.Number(atomic.LoadUint32(&n.NodeLeavingETA))
}

func (n *nodeInfo) SetLeavingETA(number pulse.Number) {
	n.SetState(nodeinfo.Leaving)
	atomic.StoreUint32(&n.NodeLeavingETA, uint32(number))
}

func (n *nodeInfo) SetAddress(address string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.NodeAddress = address
}
