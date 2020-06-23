// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pool

import (
	"context"
	"io"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

type fakeConnection struct {
	io.ReadWriteCloser
}

func (fakeConnection) Read(p []byte) (n int, err error) {
	runtime.Goexit() // prevent watchRemoteClose from triggering and writing errors to log after test is finished
	return 0, nil
}

func (fakeConnection) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (fakeConnection) Close() error {
	return nil

}

func newTransportMock(t *testing.T) transport.StreamTransport {
	tr := network.NewStreamTransportMock(t)
	tr.DialMock.Set(func(p context.Context, p1 string) (r io.ReadWriteCloser, r1 error) {
		return fakeConnection{}, nil
	})
	return tr
}

func TestNewConnectionPool(t *testing.T) {
	instestlogger.SetTestOutput(t)

	ctx := context.Background()
	tr := newTransportMock(t)

	pool := NewConnectionPool(tr)

	h, err := host.NewHost("127.0.0.1:8080")
	h2, err := host.NewHost("127.0.0.1:4200")

	conn, err := pool.GetConnection(ctx, h)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	conn2, err := pool.GetConnection(ctx, h2)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	conn3, err := pool.GetConnection(ctx, h2)
	assert.NotNil(t, conn2)
	assert.Equal(t, conn2, conn3)

	pool.CloseConnection(ctx, h)
	pool.CloseConnection(ctx, h2)
	pool.Reset()
}
