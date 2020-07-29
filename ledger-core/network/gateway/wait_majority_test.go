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
	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func TestWaitMajority_MajorityNotHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	nodeKeeper := mock.NewNodeKeeperMock(mc)
	nodeKeeper.GetAccessorMock.Set(func(p1 pulse.Number) (a1 network.Accessor) {
		accessor := mock.NewAccessorMock(mc)
		accessor.GetWorkingNodesMock.Set(func() (na1 []nodeinfo.NetworkNode) {
			return []nodeinfo.NetworkNode{}
		})
		return accessor
	})

	cert := &mandates.Certificate{MajorityRule: 4}

	b := createBase(mc)
	b.CertificateManager = mandates.NewCertificateManager(cert)
	b.NodeKeeper = nodeKeeper

	waitMajority := newWaitMajority(b)
	assert.Equal(t, nodeinfo.WaitMajority, waitMajority.GetState())
	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.GatewayMock.Set(func() network.Gateway {
		return waitMajority
	})
	waitMajority.Gatewayer = gatewayer
	waitMajority.bootstrapETA = time.Millisecond
	waitMajority.bootstrapTimer = time.NewTimer(waitMajority.bootstrapETA)

	waitMajority.Run(context.Background(), EphemeralPulse.Data)
}

func TestWaitMajority_MajorityHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state nodeinfo.NetworkState, pulse pulse.Data) {
		assert.Equal(t, nodeinfo.WaitMinRoles, state)
	})

	ref := gen.UniqueGlobalRef()
	nodeKeeper := mock.NewNodeKeeperMock(mc)
	accessor1 := mock.NewAccessorMock(mc)
	accessor1.GetWorkingNodesMock.Set(func() (na1 []nodeinfo.NetworkNode) {
		return []nodeinfo.NetworkNode{}
	})
	accessor2 := mock.NewAccessorMock(mc)
	accessor2.GetWorkingNodesMock.Set(func() (na1 []nodeinfo.NetworkNode) {
		n := node.NewNode(ref, member.PrimaryRoleHeavyMaterial, nil, "127.0.0.1:123")
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
