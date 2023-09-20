package beat

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// History provides methods for accessing pulses.
type History interface {
	TimeBeat(pulse.Number) (Beat, error)
	LatestTimeBeat() (Beat, error)
	// NodeHistory
}

type NodeHistory interface {
	GetNodeSnapshot(pulse.Number) (NodeSnapshot, error)
	FindAnyLatestNodeSnapshot() NodeSnapshot
	FindLatestNodeSnapshot() NodeSnapshot
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.Appender -o ./ -s _mock.go -g

// Appender provides method for appending pulses to storage.
type Appender interface {
	History
	AddExpectedBeat(Beat) error
	AddCommittedBeat(Beat) error
	EnsureLatestTimeBeat(Beat) error
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.NodeSnapshot -s _mock.go -g

type NodeSnapshot interface {
	GetPulseNumber() pulse.Number
	GetPopulation() census.OnlinePopulation

	// GetLocalNode() profiles.ActiveNode // use GetPopulation().GetLocalProfile() instead

	// FindNodeByRef returns an active node by reference. Returns nil when not found.
	FindNodeByRef(reference.Global) profiles.ActiveNode
	// FindNodeByAddr get active node by addr. Returns nil when not found.
	FindNodeByAddr(address string) profiles.ActiveNode
}

