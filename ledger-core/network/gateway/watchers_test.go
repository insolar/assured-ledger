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
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"
	"github.com/stretchr/testify/assert"
)

func TestPulseWatchdog(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Wait(time.Minute)
	defer mc.Finish()

	gw := network.NewGatewayMock(mc)

	wd := newPulseWatchdog(context.Background(), gw, 300*time.Millisecond)
	wd.Reset()
	<-time.After(200 * time.Millisecond)
	wd.Reset()
	<-time.After(200 * time.Millisecond)
	defer wd.Stop()
}

func TestPulseWatchdog_timeout_exceeded(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Wait(time.Minute)
	defer mc.Finish()

	gw := network.NewGatewayMock(mc)
	gw.FailStateMock.Set(func(ctx context.Context, reason string) {
		assert.Equal(t, "New valid pulse timeout exceeded", reason)
	})

	wd := newPulseWatchdog(context.Background(), gw, 300*time.Millisecond)
	wd.Reset()
	<-time.After(400 * time.Millisecond)
	defer wd.Stop()
}
