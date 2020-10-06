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

	cer, err := tls.LoadX509KeyPair("testdata/server.rsa.crt", "testdata/server.rsa.key")
	require.NoError(t, err)

	cfg := &tls.Config{Certificates: []tls.Certificate{cer}, InsecureSkipVerify: true}
	assert.NotNil(t, cfg)
	tp := NewTLS(nwapi.NewHostPort("127.0.0.1:0", true), nwapi.PreferV4, cfg)
	defer tp.Close()

	conns := make(chan (io.Reader))
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

func TestNewTCP(t *testing.T) {
	tp := NewTCP(nwapi.NewHostPort("127.0.0.1:0", true), nwapi.PreferV4)
	defer tp.Close()

	conns := make(chan (io.Reader))
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
	defer tp.Close()

	assert.Equal(t, MaxUDPSize, int(tp.MaxByteSize()))

	fn := func(local, remote nwapi.Address, b []byte, err error) (ok bool) {
		return true
	}

	outTransport, addr, err := tp.CreateListeningFactory(fn)
	assert.NoError(t, err)
	require.NotNil(t, outTransport)
	assert.False(t, addr.IsZero())
	require.NotEqual(t, 0, addr.AsUDPAddr().Port)

	tp2 := NewUDP(nwapi.NewHostPort("127.0.0.1:10000", false), nwapi.PreferV4, MaxUDPSize)
	outTransport2, err := tp2.CreateOutgoingOnlyFactory()
	assert.NoError(t, err)
	require.NotNil(t, outTransport2)
	defer outTransport2.Close()

	transport, err := outTransport2.ConnectTo(addr)
	assert.NoError(t, err)

	err = transport.SendBytes([]byte(helloMsg))
	assert.NoError(t, err)

}

func TestSessionfulTransportProvider_CreateListeningFactory_Negative(t *testing.T) {

	cer, err := tls.LoadX509KeyPair("testdata/server.rsa.crt", "testdata/server.rsa.key")
	require.NoError(t, err)

	cfg := &tls.Config{Certificates: []tls.Certificate{cer}, InsecureSkipVerify: true}
	assert.NotNil(t, cfg)

	tests := []struct {
		name     string
		provider SessionfulTransportProvider
		fn       SessionfulConnectFunc
	}{
		{
			name:     "tcp",
			provider: NewTCP(nwapi.NewHostPort("127.0.0.1:0", true), nwapi.PreferV4),
			fn:       nil,
		},
		{
			name:     "tls",
			provider: NewTLS(nwapi.NewHostPort("127.0.0.1:0", true), nwapi.PreferV4, cfg),
			fn:       nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Panics(t, func() {
				_, _, _ = test.provider.CreateListeningFactory(test.fn)
			})

		})
	}
}
