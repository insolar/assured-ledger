// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodenetwork

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/storage"
	"github.com/insolar/assured-ledger/ledger-core/pulse"

	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/resolver"
	"github.com/insolar/assured-ledger/ledger-core/network/node"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network"
)

// NewNodeNetwork create active node component
func NewNodeNetwork(configuration configuration.Transport, certificate nodeinfo.Certificate) (network.NodeNetwork, error) { // nolint:staticcheck
	isJoiner := !network.OriginIsDiscovery(certificate)
	origin, err := createOrigin(configuration, certificate, isJoiner)
	if err != nil {
		return nil, throw.W(err, "Failed to create origin node")
	}
	nodeKeeper := NewNodeKeeper(origin)
	return nodeKeeper, nil
}

func createOrigin(configuration configuration.Transport, certificate nodeinfo.Certificate, joiner bool) (nodeinfo.NetworkNode, error) {
	publicAddress, err := resolveAddress(configuration)
	if err != nil {
		return nil, throw.W(err, "Failed to resolve public address")
	}

	role := certificate.GetRole()
	if role == member.PrimaryRoleUnknown {
		panic(throw.IllegalValue())
		// global.Info("[ createOrigin ] Use role LightMaterial, since no role in certificate")
		// role = member.PrimaryRoleLightMaterial
	}

	if joiner {
		return node.NewJoiningNode(certificate.GetNodeRef(), role, certificate.GetPublicKey(), publicAddress), nil
	}
	return node.NewActiveNode(certificate.GetNodeRef(), role, certificate.GetPublicKey(), publicAddress), nil
}

func resolveAddress(configuration configuration.Transport) (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", configuration.Address)
	if err != nil {
		return "", err
	}
	address, err := resolver.Resolve(configuration.FixedPublicAddress, addr.String())
	if err != nil {
		return "", err
	}
	return address, nil
}

// NewNodeKeeper create new NodeKeeper
func NewNodeKeeper(origin nodeinfo.NetworkNode) network.NodeKeeper {
	nk := &nodekeeper{
		origin:          origin,
		snapshotStorage: storage.NewMemoryStorage(),
	}
	return nk
}

type nodekeeper struct {
	syncLock  sync.RWMutex

	origin nodeinfo.NetworkNode
	syncNodes []nodeinfo.NetworkNode

	snapshotStorage *storage.MemoryStorage
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
	return node.NewAccessor(s)
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

	snapshot := node.NewSnapshot(number, nk.syncNodes)

	if err := nk.snapshotStorage.Append(snapshot); err != nil {
		return 0, 0, err
	}

	accessor := node.NewAccessor(snapshot)

	o := accessor.GetActiveNode(nk.origin.GetReference())
	nk._updateOrigin(o)

	return len(nk.syncNodes), len(accessor.GetActiveNodes()), nil
}

func (nk *nodekeeper) _updateOrigin(n nodeinfo.NetworkNode) {
	switch {
	case n == nil:
		panic(throw.IllegalValue())
	case n.GetReference() != nk.origin.GetReference():
		panic(throw.IllegalValue())
	}
	nk.origin = n
}

