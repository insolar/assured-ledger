// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"sort"

	node2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type Accessor struct {
	snapshot  *Snapshot
	refIndex  map[reference.Global]node2.NetworkNode
	sidIndex  map[node2.ShortNodeID]node2.NetworkNode
	addrIndex map[string]node2.NetworkNode
	// should be removed in future
	active []node2.NetworkNode
}

func (a *Accessor) GetActiveNodeByShortID(shortID node2.ShortNodeID) node2.NetworkNode {
	return a.sidIndex[shortID]
}

func (a *Accessor) GetActiveNodeByAddr(address string) node2.NetworkNode {
	return a.addrIndex[address]
}

func (a *Accessor) GetActiveNodes() []node2.NetworkNode {
	result := make([]node2.NetworkNode, len(a.active))
	copy(result, a.active)
	return result
}

func (a *Accessor) GetActiveNode(ref reference.Global) node2.NetworkNode {
	return a.refIndex[ref]
}

func (a *Accessor) GetWorkingNode(ref reference.Global) node2.NetworkNode {
	node := a.GetActiveNode(ref)
	if node == nil || node.GetPower() == 0 {
		return nil
	}
	return node
}

func (a *Accessor) GetWorkingNodes() []node2.NetworkNode {
	workingList := a.snapshot.nodeList[ListWorking]
	result := make([]node2.NetworkNode, len(workingList))
	copy(result, workingList)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID().Compare(result[j].ID()) < 0
	})
	return result
}

func GetSnapshotActiveNodes(snapshot *Snapshot) []node2.NetworkNode {
	joining := snapshot.nodeList[ListJoiner]
	idle := snapshot.nodeList[ListIdle]
	working := snapshot.nodeList[ListWorking]
	leaving := snapshot.nodeList[ListLeaving]

	joinersCount := len(joining)
	idlersCount := len(idle)
	workingCount := len(working)
	leavingCount := len(leaving)

	result := make([]node2.NetworkNode, joinersCount+idlersCount+workingCount+leavingCount)

	copy(result[:joinersCount], joining)
	copy(result[joinersCount:joinersCount+idlersCount], idle)
	copy(result[joinersCount+idlersCount:joinersCount+idlersCount+workingCount], working)
	copy(result[joinersCount+idlersCount+workingCount:], leaving)

	return result
}

func (a *Accessor) addToIndex(node node2.NetworkNode) {
	a.refIndex[node.ID()] = node
	a.sidIndex[node.ShortID()] = node
	a.addrIndex[node.Address()] = node

	if node.GetPower() == 0 {
		return
	}
}

func NewAccessor(snapshot *Snapshot) *Accessor {
	result := &Accessor{
		snapshot:  snapshot,
		refIndex:  make(map[reference.Global]node2.NetworkNode),
		sidIndex:  make(map[node2.ShortNodeID]node2.NetworkNode),
		addrIndex: make(map[string]node2.NetworkNode),
	}
	result.active = GetSnapshotActiveNodes(snapshot)
	for _, node := range result.active {
		result.addToIndex(node)
	}
	return result
}
