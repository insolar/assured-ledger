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
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
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

	waitConsensus.Run(context.Background(), EphemeralPulse.Data)
}

func TestWaitConsensus_ConsensusHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state nodeinfo.NetworkState, pulse pulse.Data) {
		assert.Equal(t, nodeinfo.WaitMajority, state)
	})

	waitConsensus := newWaitConsensus(&Base{})
	assert.Equal(t, nodeinfo.WaitConsensus, waitConsensus.GetState())
	waitConsensus.Gatewayer = gatewayer
	waitConsensus.bootstrapETA = time.Second
	waitConsensus.bootstrapTimer = time.NewTimer(waitConsensus.bootstrapETA)
	waitConsensus.OnConsensusFinished(context.Background(), network.Report{
		PulseData:   EphemeralPulse.Data,
		PulseNumber: EphemeralPulse.Data.PulseNumber,
	})

	waitConsensus.Run(context.Background(), EphemeralPulse.Data)
}
