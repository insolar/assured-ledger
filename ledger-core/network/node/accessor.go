// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Accessor struct {
	snapshot  *Snapshot
	refIndex  map[reference.Global]nodeinfo.NetworkNode
	addrIndex map[string]nodeinfo.NetworkNode
}

func (a *Accessor) GetActiveNodeByAddr(address string) nodeinfo.NetworkNode {
	return a.addrIndex[address]
}

func (a *Accessor) GetActiveNodes() []nodeinfo.NetworkNode {
	return append([]nodeinfo.NetworkNode(nil), a.snapshot.activeNodes...)
}

func (a *Accessor) GetActiveNode(ref reference.Global) nodeinfo.NetworkNode {
	return a.refIndex[ref]
}

func (a *Accessor) GetWorkingNode(ref reference.Global) nodeinfo.NetworkNode {
	node := a.GetActiveNode(ref)
	if node == nil || !isWorkingNode(node) {
		return nil
	}
	return node
}

func (a *Accessor) GetWorkingNodes() []nodeinfo.NetworkNode {
	return append([]nodeinfo.NetworkNode(nil), a.snapshot.workingNodes...)
}

func (a *Accessor) addToIndex(node nodeinfo.NetworkNode) {
	a.refIndex[node.GetReference()] = node
	a.addrIndex[node.Address()] = node
}

func NewAccessor(snapshot *Snapshot) *Accessor {
	result := &Accessor{
		snapshot:  snapshot,
		refIndex:  make(map[reference.Global]nodeinfo.NetworkNode),
		addrIndex: make(map[string]nodeinfo.NetworkNode),
	}
	for _, node := range snapshot.activeNodes {
		result.addToIndex(node)
	}
	return result
}
