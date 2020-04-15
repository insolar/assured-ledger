// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/rules"
)

func newWaitMajority(b *Base) *WaitMajority {
	return &WaitMajority{b, make(chan insolar.Pulse, 1)}
}

type WaitMajority struct {
	*Base
	majorityComplete chan insolar.Pulse
}

func (g *WaitMajority) Run(ctx context.Context, pulse insolar.Pulse) {
	g.switchOnMajorityRule(ctx, pulse)

	select {
	case <-g.bootstrapTimer.C:
		g.FailState(ctx, bootstrapTimeoutMessage)
	case newPulse := <-g.majorityComplete:
		g.Gatewayer.SwitchState(ctx, insolar.WaitMinRoles, newPulse)
	}
}

func (g *WaitMajority) GetState() insolar.NetworkState {
	return insolar.WaitMajority
}

func (g *WaitMajority) OnConsensusFinished(ctx context.Context, report network.Report) {
	g.switchOnMajorityRule(ctx, EnsureGetPulse(ctx, g.PulseAccessor, report.PulseNumber))
}

func (g *WaitMajority) switchOnMajorityRule(_ context.Context, pulse insolar.Pulse) {
	_, err := rules.CheckMajorityRule(
		g.CertificateManager.GetCertificate(),
		g.NodeKeeper.GetAccessor(pulse.PulseNumber).GetWorkingNodes(),
	)

	if err == nil {
		g.majorityComplete <- pulse
		close(g.majorityComplete)
	}
}
