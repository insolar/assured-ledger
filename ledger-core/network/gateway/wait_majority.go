// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/rules"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func newWaitMajority(b *Base) *WaitMajority {
	return &WaitMajority{b, make(chan pulse.Data, 1)}
}

type WaitMajority struct {
	*Base
	majorityComplete chan pulse.Data
}

func (g *WaitMajority) Run(ctx context.Context, pulse pulse.Data) {
	g.switchOnMajorityRule(ctx, pulse)

	select {
	case <-g.bootstrapTimer.C:
		g.FailState(ctx, bootstrapTimeoutMessage)
	case newPulse := <-g.majorityComplete:
		g.Gatewayer.SwitchState(ctx, network.WaitMinRoles, newPulse)
	}
}

func (g *WaitMajority) GetState() network.State {
	return network.WaitMajority
}

func (g *WaitMajority) OnConsensusFinished(ctx context.Context, report network.Report) {
	g.switchOnMajorityRule(ctx, EnsureGetPulse(ctx, report))
}

func (g *WaitMajority) switchOnMajorityRule(_ context.Context, pulse pulse.Data) {
	_, err := rules.CheckMajorityRule(
		g.CertificateManager.GetCertificate(),
		g.NodeKeeper.GetNodeSnapshot(pulse.PulseNumber).GetPopulation(),
	)

	if err == nil {
		g.majorityComplete <- pulse
		close(g.majorityComplete)
	}
}
