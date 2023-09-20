package network

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type StatusReply struct {
	NetworkState    State
	LocalRef        reference.Holder
	LocalRole       member.PrimaryRole
	LocalNode 		nodeinfo.NetworkNode
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

