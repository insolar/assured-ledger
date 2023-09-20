package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func TestPulseWatchdog(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Wait(time.Second * 10)
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
	defer mc.Wait(time.Second * 10)
	defer mc.Finish()

	gw := network.NewGatewayMock(mc)
	gw.FailStateMock.Set(func(ctx context.Context, reason string) {
		assert.Equal(t, "New valid pulse timeout exceeded", reason)
	})

	wd := newPulseWatchdog(context.Background(), gw, 30*time.Millisecond)
	wd.Reset()
	<-time.After(40 * time.Millisecond)
	defer wd.Stop()
}
