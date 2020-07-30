// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeset

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Accessor struct {
	snapshot  *Snapshot
	local     nodeinfo.NetworkNode
	refIndex  map[reference.Global]nodeinfo.NetworkNode
	addrIndex map[string]nodeinfo.NetworkNode
}

func (a *Accessor) GetLocalNode() nodeinfo.NetworkNode {
	return a.local
}

func (a *Accessor) GetPulseNumber() pulse.Number {
	return a.snapshot.GetPulse()
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

func NewAccessor(snapshot *Snapshot, localRef reference.Global) *Accessor {
	result := &Accessor{
		snapshot:  snapshot,
		refIndex:  make(map[reference.Global]nodeinfo.NetworkNode),
		addrIndex: make(map[string]nodeinfo.NetworkNode),
	}

	for _, node := range snapshot.activeNodes {
		ref := nodeinfo.NodeRef(node)
		if ref == localRef {
			result.local = node
		}
		result.refIndex[ref] = node
		result.addrIndex[nodeinfo.NodeAddr(node)] = node
	}
	return result
}
