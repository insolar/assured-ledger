// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeset

import (
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type Snapshot struct {
	pulse pulse.Number
	state network.State

	workingNodes []nodeinfo.NetworkNode
	activeNodes  []nodeinfo.NetworkNode
}

func (s *Snapshot) GetPulse() pulse.Number {
	return s.pulse
}

// NewSnapshot create new snapshot for pulse.
func NewSnapshot(number pulse.Number, nodes []nodeinfo.NetworkNode) *Snapshot {
	return &Snapshot{
		pulse: number,
		// TODO: pass actual state
		state:        network.NoNetworkState,
		activeNodes:  nodes,
		workingNodes: SelectWorking(nodes),
	}
}

func SelectWorking(nodes []nodeinfo.NetworkNode) []nodeinfo.NetworkNode {
	result := make([]nodeinfo.NetworkNode, 0, len(nodes))
	for _, nd := range nodes {
		if isWorkingNode(nd) {
			result = append(result, nd)
		}
	}
	return result
}

func isWorkingNode(nd nodeinfo.NetworkNode) bool {
	return nd.IsPowered()
}
