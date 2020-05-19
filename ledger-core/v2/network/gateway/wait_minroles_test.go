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

	"github.com/insolar/assured-ledger/ledger-core/v2/certificate"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	node2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	pulse2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	mock "github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"
)

func TestWaitMinroles_MinrolesNotHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	nodeKeeper := mock.NewNodeKeeperMock(mc)
	nodeKeeper.GetAccessorMock.Set(func(p1 pulse.Number) (a1 network.Accessor) {
		accessor := mock.NewAccessorMock(mc)
		accessor.GetWorkingNodesMock.Set(func() (na1 []node2.NetworkNode) {
			return []node2.NetworkNode{}
		})
		return accessor
	})

	cert := &certificate.Certificate{}
	cert.MinRoles.HeavyMaterial = 1
	b := createBase(mc)
	b.CertificateManager = certificate.NewCertificateManager(cert)
	b.NodeKeeper = nodeKeeper
	waitMinRoles := newWaitMinRoles(b)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.GatewayMock.Set(func() network.Gateway {
		return waitMinRoles
	})

	assert.Equal(t, node2.WaitMinRoles, waitMinRoles.GetState())
	waitMinRoles.Gatewayer = gatewayer
	waitMinRoles.bootstrapETA = time.Millisecond
	waitMinRoles.bootstrapTimer = time.NewTimer(waitMinRoles.bootstrapETA)

	waitMinRoles.Run(context.Background(), *pulse2.EphemeralPulse)
}

func TestWaitMinroles_MinrolesHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state node2.NetworkState, pulse pulse2.Pulse) {
		assert.Equal(t, node2.WaitPulsar, state)
	})

	ref := gen.Reference()
	nodeKeeper := mock.NewNodeKeeperMock(mc)

	accessor1 := mock.NewAccessorMock(mc)
	accessor1.GetWorkingNodesMock.Set(func() (na1 []node2.NetworkNode) {
		return []node2.NetworkNode{}
	})
	accessor2 := mock.NewAccessorMock(mc)
	accessor2.GetWorkingNodesMock.Set(func() (na1 []node2.NetworkNode) {
		n := node.NewNode(ref, node2.StaticRoleLightMaterial, nil, "127.0.0.1:123", "")
		return []node2.NetworkNode{n}
	})
	nodeKeeper.GetAccessorMock.Set(func(p pulse.Number) (a1 network.Accessor) {
		if p == pulse.MinTimePulse {
			return accessor1
		}
		return accessor2
	})

	discoveryNode := certificate.BootstrapNode{NodeRef: ref.String()}
	cert := &certificate.Certificate{MajorityRule: 1, BootstrapNodes: []certificate.BootstrapNode{discoveryNode}}
	cert.MinRoles.LightMaterial = 1
	pulseAccessor := mock.NewPulseAccessorMock(mc)
	pulseAccessor.GetPulseMock.Set(func(ctx context.Context, p1 pulse.Number) (p2 pulse2.Pulse, err error) {
		p := *pulse2.GenesisPulse
		p.PulseNumber += 10
		return p, nil
	})
	waitMinRoles := newWaitMinRoles(&Base{
		CertificateManager: certificate.NewCertificateManager(cert),
		NodeKeeper:         nodeKeeper,
		PulseAccessor:      pulseAccessor,
	})
	waitMinRoles.Gatewayer = gatewayer
	waitMinRoles.bootstrapETA = time.Second * 2
	waitMinRoles.bootstrapTimer = time.NewTimer(waitMinRoles.bootstrapETA)

	go waitMinRoles.Run(context.Background(), *pulse2.EphemeralPulse)
	time.Sleep(100 * time.Millisecond)

	waitMinRoles.OnConsensusFinished(context.Background(), network.Report{PulseNumber: pulse.MinTimePulse + 10})
}
