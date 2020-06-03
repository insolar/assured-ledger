// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"sync"

	"go.opencensus.io/stats"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
)

func NewGatewayer(g network.Gateway) network.Gatewayer {
	return &gatewayer{
		gateway: g,
	}
}

type gatewayer struct {
	gatewayMu sync.RWMutex
	gateway   network.Gateway
}

func (n *gatewayer) Gateway() network.Gateway {
	n.gatewayMu.RLock()
	defer n.gatewayMu.RUnlock()

	return n.gateway
}

func (n *gatewayer) SwitchState(ctx context.Context, state node.NetworkState, pulse pulsestor.Pulse) {
	n.gatewayMu.Lock()
	defer n.gatewayMu.Unlock()

	inslogger.FromContext(ctx).Infof("Gateway switch %s->%s, pulse: %d", n.gateway.GetState(), state, pulse.PulseNumber)

	if n.gateway.GetState() == state {
		inslogger.FromContext(ctx).Warn("Trying to set gateway to the same state")
		return
	}

	gateway := n.gateway.NewGateway(ctx, state)
	gateway.BeforeRun(ctx, pulse)

	n.gateway = gateway
	go n.gateway.Run(ctx, pulse)
	stats.Record(ctx, networkState.M(int64(state)))
}
