// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodenetwork

import (
	"context"
	"net"
	"sync"

	node2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/storage"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/hostnetwork/resolver"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/node"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"

	"github.com/pkg/errors"
	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/version"
)

// NewNodeNetwork create active node component
func NewNodeNetwork(configuration configuration.Transport, certificate insolar.Certificate) (network.NodeNetwork, error) { // nolint:staticcheck
	origin, err := createOrigin(configuration, certificate)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create origin node")
	}
	nodeKeeper := NewNodeKeeper(origin)
	if !network.OriginIsDiscovery(certificate) {
		origin.(node.MutableNode).SetState(node2.NodeJoining)
	}
	return nodeKeeper, nil
}

func createOrigin(configuration configuration.Transport, certificate insolar.Certificate) (node2.NetworkNode, error) {
	publicAddress, err := resolveAddress(configuration)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to resolve public address")
	}

	role := certificate.GetRole()
	if role == node2.StaticRoleUnknown {
		global.Info("[ createOrigin ] Use insolar.StaticRoleLightMaterial, since no role in certificate")
		role = node2.StaticRoleLightMaterial
	}

	return node.NewNode(
		certificate.GetNodeRef(),
		role,
		certificate.GetPublicKey(),
		publicAddress,
		version.Version,
	), nil
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
func NewNodeKeeper(origin node2.NetworkNode) network.NodeKeeper {
	nk := &nodekeeper{
		origin:          origin,
		syncNodes:       make([]node2.NetworkNode, 0),
		SnapshotStorage: storage.NewMemoryStorage(),
	}
	return nk
}

type nodekeeper struct {
	origin node2.NetworkNode

	syncLock  sync.RWMutex
	syncNodes []node2.NetworkNode

	SnapshotStorage storage.SnapshotStorage
}

func (nk *nodekeeper) SetInitialSnapshot(nodes []node2.NetworkNode) {
	ctx := context.TODO()
	nk.Sync(ctx, pulse.MinTimePulse, nodes)
	nk.MoveSyncToActive(ctx, pulse.MinTimePulse)
}

func (nk *nodekeeper) GetAccessor(pn insolar.PulseNumber) network.Accessor {
	s, err := nk.SnapshotStorage.ForPulseNumber(pn)
	if err != nil {
		panic("GetAccessor(): " + err.Error())
	}
	return node.NewAccessor(s)
}

func (nk *nodekeeper) GetOrigin() node2.NetworkNode {
	nk.syncLock.RLock()
	defer nk.syncLock.RUnlock()

	return nk.origin
}

func (nk *nodekeeper) Sync(ctx context.Context, number insolar.PulseNumber, nodes []node2.NetworkNode) {
	nk.syncLock.Lock()
	defer nk.syncLock.Unlock()

	inslogger.FromContext(ctx).Debugf("Sync, nodes: %d", len(nodes))
	nk.syncNodes = nodes
}

func (nk *nodekeeper) updateOrigin(power node2.Power, state node2.NodeState) {
	nk.origin.(node.MutableNode).SetPower(power)
	nk.origin.(node.MutableNode).SetState(state)
}

func (nk *nodekeeper) MoveSyncToActive(ctx context.Context, number insolar.PulseNumber) {
	nk.syncLock.Lock()
	defer nk.syncLock.Unlock()

	snapshot := node.NewSnapshot(number, nk.syncNodes)
	err := nk.SnapshotStorage.Append(number, snapshot)
	if err != nil {
		inslogger.FromContext(ctx).Panic("MoveSyncToActive(): ", err.Error())
	}

	accessor := node.NewAccessor(snapshot)

	inslogger.FromContext(ctx).Infof("[ MoveSyncToActive ] New active list confirmed. Active list size: %d -> %d",
		len(nk.syncNodes),
		len(accessor.GetActiveNodes()),
	)

	o := accessor.GetActiveNode(nk.origin.ID())
	nk.updateOrigin(o.GetPower(), o.GetState())

	stats.Record(ctx, network.ActiveNodes.M(int64(len(accessor.GetActiveNodes()))))
}
