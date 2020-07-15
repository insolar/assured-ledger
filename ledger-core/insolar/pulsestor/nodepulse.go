// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor.PulseDistributor -s _mock.go -g

// PulseDistributor is interface for pulse distribution.
type PulseDistributor interface {
	// Distribute distributes a pulse across the network.
	Distribute(context.Context, Pulse)
}

type StatusReply struct {
	NetworkState    node.NetworkState
	Origin          node.NetworkNode
	ActiveListSize  int
	WorkingListSize int
	// Nodes from active list
	Nodes []node.NetworkNode
	// Pulse from network pulse storage
	Pulse     Pulse
	Version   string
	Timestamp time.Time
	// node start timestamp for uptime duration
	StartTime time.Time
}

type NetworkStatus interface {
	GetNetworkStatus() StatusReply
}

