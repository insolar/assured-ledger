// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network"
)

func newWaitConsensus(b *Base) *WaitConsensus {
	return &WaitConsensus{b, make(chan network.NetworkedPulse, 1)}
}

type WaitConsensus struct {
	*Base

	consensusFinished chan network.NetworkedPulse
}

func (g *WaitConsensus) Run(ctx context.Context, pulse network.NetworkedPulse) {
	select {
	case <-g.bootstrapTimer.C:
		g.FailState(ctx, bootstrapTimeoutMessage)
	case newPulse := <-g.consensusFinished:
		g.Gatewayer.SwitchState(ctx, nodeinfo.WaitMajority, newPulse)
	}
}

func (g *WaitConsensus) GetState() nodeinfo.NetworkState {
	return nodeinfo.WaitConsensus
}

func (g *WaitConsensus) OnConsensusFinished(ctx context.Context, report network.Report) {
	g.consensusFinished <- EnsureGetPulse(ctx, g.PulseAccessor, report.PulseNumber)
	close(g.consensusFinished)
}
