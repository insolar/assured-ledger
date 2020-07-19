// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

// TODO: spans, metrics

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
)

func newNoNetwork(b *Base) *NoNetwork {
	return &NoNetwork{Base: b}
}

// NoNetwork initial state
type NoNetwork struct {
	*Base
}

func (g *NoNetwork) pause() time.Duration {
	// todo: use synckit.Backoff
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

func (g *NoNetwork) Run(ctx context.Context, pulse pulsestor.Pulse) {
	cert := g.CertificateManager.GetCertificate()
	origin := g.NodeKeeper.GetOrigin()
	discoveryNodes := network.ExcludeOrigin(cert.GetDiscoveryNodes(), origin.ID())

	g.NodeKeeper.SetInitialSnapshot([]node.NetworkNode{origin})

	if len(discoveryNodes) == 0 {
		inslogger.FromContext(ctx).Warn("No discovery nodes found in certificate")
		return
	}

	// remember who is Me and who is joinAssistant
	g.isDiscovery = network.OriginIsDiscovery(cert)
	g.isJoinAssistant = network.OriginIsJoinAssistant(cert)
	g.joinAssistant = network.JoinAssistant(cert)

	if g.isJoinAssistant {
		// Reset backoff if not insolar.JoinerBootstrap.
		g.backoff = 0

		g.bootstrapTimer = time.NewTimer(g.bootstrapETA)
		g.Gatewayer.SwitchState(ctx, node.WaitConsensus, pulse)
		return
	}

	time.Sleep(g.pause())
	if g.isDiscovery {
		g.Gatewayer.SwitchState(ctx, node.DiscoveryBootstrap, pulse)
	} else {
		g.Gatewayer.SwitchState(ctx, node.JoinerBootstrap, pulse)
	}
}

func (g *NoNetwork) GetState() node.NetworkState {
	return node.NoNetworkState
}
