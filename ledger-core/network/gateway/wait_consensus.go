package gateway

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func newWaitConsensus(b *Base) *WaitConsensus {
	return &WaitConsensus{b, make(chan pulse.Data, 1)}
}

type WaitConsensus struct {
	*Base

	consensusFinished chan pulse.Data
}

func (g *WaitConsensus) Run(ctx context.Context, _ pulse.Data) {
	select {
	case <-g.bootstrapTimer.C:
		g.FailState(ctx, bootstrapTimeoutMessage)
	case newPulse := <-g.consensusFinished:
		g.Gatewayer.SwitchState(ctx, network.WaitMajority, newPulse)
	}
}

func (g *WaitConsensus) GetState() network.State {
	return network.WaitConsensus
}

func (g *WaitConsensus) OnConsensusFinished(ctx context.Context, report network.Report) {
	g.consensusFinished <- EnsureGetPulse(ctx, report)
	close(g.consensusFinished)
}
