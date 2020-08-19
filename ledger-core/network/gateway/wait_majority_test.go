// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"
)

func TestWaitMajority_MajorityNotHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	cert := &mandates.Certificate{MajorityRule: 4}

	b := createBase(mc)
	b.CertificateManager = mandates.NewCertificateManager(cert)
	nodeKeeper := b.NodeKeeper.(*beat.NodeKeeperMock)

	pop := census.NewOnlinePopulationMock(mc)
	pop.GetProfilesMock.Return(nil)

	accessor := beat.NewNodeSnapshotMock(mc)
	accessor.GetPopulationMock.Return(pop)

	nodeKeeper.GetNodeSnapshotMock.Return(accessor)

	waitMajority := newWaitMajority(b)
	assert.Equal(t, network.WaitMajority, waitMajority.GetState())
	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.GatewayMock.Return(waitMajority)
	waitMajority.Gatewayer = gatewayer
	waitMajority.bootstrapETA = time.Millisecond
	waitMajority.bootstrapTimer = time.NewTimer(waitMajority.bootstrapETA)

	waitMajority.Run(context.Background(), EphemeralPulse.Data)
}

func TestWaitMajority_MajorityHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state network.State, pulse pulse.Data) {
		assert.Equal(t, network.WaitMinRoles, state)
	})

	ref := gen.UniqueGlobalRef()
	nodeKeeper := beat.NewNodeKeeperMock(mc)

	pop1 := census.NewOnlinePopulationMock(mc)
	pop1.GetProfilesMock.Return(nil)

	accessor1 := beat.NewNodeSnapshotMock(mc)
	accessor1.GetPopulationMock.Return(pop1)
	nodeKeeper.GetNodeSnapshotMock.When(pulse.MinTimePulse).Then(accessor1)


	n := mutable.NewTestNode(ref, member.PrimaryRoleHeavyMaterial, "127.0.0.1:123")
	pop2 := census.NewOnlinePopulationMock(mc)
	pop2.GetProfilesMock.Return([]nodeinfo.NetworkNode{n})

	accessor2 := beat.NewNodeSnapshotMock(mc)
	accessor2.GetPopulationMock.Return(pop2)
	nodeKeeper.GetNodeSnapshotMock.When(pulse.MinTimePulse + 10).Then(accessor2)

	discoveryNode := mandates.BootstrapNode{NodeRef: ref.String()}
	cert := &mandates.Certificate{MajorityRule: 1, BootstrapNodes: []mandates.BootstrapNode{discoveryNode}}
	waitMajority := newWaitMajority(&Base{
		CertificateManager: mandates.NewCertificateManager(cert),
		NodeKeeper:         nodeKeeper,
	})
	waitMajority.Gatewayer = gatewayer
	waitMajority.bootstrapETA = time.Second * 2
	waitMajority.bootstrapTimer = time.NewTimer(waitMajority.bootstrapETA)

	go waitMajority.Run(context.Background(), EphemeralPulse.Data)
	time.Sleep(100 * time.Millisecond)

	waitMajority.OnConsensusFinished(context.Background(), network.Report{
		PulseNumber: pulse.MinTimePulse + 10,
		PulseData:   pulse.Data{PulseNumber: pulse.MinTimePulse + 10},
	})
}
