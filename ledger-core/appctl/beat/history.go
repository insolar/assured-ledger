// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package beat

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.History -o ./ -s _mock.go -g

// History provides methods for accessing pulses.
type History interface {
	TimeBeat(pulse.Number) (Beat, error)
	LatestTimeBeat() (Beat, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.Appender -o ./ -s _mock.go -g

// Appender provides method for appending pulses to storage.
type Appender interface {
	History
	AddCommittedBeat(Beat) error
	EnsureLatestTimeBeat(Beat) error
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeSnapshot -s _mock.go -g

type NodeSnapshot interface {
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

