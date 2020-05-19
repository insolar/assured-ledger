// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"reflect"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	node2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	protonode "github.com/insolar/assured-ledger/ledger-core/v2/network/node/internal/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
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
	pulse insolar.PulseNumber
	state node2.NetworkState

	nodeList [ListLength][]node2.NetworkNode
}

func (s *Snapshot) GetPulse() insolar.PulseNumber {
	return s.pulse
}

func (s *Snapshot) Copy() *Snapshot {
	result := &Snapshot{
		pulse: s.pulse,
		state: s.state,
	}
	for i := 0; i < int(ListLength); i++ {
		result.nodeList[i] = make([]node2.NetworkNode, len(s.nodeList[i]))
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
func NewSnapshot(number insolar.PulseNumber, nodes []node2.NetworkNode) *Snapshot {
	return &Snapshot{
		pulse: number,
		// TODO: pass actual state
		state:    node2.NoNetworkState,
		nodeList: splitNodes(nodes),
	}
}

// splitNodes temporary method to create snapshot lists. Will be replaced by special function that will take in count
// previous snapshot and approved claims.
func splitNodes(nodes []node2.NetworkNode) [ListLength][]node2.NetworkNode {
	var result [ListLength][]node2.NetworkNode
	for i := 0; i < int(ListLength); i++ {
		result[i] = make([]node2.NetworkNode, 0)
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

func nodeStateToListType(node node2.NetworkNode) ListType {
	switch node.GetState() {
	case node2.NodeReady:
		if node.GetPower() > 0 {
			return ListWorking
		}
		return ListIdle
	case node2.NodeJoining:
		return ListJoiner
	case node2.NodeUndefined, node2.NodeLeaving:
		return ListLeaving
	}
	// special case for no match
	return ListLength
}

func (s *Snapshot) Encode() ([]byte, error) {
	ss := protonode.Snapshot{}
	ss.PulseNumber = uint32(s.pulse)
	ss.State = uint32(s.state)
	keyProc := platformpolicy.NewKeyProcessor()

	ss.Nodes = make(map[uint32]*protonode.NodeList)
	for t, list := range s.nodeList {
		protoNodeList := make([]*protonode.Node, len(list))
		for i, n := range list {

			exportedKey, err := keyProc.ExportPublicKeyBinary(n.PublicKey())
			if err != nil {
				return nil, errors.Wrap(err, "Failed to export a public key")
			}

			protoNode := &protonode.Node{
				NodeID:         n.ID().AsBytes(),
				NodeShortID:    uint32(n.ShortID()),
				NodeRole:       uint32(n.Role()),
				NodePublicKey:  exportedKey,
				NodeAddress:    n.Address(),
				NodeVersion:    n.Version(),
				NodeLeavingETA: uint32(n.LeavingETA()),
				State:          uint32(n.GetState()),
			}

			protoNodeList[i] = protoNode
		}

		l := &protonode.NodeList{}
		l.List = protoNodeList
		ss.Nodes[uint32(t)] = l
	}

	return ss.Marshal()
}

func (s *Snapshot) Decode(buff []byte) error {
	ss := protonode.Snapshot{}
	err := ss.Unmarshal(buff)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal node")
	}

	keyProc := platformpolicy.NewKeyProcessor()
	s.pulse = insolar.PulseNumber(ss.PulseNumber)
	s.state = node2.NetworkState(ss.State)

	for t, nodes := range ss.Nodes {
		nodeList := make([]node2.NetworkNode, len(nodes.List))
		for i, n := range nodes.List {

			pk, err := keyProc.ImportPublicKeyBinary(n.NodePublicKey)
			if err != nil {
				return errors.Wrap(err, "Failed to ImportPublicKeyBinary")
			}

			ref := reference.GlobalFromBytes(n.NodeID)
			nodeList[i] = newMutableNode(ref, node2.StaticRole(n.NodeRole), pk, node2.NodeState(n.State), n.NodeAddress, n.NodeVersion)
		}
		s.nodeList[t] = nodeList
	}

	return nil
}

func Select(nodes []node2.NetworkNode, typ ListType) []node2.NetworkNode {
	lists := splitNodes(nodes)
	return lists[typ]
}
