// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hostnetwork

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func TestNewStreamHandler(t *testing.T) {
	t.Skip("fixme")
	defer testutils.LeakTester(t)

	ctx := instestlogger.TestContext(t)

	requestHandler := func(ctx context.Context, p *packet.ReceivedPacket) {
		inslogger.FromContext(ctx).Info("requestHandler")
	}

	h := NewStreamHandler(requestHandler, nil)

	con1, _ := net.Pipe()

	done := make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		h.HandleStream(ctx, con1)
		done <- struct{}{}
	}()

	cancel()
	// con2.Close()

	select {
	case <-done:
		return
	case <-time.After(time.Second * 5):
		t.Fail()
	}
}
