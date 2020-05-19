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

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	pulse2 "github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	mock "github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"
)

func TestWaitConsensus_ConsensusNotHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	waitConsensus := newWaitConsensus(createBase(mc))
	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.GatewayMock.Set(func() network.Gateway {
		return waitConsensus
	})
	waitConsensus.Gatewayer = gatewayer
	waitConsensus.bootstrapETA = time.Millisecond
	waitConsensus.bootstrapTimer = time.NewTimer(waitConsensus.bootstrapETA)

	waitConsensus.Run(context.Background(), *pulse.EphemeralPulse)
}

func TestWaitConsensus_ConsensusHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state node.NetworkState, pulse pulse.Pulse) {
		assert.Equal(t, node.WaitMajority, state)
	})

	waitConsensus := newWaitConsensus(&Base{})
	assert.Equal(t, node.WaitConsensus, waitConsensus.GetState())
	waitConsensus.Gatewayer = gatewayer
	accessorMock := mock.NewPulseAccessorMock(mc)
	accessorMock.GetPulseMock.Set(func(ctx context.Context, p1 pulse2.Number) (p2 pulse.Pulse, err error) {
		return *pulse.EphemeralPulse, nil
	})
	waitConsensus.PulseAccessor = accessorMock
	waitConsensus.bootstrapETA = time.Second
	waitConsensus.bootstrapTimer = time.NewTimer(waitConsensus.bootstrapETA)
	waitConsensus.OnConsensusFinished(context.Background(), network.Report{})

	waitConsensus.Run(context.Background(), *pulse.EphemeralPulse)
}
