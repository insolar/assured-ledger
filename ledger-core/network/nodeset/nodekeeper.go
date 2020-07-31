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

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network"
)

// NewNodeKeeper create new NodeKeeper
func NewNodeKeeper(localRef reference.Holder, localRole member.PrimaryRole) network.NodeKeeper {
	return &nodekeeper{
		localRef:  reference.Copy(localRef),
		localRole: localRole,
		storage:   NewMemoryStorage(),
	}
}

type nodekeeper struct {
	localRef  reference.Global
	localRole member.PrimaryRole

	mutex sync.RWMutex

	storage *MemoryStorage
	last    network.Accessor
	expected int
}

func (nk *nodekeeper) GetLocalNodeReference() reference.Holder {
	return nk.localRef
}

func (nk *nodekeeper) GetLocalNodeRole() member.PrimaryRole {
	return nk.localRole
}

func (nk *nodekeeper) GetAccessor(pn pulse.Number) network.Accessor {
	la := nk.GetLatestAccessor()
	if la != nil && la.GetPulseNumber() == pn {
		return la
	}

	s, err := nk.storage.ForPulseNumber(pn)
	if err != nil {
		panic(fmt.Sprintf("GetAccessor(%d): %s", pn, err.Error()))
	}
	return NewAccessor(s)
}

func (nk *nodekeeper) GetLatestAccessor() network.Accessor {
	nk.mutex.RLock()
	defer nk.mutex.RUnlock()

	if nk.last == nil {
		return nil
	}
	return nk.last
}

func (nk *nodekeeper) SetExpectedPopulation(ctx context.Context, _ pulse.Number, nodes census.OnlinePopulation) {
	inslogger.FromContext(ctx).Debugf("SetExpectedPopulation, nodes: %d", nodes.GetIndexedCount())

	nk.mutex.Lock()
	defer nk.mutex.Unlock()
	nk.expected = nodes.GetIndexedCount()
}

func (nk *nodekeeper) AddActivePopulation(ctx context.Context, pn pulse.Number, population census.OnlinePopulation) {
	before, err := nk.moveSyncToActive(pn, population)
	if err != nil {
		inslogger.FromContext(ctx).Panic("AddActivePopulation(): ", err.Error())
	}

	after := population.GetIndexedCount()
	inslogger.FromContext(ctx).Infof("[ AddActivePopulation ] New active list confirmed. Active list size: %d -> %d",
		before, after,
	)

	stats.Record(ctx, network.ActiveNodes.M(int64(after)))
}

func (nk *nodekeeper) moveSyncToActive(number pulse.Number, population census.OnlinePopulation) (before int, err error) {
	if nodeinfo.NodeRef(population.GetLocalProfile()) != nk.localRef {
		panic(throw.IllegalValue())
	}

	nk.mutex.Lock()
	defer nk.mutex.Unlock()

	snapshot := NewSnapshot(number, population)
	accessor := NewAccessor(snapshot)

	if err := nk.storage.Append(snapshot); err != nil {
		return 0, err
	}
	nk.last = accessor

	return nk.expected, nil
}
