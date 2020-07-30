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

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"
)

func TestWaitMinroles_MinrolesNotHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	cert := &mandates.Certificate{}
	cert.MinRoles.HeavyMaterial = 1
	b := createBase(mc)
	b.CertificateManager = mandates.NewCertificateManager(cert)

	nodeKeeper := b.NodeKeeper.(*mock.NodeKeeperMock)
	accessor := mock.NewAccessorMock(mc)
	accessor.GetWorkingNodesMock.Return([]nodeinfo.NetworkNode{})
	nodeKeeper.GetAccessorMock.Return(accessor)

	waitMinRoles := newWaitMinRoles(b)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.GatewayMock.Set(func() network.Gateway {
		return waitMinRoles
	})

	assert.Equal(t, nodeinfo.WaitMinRoles, waitMinRoles.GetState())
	waitMinRoles.Gatewayer = gatewayer
	waitMinRoles.bootstrapETA = time.Millisecond
	waitMinRoles.bootstrapTimer = time.NewTimer(waitMinRoles.bootstrapETA)

	waitMinRoles.Run(context.Background(), EphemeralPulse.Data)
}

func TestWaitMinroles_MinrolesHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state nodeinfo.NetworkState, pulse pulse.Data) {
		assert.Equal(t, nodeinfo.WaitPulsar, state)
	})

	ref := gen.UniqueGlobalRef()
	nodeKeeper := mock.NewNodeKeeperMock(mc)

	accessor1 := mock.NewAccessorMock(mc)
	accessor1.GetWorkingNodesMock.Set(func() (na1 []nodeinfo.NetworkNode) {
		return []nodeinfo.NetworkNode{}
	})
	accessor2 := mock.NewAccessorMock(mc)
	accessor2.GetWorkingNodesMock.Set(func() (na1 []nodeinfo.NetworkNode) {
		n := mutable.NewTestNode(ref, member.PrimaryRoleLightMaterial, nil, "127.0.0.1:123")
		return []nodeinfo.NetworkNode{n}
	})
	nodeKeeper.GetAccessorMock.Set(func(p pulse.Number) (a1 network.Accessor) {
		if p == pulse.MinTimePulse {
			return accessor1
		}
		return accessor2
	})

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
