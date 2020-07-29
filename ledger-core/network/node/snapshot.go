// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type Snapshot struct {
	pulse pulse.Number
	state nodeinfo.NetworkState

	workingNodes []nodeinfo.NetworkNode
	activeNodes  []nodeinfo.NetworkNode
}

func (s *Snapshot) GetPulse() pulse.Number {
	return s.pulse
}

func (s *Snapshot) Equal(s2 *Snapshot) bool {
	if s.pulse != s2.pulse || s.state != s2.state {
		return false
	}
	return reflect.DeepEqual(s.workingNodes, s2.workingNodes)
}

// NewSnapshot create new snapshot for pulse.
func NewSnapshot(number pulse.Number, nodes []nodeinfo.NetworkNode) *Snapshot {
	return &Snapshot{
		pulse: number,
		// TODO: pass actual state
		state:        nodeinfo.NoNetworkState,
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
	return nd.GetState() == nodeinfo.Ready && nd.GetPower() > 0
}
