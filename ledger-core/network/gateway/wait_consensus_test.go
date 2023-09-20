package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/mocklog"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func TestWaitConsensus_ConsensusNotHappenedInETA(t *testing.T) {
	mc := minimock.NewController(mocklog.T(t))
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	waitConsensus := newWaitConsensus(createBase(mc))
	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.GatewayMock.Return(waitConsensus)

	waitConsensus.Gatewayer = gatewayer
	waitConsensus.bootstrapETA = time.Millisecond
	waitConsensus.bootstrapTimer = time.NewTimer(waitConsensus.bootstrapETA)

	waitConsensus.Run(context.Background(), EphemeralPulse.Data)
}

func TestWaitConsensus_ConsensusHappenedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state network.State, pulse pulse.Data) {
		assert.Equal(t, network.WaitMajority, state)
	})

	waitConsensus := newWaitConsensus(&Base{})
	assert.Equal(t, network.WaitConsensus, waitConsensus.GetState())
	waitConsensus.Gatewayer = gatewayer
	waitConsensus.bootstrapETA = time.Second
	waitConsensus.bootstrapTimer = time.NewTimer(waitConsensus.bootstrapETA)
	waitConsensus.OnConsensusFinished(context.Background(), network.Report{
		PulseData:   EphemeralPulse.Data,
		PulseNumber: EphemeralPulse.Data.PulseNumber,
	})

	waitConsensus.Run(context.Background(), EphemeralPulse.Data)
}
