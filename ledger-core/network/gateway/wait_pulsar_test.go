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

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func createBase(mc *minimock.Controller) *Base {
	b := &Base{}

	aborter := network.NewAborterMock(mc)
	aborter.AbortMock.Set(func(ctx context.Context, reason string) {
		require.Contains(mc, reason, bootstrapTimeoutMessage)
	})

	nk := beat.NewNodeKeeperMock(mc)
	ref := gen.UniqueGlobalRef()
	nk.GetLocalNodeReferenceMock.Return(ref)
	nk.GetLocalNodeRoleMock.Return(member.PrimaryRoleVirtual)
	nk.FindAnyLatestNodeSnapshotMock.Return(nil)

	// avoid errors when these methods were not used
	nk.FindAnyLatestNodeSnapshot()
	nk.GetLocalNodeReference()
	nk.GetLocalNodeRole()

	b.NodeKeeper = nk
	b.Aborter = aborter
	return b
}

func TestWaitPulsar_PulseNotArrivedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	waitPulsar := newWaitPulsar(createBase(mc))
	assert.Equal(t, network.WaitPulsar, waitPulsar.GetState())
	gatewayer := mock.NewGatewayerMock(mc)
	waitPulsar.Gatewayer = gatewayer
	gatewayer.GatewayMock.Set(func() network.Gateway {
		return waitPulsar
	})

	waitPulsar.bootstrapETA = time.Millisecond
	waitPulsar.bootstrapTimer = time.NewTimer(waitPulsar.bootstrapETA)

	waitPulsar.Run(context.Background(), EphemeralPulse.Data)
}

func TestWaitPulsar_PulseArrivedInETA(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second*10)

	gatewayer := mock.NewGatewayerMock(mc)
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state network.State, pulse pulse.Data) {
		assert.Equal(t, network.CompleteNetworkState, state)
	})

	waitPulsar := newWaitPulsar(&Base{})
	waitPulsar.Gatewayer = gatewayer
	waitPulsar.bootstrapETA = time.Second * 20
	waitPulsar.bootstrapTimer = time.NewTimer(waitPulsar.bootstrapETA)

	go waitPulsar.Run(context.Background(), EphemeralPulse.Data)
	time.Sleep(100 * time.Millisecond)

	pulseNumber := pulse.OfNow()
	waitPulsar.OnConsensusFinished(context.Background(), network.Report{
		PulseNumber: pulseNumber,
		PulseData: pulse.Data{
			PulseNumber: pulseNumber,
			DataExt:     pulse.DataExt{PulseEpoch: pulse.Epoch(pulseNumber)},
		},
	})
}
