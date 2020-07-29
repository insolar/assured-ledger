// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

// TODO: spans, metrics

import (
	"context"
	"math/rand"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func newNoNetwork(b *Base) *NoNetwork {
	return &NoNetwork{Base: b}
}

// NoNetwork initial state
type NoNetwork struct {
	*Base
}

func (g *NoNetwork) Run(ctx context.Context, pulse pulse.Data) {
	cert := g.CertificateManager.GetCertificate()
	origin := g.NodeKeeper.GetOrigin()
	discoveryNodes := network.ExcludeOrigin(cert.GetDiscoveryNodes(), origin.GetReference())

	g.NodeKeeper.SetInitialSnapshot([]nodeinfo.NetworkNode{origin})

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
		g.backoff.Reset()

		g.bootstrapTimer = time.NewTimer(g.bootstrapETA)
		g.Gatewayer.SwitchState(ctx, nodeinfo.WaitConsensus, pulse)
		return
	}

	time.Sleep(g.backoff.Duration())
	if g.isDiscovery {
		time.Sleep(time.Second * time.Duration(rand.Intn(20)))
		g.Gatewayer.SwitchState(ctx, nodeinfo.DiscoveryBootstrap, pulse)
	} else {
		g.Gatewayer.SwitchState(ctx, nodeinfo.JoinerBootstrap, pulse)
	}
}

func (g *NoNetwork) GetState() nodeinfo.NetworkState {
	return nodeinfo.NoNetworkState
}
