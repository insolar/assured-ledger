// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/network/rules"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func newWaitMinRoles(b *Base) *WaitMinRoles {
	return &WaitMinRoles{b, make(chan network.NetworkedPulse, 1)}
}

type WaitMinRoles struct {
	*Base
	minrolesComplete chan network.NetworkedPulse
}

func (g *WaitMinRoles) Run(ctx context.Context, pulse network.NetworkedPulse) {
	g.switchOnMinRoles(ctx, pulse)

	select {
	case <-g.bootstrapTimer.C:
		g.FailState(ctx, bootstrapTimeoutMessage)
	case newPulse := <-g.minrolesComplete:
		g.Gatewayer.SwitchState(ctx, node2.WaitPulsar, newPulse)
	}
}

func (g *WaitMinRoles) UpdateState(ctx context.Context, pulseNumber pulse.Number, nodes []node2.NetworkNode, cloudStateHash []byte) {
	workingNodes := node.Select(nodes, node.ListWorking)

	if _, err := rules.CheckMajorityRule(g.CertificateManager.GetCertificate(), workingNodes); err != nil {
		g.FailState(ctx, err.Error())
	}

	g.Base.UpdateState(ctx, pulseNumber, nodes, cloudStateHash)
}

func (g *WaitMinRoles) GetState() node2.NetworkState {
	return node2.WaitMinRoles
}

func (g *WaitMinRoles) OnConsensusFinished(ctx context.Context, report network.Report) {
	g.switchOnMinRoles(ctx, EnsureGetPulse(ctx, g.PulseAccessor, report.PulseNumber))
}

func (g *WaitMinRoles) switchOnMinRoles(_ context.Context, pulse network.NetworkedPulse) {
	err := rules.CheckMinRole(
		g.CertificateManager.GetCertificate(),
		g.NodeKeeper.GetAccessor(pulse.PulseNumber).GetWorkingNodes(),
	)

	if err == nil {
		g.minrolesComplete <- pulse
		close(g.minrolesComplete)
	}
}
