// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type ListType int

const (
	ListWorking ListType = iota
	ListIdle
	ListLeaving
	ListSuspected
	ListJoiner

	ListLength
)

type Snapshot struct {
	pulse pulse.Number
	state node.NetworkState

	nodeList [ListLength][]node.NetworkNode
}

func (s *Snapshot) GetPulse() pulse.Number {
	return s.pulse
}

func (s *Snapshot) Copy() *Snapshot {
	result := &Snapshot{
		pulse: s.pulse,
		state: s.state,
	}
	for i := 0; i < int(ListLength); i++ {
		result.nodeList[i] = make([]node.NetworkNode, len(s.nodeList[i]))
		copy(result.nodeList[i], s.nodeList[i])
	}
	return result
}

func (s *Snapshot) Equal(s2 *Snapshot) bool {
	if s.pulse != s2.pulse || s.state != s2.state {
		return false
	}

	for t, list := range s.nodeList {
		if len(list) != len(s2.nodeList[t]) {
			return false
		}
		for i, n := range list {
			n2 := s2.nodeList[t][i]
			if !reflect.DeepEqual(n, n2) {
				return false
			}
		}
	}
	return true
}

// NewSnapshot create new snapshot for pulse.
func NewSnapshot(number pulse.Number, nodes []node.NetworkNode) *Snapshot {
	return &Snapshot{
		pulse: number,
		// TODO: pass actual state
		state:    node.NoNetworkState,
		nodeList: splitNodes(nodes),
	}
}

// splitNodes temporary method to create snapshot lists. Will be replaced by special function that will take in count
// previous snapshot and approved claims.
func splitNodes(nodes []node.NetworkNode) [ListLength][]node.NetworkNode {
	var result [ListLength][]node.NetworkNode
	for i := 0; i < int(ListLength); i++ {
		result[i] = make([]node.NetworkNode, 0)
	}
	for _, node := range nodes {
		listType := nodeStateToListType(node)
		if listType == ListLength {
			continue
		}
		result[listType] = append(result[listType], node)
	}
	return result
}

func nodeStateToListType(nd node.NetworkNode) ListType {
	switch nd.GetState() {
	case node.Ready:
		if nd.GetPower() > 0 {
			return ListWorking
		}
		return ListIdle
	case node.Joining:
		return ListJoiner
	case node.Undefined, node.Leaving:
		return ListLeaving
	}
	// special case for no match
	return ListLength
}

func Select(nodes []node.NetworkNode, typ ListType) []node.NetworkNode {
	lists := splitNodes(nodes)
	return lists[typ]
}
