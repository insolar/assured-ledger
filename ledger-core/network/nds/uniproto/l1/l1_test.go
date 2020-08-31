// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l1

import (
	"crypto/tls"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

const helloMsg = "hello"

func TestNewTLS(t *testing.T) {

	cer, err  := tls.LoadX509KeyPair("testdata/server.rsa.crt", "testdata/server.rsa.key")
	require.NoError(t, err)

	cfg := &tls.Config{Certificates: []tls.Certificate{cer}, InsecureSkipVerify: true}
	assert.NotNil(t, cfg)
	tp := NewTLS(nwapi.NewHostPort("127.0.0.1:0", true), nwapi.PreferV4, cfg)

	conns := make(chan(io.Reader))
	fn := func(local, remote nwapi.Address, conn io.ReadWriteCloser, w OneWayTransport, err error) (ok bool) {
		conns <- conn
		return true
	}
	outTransport, addr, err := tp.CreateListeningFactory(fn)
	assert.NoError(t, err)
	require.NotNil(t, outTransport)
	defer outTransport.Close()
	assert.False(t, addr.IsZero())

	outTransport2, err := tp.CreateOutgoingOnlyFactory(fn)
	assert.NoError(t, err)
	require.NotNil(t, outTransport2)
	defer outTransport2.Close()

	transport, err := outTransport2.ConnectTo(addr)
	assert.NoError(t, err)

	err = transport.SendBytes([]byte(helloMsg))
	assert.NoError(t, err)

	buf := make([]byte, len(helloMsg))
	_, err = io.ReadAtLeast(<-conns, buf, len(helloMsg))
	assert.NoError(t, err)
	assert.Equal(t, helloMsg, string(buf))
}

func TestNewUDP(t *testing.T) {
	tp := NewUDP(nwapi.NewHostPort("127.0.0.1:0", true), nwapi.PreferV4, MaxUDPSize)
	assert.Equal(t, MaxUDPSize, int(tp.MaxByteSize()))

	fn := func(local, remote nwapi.Address, b []byte, err error) (ok bool) {
		return true
	}

	outTransport, addr, err := tp.CreateListeningFactory(fn)
	assert.NoError(t, err)
	require.NotNil(t, outTransport)
	assert.False(t, addr.IsZero())
	require.NotEqual(t, 0, addr.AsUDPAddr().Port)


	outTransport2, err := tp.CreateOutgoingOnlyFactory()
	assert.NoError(t, err)
	require.NotNil(t, outTransport2)
	defer outTransport2.Close()

	transport, err := outTransport2.ConnectTo(addr)
	assert.NoError(t, err)

	err = transport.SendBytes([]byte(helloMsg))
	assert.NoError(t, err)

}
