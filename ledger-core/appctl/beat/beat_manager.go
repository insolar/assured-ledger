// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package beat

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeNetwork -s _mock.go -g

type NodeNetwork interface {
	// GetLocalNodeReference returns a node reference for this/local node. Safe to call at any time. Immutable.
	GetLocalNodeReference() reference.Holder
	// GetLocalNodeRole returns a role for this/local node. Safe to call at any time. Immutable.
	GetLocalNodeRole() member.PrimaryRole

	// GetAccessor get accessor to the internal snapshot for the current pulse. Panics on unknown pulse.
	GetAccessor(pulse.Number) NodeAccessor
	// GetLatestAccessor returns the latest set of nodes. Will return nil when nothing was added yet.
	GetLatestAccessor() NodeAccessor
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeKeeper -s _mock.go -g

type NodeKeeper interface {
	NodeNetwork

	SetExpectedPopulation(context.Context, Beat)
	AddActivePopulation(context.Context, Beat)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeAccessor -s _mock.go -g

// NodeAccessor is interface that provides read access to nodekeeper internal snapshot
type NodeAccessor interface {
	GetPulseNumber() pulse.Number
	GetLocalNode() profiles.ActiveNode
	GetPopulation() census.OnlinePopulation

	// GetPoweredNode get working node by its reference. Returns nil if node is not found or is not working.
	GetPoweredNode(reference.Global) profiles.ActiveNode
	// GetOnlineNode returns active node.
	GetOnlineNode(reference.Global) profiles.ActiveNode
	// GetOnlineNodes returns unsorted list of all active nodes.
	GetOnlineNodes() []profiles.ActiveNode
	// GetOnlineNodeByAddr get active node by addr. Returns nil if node is not found.
	GetOnlineNodeByAddr(address string) profiles.ActiveNode
}

