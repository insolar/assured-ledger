// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/rules"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func newWaitMinRoles(b *Base) *WaitMinRoles {
	return &WaitMinRoles{b, make(chan pulse.Data, 1)}
}

type WaitMinRoles struct {
	*Base
	minrolesComplete chan pulse.Data
}

func (g *WaitMinRoles) Run(ctx context.Context, pulse pulse.Data) {
	g.switchOnMinRoles(ctx, pulse)

	select {
	case <-g.bootstrapTimer.C:
		g.FailState(ctx, bootstrapTimeoutMessage)
	case newPulse := <-g.minrolesComplete:
		g.Gatewayer.SwitchState(ctx, network.WaitPulsar, newPulse)
	}
}

func (g *WaitMinRoles) UpdateState(ctx context.Context, beat beat.Beat) {
	if _, err := rules.CheckMajorityRule(g.CertificateManager.GetCertificate(), beat.Online); err != nil {
		g.FailState(ctx, err.Error())
	}

	g.Base.UpdateState(ctx, beat)
}

// deprecated // improve
func SelectWorking(nodes []nodeinfo.NetworkNode) []nodeinfo.NetworkNode {
	result := make([]nodeinfo.NetworkNode, 0, len(nodes))
	for _, nd := range nodes {
		if isWorkingNode(nd) {
			result = append(result, nd)
		}
	}
	return result
}

func isWorkingNode(nd nodeinfo.NetworkNode) bool {
	return nd.IsPowered()
}

func (g *WaitMinRoles) GetState() network.State {
	return network.WaitMinRoles
}

func (g *WaitMinRoles) OnConsensusFinished(ctx context.Context, report network.Report) {
	g.switchOnMinRoles(ctx, EnsureGetPulse(ctx, report))
}

func (g *WaitMinRoles) switchOnMinRoles(_ context.Context, pulse pulse.Data) {
	err := rules.CheckMinRole(
		g.CertificateManager.GetCertificate(),
		g.NodeKeeper.GetNodeSnapshot(pulse.PulseNumber).GetPopulation(),
	)

	if err == nil {
		g.minrolesComplete <- pulse
		close(g.minrolesComplete)
	}
}
