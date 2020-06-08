// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/network/rules"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func newWaitPulsar(b *Base) *WaitPulsar {
	return &WaitPulsar{b, make(chan pulsestor.Pulse, 1)}
}

type WaitPulsar struct {
	*Base
	pulseArrived chan pulsestor.Pulse
}

func (g *WaitPulsar) Run(ctx context.Context, pulse pulsestor.Pulse) {
	g.switchOnRealPulse(pulse)

	select {
	case <-g.bootstrapTimer.C:
		g.FailState(ctx, bootstrapTimeoutMessage)
	case newPulse := <-g.pulseArrived:
		g.Gatewayer.SwitchState(ctx, node2.CompleteNetworkState, newPulse)
	}
}

func (g *WaitPulsar) UpdateState(ctx context.Context, pulseNumber pulse.Number, nodes []node2.NetworkNode, cloudStateHash []byte) {
	workingNodes := node.Select(nodes, node.ListWorking)

	if _, err := rules.CheckMajorityRule(g.CertificateManager.GetCertificate(), workingNodes); err != nil {
		g.FailState(ctx, err.Error())
	}

	if err := rules.CheckMinRole(g.CertificateManager.GetCertificate(), workingNodes); err != nil {
		g.FailState(ctx, err.Error())
	}

	g.Base.UpdateState(ctx, pulseNumber, nodes, cloudStateHash)
}

func (g *WaitPulsar) GetState() node2.NetworkState {
	return node2.WaitPulsar
}

func (g *WaitPulsar) OnConsensusFinished(ctx context.Context, report network.Report) {
	g.switchOnRealPulse(EnsureGetPulse(ctx, g.PulseAccessor, report.PulseNumber))
}

func (g *WaitPulsar) switchOnRealPulse(pulseObject pulsestor.Pulse) {
	if pulseObject.PulseNumber.IsTimePulse() && pulseObject.EpochPulseNumber.IsTimeEpoch() {
		g.pulseArrived <- pulseObject
		close(g.pulseArrived)
	}
}
