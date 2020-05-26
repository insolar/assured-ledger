// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

const (
	ShortNodeIDSize = 4
)

// ShortNodeID is the shortened ID of node that is unique inside the globe
type ShortNodeID uint32 // ZERO is RESERVED

const AbsentShortNodeID ShortNodeID = 0

func (v ShortNodeID) IsAbsent() bool { return v == AbsentShortNodeID }

func (v ShortNodeID) Equal(other ShortNodeID) bool { return v == other }

// GlobuleID is the ID of the globe
type GlobuleID uint32

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

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/node.NetworkNode -o ../../testutils/network -s _mock.go -g

type NetworkNode interface {
	// ID is the unique identifier of the node
	ID() reference.Global
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
	LeavingETA() pulse.Number
	// GetState get state of the node
	GetState() State
	// GetPower get power of node
	GetPower() Power
}
