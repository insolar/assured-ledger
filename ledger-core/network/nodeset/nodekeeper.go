// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeset

import (
	"context"
	"fmt"
	"sync"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network"
)

// NewNodeKeeper create new NodeKeeper
func NewNodeKeeper(origin nodeinfo.NetworkNode) network.NodeKeeper {
	nk := &nodekeeper{
		origin:          origin,
		originRef:       nodeinfo.NodeRef(origin),
		snapshotStorage: NewMemoryStorage(),
	}
	return nk
}

type nodekeeper struct {
	syncLock  sync.RWMutex

	originRef reference.Global
	origin nodeinfo.NetworkNode
	syncNodes []nodeinfo.NetworkNode

	snapshotStorage *MemoryStorage
}

func (nk *nodekeeper) GetLocalNodeReference() reference.Holder {
	return nk.originRef
}

func (nk *nodekeeper) SetInitialSnapshot(nodes []nodeinfo.NetworkNode) {
	ctx := context.TODO()
	nk.Sync(ctx, nodes)
	nk.MoveSyncToActive(ctx, pulse.Unknown)
	nk.Sync(ctx, nodes)
	nk.MoveSyncToActive(ctx, pulse.MinTimePulse)
}

func (nk *nodekeeper) GetAccessor(pn pulse.Number) network.Accessor {
	s, err := nk.snapshotStorage.ForPulseNumber(pn)
	if err != nil {
		panic(fmt.Sprintf("GetAccessor(%d): %s", pn, err.Error()))
	}
	return NewAccessor(s)
}

func (nk *nodekeeper) GetOrigin() nodeinfo.NetworkNode {
	nk.syncLock.RLock()
	defer nk.syncLock.RUnlock()

	return nk.origin
}

func (nk *nodekeeper) Sync(ctx context.Context, nodes []nodeinfo.NetworkNode) {
	inslogger.FromContext(ctx).Debugf("Sync, nodes: %d", len(nodes))

	nk.syncLock.Lock()
	defer nk.syncLock.Unlock()
	nk.syncNodes = nodes
}

func (nk *nodekeeper) MoveSyncToActive(ctx context.Context, pn pulse.Number) {
	before, after, err := nk.moveSyncToActive(pn)
	if err != nil {
		inslogger.FromContext(ctx).Panic("MoveSyncToActive(): ", err.Error())
	}

	inslogger.FromContext(ctx).Infof("[ MoveSyncToActive ] New active list confirmed. Active list size: %d -> %d",
		before, after,
	)

	stats.Record(ctx, network.ActiveNodes.M(int64(after)))
}

func (nk *nodekeeper) moveSyncToActive(number pulse.Number) (before, after int, err error) {
	nk.syncLock.Lock()
	defer nk.syncLock.Unlock()

	snapshot := NewSnapshot(number, nk.syncNodes)

	if err := nk.snapshotStorage.Append(snapshot); err != nil {
		return 0, 0, err
	}

	accessor := NewAccessor(snapshot)

	o := accessor.GetActiveNode(nk.originRef)
	if o == nil {
		panic(throw.IllegalValue())
	}
	nk.origin = o

	return len(nk.syncNodes), len(accessor.GetActiveNodes()), nil
}
