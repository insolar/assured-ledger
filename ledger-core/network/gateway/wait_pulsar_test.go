// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	node2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	mock "github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"
)

func createBase(mc *minimock.Controller) *Base {
	b := &Base{}

	op := mock.NewOriginProviderMock(mc)
	op.GetOriginMock.Set(func() node2.NetworkNode {
		return node.NewNode(gen.UniqueReference(), node2.StaticRoleVirtual, nil, "127.0.0.1:123", "")
	})

	aborter := network.NewAborterMock(mc)
	aborter.AbortMock.Set(func(ctx context.Context, reason string) {
		require.Contains(mc, reason, bootstrapTimeoutMessage)
	})

	b.Aborter = aborter
	b.OriginProvider = op
	return b
}

func TestWaitPulsar_PulseNotArrivedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	waitPulsar := newWaitPulsar(createBase(mc))
	assert.Equal(t, node2.WaitPulsar, waitPulsar.GetState())
	gatewayer := mock.NewGatewayerMock(mc)
	waitPulsar.Gatewayer = gatewayer
	gatewayer.GatewayMock.Set(func() network.Gateway {
		return waitPulsar
	})

	waitPulsar.bootstrapETA = time.Millisecond
	waitPulsar.bootstrapTimer = time.NewTimer(waitPulsar.bootstrapETA)

	waitPulsar.Run(context.Background(), *pulsestor.EphemeralPulse)
}

func TestWaitPulsar_PulseArrivedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state node2.NetworkState, pulse pulsestor.Pulse) {
		assert.Equal(t, node2.CompleteNetworkState, state)
	})

	pulseAccessor := mock.NewPulseAccessorMock(mc)
	pulseAccessor.GetPulseMock.Set(func(ctx context.Context, p1 pulse.Number) (p2 pulsestor.Pulse, err error) {
		p := *pulsestor.GenesisPulse
		p.PulseNumber += 10
		return p, nil
	})

	waitPulsar := newWaitPulsar(&Base{
		PulseAccessor: pulseAccessor,
	})
	waitPulsar.Gatewayer = gatewayer
	waitPulsar.bootstrapETA = time.Second * 2
	waitPulsar.bootstrapTimer = time.NewTimer(waitPulsar.bootstrapETA)

	go waitPulsar.Run(context.Background(), *pulsestor.EphemeralPulse)
	time.Sleep(100 * time.Millisecond)

	waitPulsar.OnConsensusFinished(context.Background(), network.Report{PulseNumber: pulse.MinTimePulse + 10})
}
