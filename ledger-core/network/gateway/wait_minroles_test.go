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
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/mocklog"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func TestWaitMinroles_MinrolesNotHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	cert := &mandates.Certificate{}
	cert.MinRoles.HeavyMaterial = 1
	b := createBase(mc)
	b.CertificateManager = mandates.NewCertificateManager(cert)

	nodeKeeper := b.NodeKeeper.(*beat.NodeKeeperMock)

	role := census.NewRolePopulationMock(mc)
	role.GetWorkingCountMock.Return(0)
	role.GetIdleCountMock.Return(0)

	pop := census.NewOnlinePopulationMock(mc)
	pop.GetRolePopulationMock.Return(role)

	accessor := beat.NewNodeSnapshotMock(mc)
	accessor.GetPopulationMock.Return(pop)

	nodeKeeper.GetNodeSnapshotMock.Return(accessor)

	waitMinRoles := newWaitMinRoles(b)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.GatewayMock.Set(func() network.Gateway {
		return waitMinRoles
	})

	assert.Equal(t, network.WaitMinRoles, waitMinRoles.GetState())
	waitMinRoles.Gatewayer = gatewayer
	waitMinRoles.bootstrapETA = time.Millisecond
	waitMinRoles.bootstrapTimer = time.NewTimer(waitMinRoles.bootstrapETA)

	waitMinRoles.Run(context.Background(), EphemeralPulse.Data)
}

func TestWaitMinroles_MinrolesHappenedInETA(t *testing.T) {
	mc := minimock.NewController(mocklog.T(t))
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state network.State, pulse pulse.Data) {
		assert.Equal(t, network.WaitPulsar, state)
	})

	ref := gen.UniqueGlobalRef()
	nodeKeeper := beat.NewNodeKeeperMock(mc)

	role1 := census.NewRolePopulationMock(mc)
	role1.GetWorkingCountMock.Return(0)
	role1.GetIdleCountMock.Return(0)

	pop1 := census.NewOnlinePopulationMock(mc)
	pop1.GetRolePopulationMock.Return(role1)

	accessor1 := beat.NewNodeSnapshotMock(mc)
	accessor1.GetPopulationMock.Return(pop1)
	nodeKeeper.GetNodeSnapshotMock.When(pulse.MinTimePulse).Then(accessor1)

	role2 := census.NewRolePopulationMock(mc)
	role2.GetWorkingCountMock.Return(5)
	role2.GetIdleCountMock.Return(0)

	pop2 := census.NewOnlinePopulationMock(mc)
	pop2.GetRolePopulationMock.Return(role2)

	accessor2 := beat.NewNodeSnapshotMock(mc)
	accessor2.GetPopulationMock.Return(pop2)
	nodeKeeper.GetNodeSnapshotMock.When(pulse.MinTimePulse + 10).Then(accessor2)


	discoveryNode := mandates.BootstrapNode{NodeRef: ref.String()}
	cert := &mandates.Certificate{MajorityRule: 1, BootstrapNodes: []mandates.BootstrapNode{discoveryNode}}
	cert.MinRoles.LightMaterial = 1
	waitMinRoles := newWaitMinRoles(&Base{
		CertificateManager: mandates.NewCertificateManager(cert),
		NodeKeeper:         nodeKeeper,
	})
	waitMinRoles.Gatewayer = gatewayer
	waitMinRoles.bootstrapETA = time.Second * 2
	waitMinRoles.bootstrapTimer = time.NewTimer(waitMinRoles.bootstrapETA)

	go waitMinRoles.Run(context.Background(), EphemeralPulse.Data)
	time.Sleep(100 * time.Millisecond)

	waitMinRoles.OnConsensusFinished(context.Background(), network.Report{
		PulseNumber: pulse.MinTimePulse + 10,
		PulseData:   pulse.Data{PulseNumber: pulse.MinTimePulse + 10},
	})
}
