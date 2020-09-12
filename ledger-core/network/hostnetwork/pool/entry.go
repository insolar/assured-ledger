// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pool

import (
	"context"
	"io"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/network"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
)

type onClose func(ctx context.Context, host *legacyhost.Host)

type entry struct {
	sync.Mutex
	transport transport.StreamTransport
	host      *legacyhost.Host
	onClose   onClose
	conn      io.ReadWriteCloser
}

func newEntry(t transport.StreamTransport, conn io.ReadWriteCloser, host *legacyhost.Host, onClose onClose) *entry {
	return &entry{
		transport: t,
		conn:      conn,
		host:      host,
		onClose:   onClose,
	}
}

func (e *entry) watchRemoteClose(ctx context.Context) {
	b := make([]byte, 1)
	_, err := e.conn.Read(b)
	if err != nil {
		inslogger.FromContext(ctx).Infof("[ watchRemoteClose ] remote host 'closed' connection to %s: %s", e.host.String(), err)
		e.onClose(ctx, e.host)
		return
	}

	inslogger.FromContext(ctx).Errorf("[ watchRemoteClose ] unexpected data on connection to %s", e.host.String())
}

func (e *entry) open(ctx context.Context) (io.ReadWriteCloser, error) {
	e.Lock()
	defer e.Unlock()
	if e.conn != nil {
		return e.conn, nil
	}

	conn, err := e.dial(ctx)
	if err != nil {
		return nil, err
	}

	e.conn = conn
	go e.watchRemoteClose(ctx)
	return e.conn, nil
}

func (e *entry) dial(ctx context.Context) (io.ReadWriteCloser, error) {
	conn, err := e.transport.Dial(ctx, e.host.Address.String())
	if err != nil {
		return nil, errors.W(err, "[ Open ] Failed to create TCP connection")
	}

	return conn, nil
}

func (e *entry) close() {
	e.Lock()
	defer e.Unlock()

	if e.conn != nil {
		network.CloseVerbose(e.conn)
	}
}
