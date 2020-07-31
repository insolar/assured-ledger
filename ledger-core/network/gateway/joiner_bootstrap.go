// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func newJoinerBootstrap(b *Base) *JoinerBootstrap {
	return &JoinerBootstrap{b}
}

// JoinerBootstrap void network state
type JoinerBootstrap struct {
	*Base
}

func (g *JoinerBootstrap) Run(ctx context.Context, p pulse.Data) {
	logger := inslogger.FromContext(ctx)
	cert := g.CertificateManager.GetCertificate()
	permit, err := g.BootstrapRequester.Authorize(ctx, cert)
	if err != nil {
		logger.Warn("Failed to authorize: ", err.Error())
		g.Gatewayer.SwitchState(ctx, network.NoNetworkState, p)
		return
	}

	resp, err := g.BootstrapRequester.Bootstrap(ctx, permit, *g.localCandidate)
	if err != nil {
		logger.Warn("Failed to bootstrap: ", err.Error())
		g.Gatewayer.SwitchState(ctx, network.NoNetworkState, p)
		return
	}

	logger.Infof("Bootstrapping to node %s", permit.Payload.ReconnectTo)

	// Reset backoff if not insolar.NoNetworkState.
	g.backoff = 0

	// no needed
	responsePulse := beat.Beat{}

	g.bootstrapETA = time.Second * time.Duration(resp.ETASeconds)
	g.bootstrapTimer = time.NewTimer(g.bootstrapETA)
	g.Gatewayer.SwitchState(ctx, network.WaitConsensus, responsePulse.Data)
}

func (g *JoinerBootstrap) GetState() network.State {
	return network.JoinerBootstrap
}
