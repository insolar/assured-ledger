// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"sort"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Accessor struct {
	snapshot  *Snapshot
	refIndex  map[reference.Global]nodeinfo.NetworkNode
	sidIndex  map[node2.ShortNodeID]nodeinfo.NetworkNode
	addrIndex map[string]nodeinfo.NetworkNode
	// should be removed in future
	active []nodeinfo.NetworkNode
}

func (a *Accessor) GetActiveNodeByShortID(shortID node2.ShortNodeID) nodeinfo.NetworkNode {
	return a.sidIndex[shortID]
}

func (a *Accessor) GetActiveNodeByAddr(address string) nodeinfo.NetworkNode {
	return a.addrIndex[address]
}

func (a *Accessor) GetActiveNodes() []nodeinfo.NetworkNode {
	result := make([]nodeinfo.NetworkNode, len(a.active))
	copy(result, a.active)
	return result
}

func (a *Accessor) GetActiveNode(ref reference.Global) nodeinfo.NetworkNode {
	return a.refIndex[ref]
}

func (a *Accessor) GetWorkingNode(ref reference.Global) nodeinfo.NetworkNode {
	node := a.GetActiveNode(ref)
	if node == nil || node.GetPower() == 0 {
		return nil
	}
	return node
}

func (a *Accessor) GetWorkingNodes() []nodeinfo.NetworkNode {
	workingList := a.snapshot.nodeList[ListWorking]
	result := make([]nodeinfo.NetworkNode, len(workingList))
	copy(result, workingList)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID().Compare(result[j].ID()) < 0
	})
	return result
}

func GetSnapshotActiveNodes(snapshot *Snapshot) []nodeinfo.NetworkNode {
	joining := snapshot.nodeList[ListJoiner]
	idle := snapshot.nodeList[ListIdle]
	working := snapshot.nodeList[ListWorking]
	leaving := snapshot.nodeList[ListLeaving]

	joinersCount := len(joining)
	idlersCount := len(idle)
	workingCount := len(working)
	leavingCount := len(leaving)

	result := make([]nodeinfo.NetworkNode, joinersCount+idlersCount+workingCount+leavingCount)

	copy(result[:joinersCount], joining)
	copy(result[joinersCount:joinersCount+idlersCount], idle)
	copy(result[joinersCount+idlersCount:joinersCount+idlersCount+workingCount], working)
	copy(result[joinersCount+idlersCount+workingCount:], leaving)

	return result
}

func (a *Accessor) addToIndex(node nodeinfo.NetworkNode) {
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
		refIndex:  make(map[reference.Global]nodeinfo.NetworkNode),
		sidIndex:  make(map[node2.ShortNodeID]nodeinfo.NetworkNode),
		addrIndex: make(map[string]nodeinfo.NetworkNode),
	}
	result.active = GetSnapshotActiveNodes(snapshot)
	for _, node := range result.active {
		result.addToIndex(node)
	}
	return result
}
