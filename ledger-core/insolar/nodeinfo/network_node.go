// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeinfo

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// Power is node power
type Power uint8

//go:generate stringer -type=State
type State uint8

const (
	// Undefined node started but is not connected to network yet
	Undefined State = iota
	// Joining node is in first pulse of discovery bootstrap or is joining to a bootstrapped network
	Joining
	// Ready node is connected to network
	Ready
	// Leaving node is about to leave network
	Leaving
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo.NetworkNode -o ../../testutils/network -s _mock.go -g

type NetworkNode interface {
	// ID is the unique identifier of the node
	GetReference() reference.Global
	// ShortID get short ID of node
	GetNodeID() node.ShortNodeID

	GetPrimaryRole() member.PrimaryRole
	// PublicKey is the public key of the node
	PublicKey() crypto.PublicKey
	// Address is the network address of the node
	Address() string
	// GetState get state of the node
	GetState() State
	// GetPower get power of node
	GetPower() Power

	GetSignature() ([]byte, cryptography.Signature)
}
