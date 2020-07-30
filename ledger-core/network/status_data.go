// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package network

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type StatusReply struct {
	NetworkState    State
	Origin          nodeinfo.NetworkNode
	ActiveListSize  int
	WorkingListSize int
	// Nodes from active list
	Nodes []nodeinfo.NetworkNode
	Version   string
	Timestamp time.Time
	// node start timestamp for uptime duration
	StartTime    time.Time

	PulseNumber  pulse.Number
}

type Status interface {
	GetNetworkStatus() StatusReply
}

