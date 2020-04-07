// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
)

const (
	ShortNodeIDSize = 4
)

type ShortNodeID = apinetwork.ShortNodeID

const AbsentShortNodeID = apinetwork.AbsentShortNodeID

// GlobuleID is the ID of the globe
type GlobuleID uint32

// NodeState is the state of the node
type NodeState uint8

// Power is node power
type Power uint8

//go:generate stringer -type=NodeState
const (
	// NodeUndefined node started but is not connected to network yet
	NodeUndefined NodeState = iota
	// NodeJoining node is in first pulse of discovery bootstrap or is joining to a bootstrapped network
	NodeJoining
	// NodeReady node is connected to network
	NodeReady
	// NodeLeaving node is about to leave network
	NodeLeaving
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar.NetworkNode -o ../testutils/network -s _mock.go -g

type NetworkNode interface {
	// ID is the unique identifier of the node
	ID() Reference
	// ShortID get short ID of node
	ShortID() ShortNodeID
	// Role is the candidate Role for the node
	Role() StaticRole
	// PublicKey is the public key of the node
	PublicKey() crypto.PublicKey
	// Address is the network address of the node
	Address() string
	// GetGlobuleID returns node current globule id
	GetGlobuleID() GlobuleID
	// Version of node software
	Version() string
	// LeavingETA is pulse number, after which node leave
	LeavingETA() PulseNumber
	// GetState get state of the node
	GetState() NodeState
	// GetPower get power of node
	GetPower() Power
}
