// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type StatusReply struct {
	NetworkState    NetworkState
	Origin          NetworkNode
	ActiveListSize  int
	WorkingListSize int
	// Nodes from active list
	Nodes []NetworkNode
	// Pulse from network pulse storage
	Pulse     pulsestor.Pulse
	Version   string
	Timestamp time.Time
	// node start timestamp for uptime duration
	StartTime time.Time
}

type NetworkStatus interface {
	GetNetworkStatus() StatusReply
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/node.Leaver -o ../../testutils -s _mock.go -g

type Leaver interface {
	// Leave notify other nodes that this node want to leave and doesn't want to receive new tasks
	Leave(ctx context.Context, ETA pulse.Number)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/node.CertificateGetter -o ../../testutils -s _mock.go -g

type CertificateGetter interface {
	// GetCert registers reference and returns new certificate for it
	GetCert(context.Context, reference.Global) (Certificate, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/node.PulseDistributor -o ../../testutils -s _mock.go -g

// PulseDistributor is interface for pulse distribution.
type PulseDistributor interface {
	// Distribute distributes a pulse across the network.
	Distribute(context.Context, pulsestor.Pulse)
}

// NetworkState type for bootstrapping process
type NetworkState int

//go:generate stringer -type=NetworkState
const (
	// NoNetworkState state means that nodes doesn`t match majority_rule
	NoNetworkState NetworkState = iota
	JoinerBootstrap
	DiscoveryBootstrap
	WaitConsensus
	WaitMajority
	WaitMinRoles
	WaitPulsar
	CompleteNetworkState
)
