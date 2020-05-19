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

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	node2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type MutableNode interface {
	node2.NetworkNode

	SetShortID(shortID node2.ShortNodeID)
	SetState(state node2.NodeState)
	GetSignature() ([]byte, cryptography.Signature)
	SetSignature(digest []byte, signature cryptography.Signature)
	ChangeState()
	SetLeavingETA(number insolar.PulseNumber)
	SetVersion(version string)
	SetPower(power node2.Power)
	SetAddress(address string)
}

// GenerateUintShortID generate short ID for node without checking collisions
func GenerateUintShortID(ref reference.Global) uint32 {
	return crc32.ChecksumIEEE(ref.AsBytes())
}

type node struct {
	NodeID        reference.Global
	NodeShortID   uint32
	NodeRole      node2.StaticRole
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

func (n *node) SetVersion(version string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.NodeVersion = version
}

func (n *node) SetState(state node2.NodeState) {
	atomic.StoreUint32(&n.state, uint32(state))
}

func (n *node) GetState() node2.NodeState {
	return node2.NodeState(atomic.LoadUint32(&n.state))
}

func (n *node) ChangeState() {
	// we don't expect concurrent changes, so do not CAS
	currentState := atomic.LoadUint32(&n.state)
	if currentState >= uint32(node2.NodeReady) {
		return
	}
	atomic.StoreUint32(&n.state, currentState+1)
}

func newMutableNode(
	id reference.Global,
	role node2.StaticRole,
	publicKey crypto.PublicKey,
	state node2.NodeState,
	address, version string) MutableNode {

	return &node{
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
	role node2.StaticRole,
	publicKey crypto.PublicKey,
	address, version string) node2.NetworkNode {
	return newMutableNode(id, role, publicKey, node2.NodeReady, address, version)
}

func (n *node) ID() reference.Global {
	return n.NodeID
}

func (n *node) ShortID() node2.ShortNodeID {
	return node2.ShortNodeID(atomic.LoadUint32(&n.NodeShortID))
}

func (n *node) Role() node2.StaticRole {
	return n.NodeRole
}

func (n *node) PublicKey() crypto.PublicKey {
	return n.NodePublicKey
}

func (n *node) Address() string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.NodeAddress
}

func (n *node) GetGlobuleID() node2.GlobuleID {
	return 0
}

func (n *node) GetPower() node2.Power {
	return node2.Power(atomic.LoadUint32(&n.NodePower))
}

func (n *node) SetPower(power node2.Power) {
	atomic.StoreUint32(&n.NodePower, uint32(power))
}

func (n *node) Version() string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.NodeVersion
}

func (n *node) GetSignature() ([]byte, cryptography.Signature) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	return n.digest, n.signature
}

func (n *node) SetSignature(digest []byte, signature cryptography.Signature) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.signature = signature
	n.digest = digest
}

func (n *node) SetShortID(id node2.ShortNodeID) {
	atomic.StoreUint32(&n.NodeShortID, uint32(id))
}

func (n *node) LeavingETA() insolar.PulseNumber {
	return insolar.PulseNumber(atomic.LoadUint32(&n.NodeLeavingETA))
}

func (n *node) SetLeavingETA(number insolar.PulseNumber) {
	n.SetState(node2.NodeLeaving)
	atomic.StoreUint32(&n.NodeLeavingETA, uint32(number))
}

func (n *node) SetAddress(address string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.NodeAddress = address
}
