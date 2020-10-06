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

	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func TestNewGatewayer(t *testing.T) {
	instestlogger.SetTestOutput(t)

	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Second * 10)

	gw := mock.NewGatewayMock(mc)

	gw.GetStateMock.Return(network.NoNetworkState)

	gw.NewGatewayMock.Set(func(ctx context.Context, s network.State) (g1 network.Gateway) {
		assert.Equal(t, network.WaitConsensus, s)
		return gw
	})

	gw.RunMock.Return()

	gatewayer := NewGatewayer(gw)
	assert.Equal(t, gw, gatewayer.Gateway())
	assert.Equal(t, network.NoNetworkState, gatewayer.Gateway().GetState())

	gatewayer.SwitchState(context.Background(), network.WaitConsensus, pulsestor.GenesisPulse.Data)
}
