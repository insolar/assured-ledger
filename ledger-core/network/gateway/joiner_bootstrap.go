// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/gateway/bootstrap"
)

func newJoinerBootstrap(b *Base) *JoinerBootstrap {
	return &JoinerBootstrap{b}
}

// JoinerBootstrap void network state
type JoinerBootstrap struct {
	*Base
}

func (g *JoinerBootstrap) Run(ctx context.Context, p beat.Beat) {
	logger := inslogger.FromContext(ctx)
	cert := g.CertificateManager.GetCertificate()
	permit, err := g.BootstrapRequester.Authorize(ctx, cert)
	if err != nil {
		logger.Warn("Failed to authorize: ", err.Error())
		g.Gatewayer.SwitchState(ctx, nodeinfo.NoNetworkState, p)
		return
	}

	resp, err := g.BootstrapRequester.Bootstrap(ctx, permit, *g.originCandidate, p)
	if err != nil {
		logger.Warn("Failed to bootstrap: ", err.Error())
		g.Gatewayer.SwitchState(ctx, nodeinfo.NoNetworkState, p)
		return
	}

	logger.Infof("Bootstrapping to node %s", permit.Payload.ReconnectTo)

	// Reset backoff if not insolar.NoNetworkState.
	g.backoff = 0

	responsePulse := bootstrap.FromProto(&resp.Pulse)

	g.bootstrapETA = time.Second * time.Duration(resp.ETASeconds)
	g.bootstrapTimer = time.NewTimer(g.bootstrapETA)
	g.Gatewayer.SwitchState(ctx, nodeinfo.WaitConsensus, responsePulse)
}

func (g *JoinerBootstrap) GetState() nodeinfo.NetworkState {
	return nodeinfo.JoinerBootstrap
}
