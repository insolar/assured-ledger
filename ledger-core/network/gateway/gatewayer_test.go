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

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/network"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func TestNewGatewayer(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	gw := mock.NewGatewayMock(mc)

	gw.GetStateMock.Set(func() (n1 node.NetworkState) {
		return node.NoNetworkState
	})

	gw.NewGatewayMock.Set(func(ctx context.Context, s node.NetworkState) (g1 network.Gateway) {
		assert.Equal(t, node.WaitConsensus, s)
		return gw
	})

	gw.BeforeRunMock.Set(func(ctx context.Context, pulse pulsestor.Pulse) {
	})

	gw.RunMock.Set(func(ctx context.Context, pulse pulsestor.Pulse) {
	})

	gatewayer := NewGatewayer(gw)
	assert.Equal(t, gw, gatewayer.Gateway())
	assert.Equal(t, node.NoNetworkState, gatewayer.Gateway().GetState())

	gatewayer.SwitchState(context.Background(), node.WaitConsensus, *pulsestor.GenesisPulse)
}
