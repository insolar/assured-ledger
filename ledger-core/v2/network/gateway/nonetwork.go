// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

// TODO: spans, metrics

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
)

func newNoNetwork(b *Base) *NoNetwork {
	return &NoNetwork{Base: b}
}

// NoNetwork initial state
type NoNetwork struct {
	*Base
}

func (g *NoNetwork) pause() time.Duration {
	var sleep time.Duration
	switch g.backoff {
	case g.Options.MaxTimeout:
		sleep = g.backoff
	case 0:
		g.backoff = g.Options.MinTimeout
	default:
		sleep = g.backoff
		g.backoff *= g.Options.TimeoutMult
		if g.backoff > g.Options.MaxTimeout {
			g.backoff = g.Options.MaxTimeout
		}
	}
	return sleep
}

func (g *NoNetwork) Run(ctx context.Context, pulse insolar.Pulse) {
	cert := g.CertificateManager.GetCertificate()
	origin := g.NodeKeeper.GetOrigin()
	discoveryNodes := network.ExcludeOrigin(cert.GetDiscoveryNodes(), origin.ID())

	g.NodeKeeper.SetInitialSnapshot([]insolar.NetworkNode{origin})

	if len(discoveryNodes) == 0 {
		inslogger.FromContext(ctx).Warn("No discovery nodes found in certificate")
		return
	}

	if network.OriginIsJoinAssistant(cert) {
		// Reset backoff if not insolar.JoinerBootstrap.
		g.backoff = 0

		g.bootstrapTimer = time.NewTimer(g.bootstrapETA)
		g.Gatewayer.SwitchState(ctx, insolar.WaitConsensus, pulse)
		return
	}

	time.Sleep(g.pause())
	g.Gatewayer.SwitchState(ctx, insolar.JoinerBootstrap, pulse)
}

func (g *NoNetwork) GetState() insolar.NetworkState {
	return insolar.NoNetworkState
}
