// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestServerWrappedFactory(t *testing.T) {
	const Server1 = "127.0.0.1:0"
	const Server2 = "127.0.0.1:0"

	marshaller := &TestProtocolMarshaller{}

	vf := TestVerifierFactory{}
	sk := cryptkit.NewSigningKey(longbits.Zero(testDigestSize), testSigningMethod, cryptkit.PublicAsymmetricKey)

	var peerProfileFn PeerMapperFunc
	peerProfileFn = func(peer *Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk)
		return nwapi.Address{}, nil
	}

	var dispatcher1 Dispatcher
	dispatcher1.RegisterProtocol(0, TestProtocolDescriptor, marshaller, marshaller)

	buffer1 := NewReceiveBuffer(5, 0, 2, &dispatcher1)
	buffer1.RunWorkers(1, false)

	ups1 := NewUnifiedServer(&buffer1, TestLogAdapter{t})
	ups1.SetConfig(ServerConfig{
		BindingAddress: Server1,
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	})
	ups1.SetPeerFactory(peerProfileFn)
	ups1.SetSignatureFactory(vf)

	ch := make(chan string, 3)

	provider := MapTransportProvider(defaultProvider,
		func(lessProvider l1.SessionlessTransportProvider) l1.SessionlessTransportProvider {
			return l1.MapSessionlessProvider(lessProvider, func(factory l1.OutTransportFactory) l1.OutTransportFactory {
				return l1.MapOutputFactory(factory, func(transport l1.OneWayTransport) l1.OneWayTransport {
					return &TestOneWayTransport{transport, ch}
				})
			})
		},
		func(fullProvider l1.SessionfulTransportProvider) l1.SessionfulTransportProvider {
			return l1.MapSessionFullProvider(fullProvider, func(factory l1.OutTransportFactory) l1.OutTransportFactory {
				return l1.MapOutputFactory(factory, func(transport l1.OneWayTransport) l1.OneWayTransport {
					return &TestOneWayTransport{transport, ch}
				})
			})
		})

	ups1.SetTransportProvider(provider)

	ups1.StartListen()
	dispatcher1.SetMode(uniproto.AllowAll)

	pm1 := ups1.PeerManager()
	_, err := pm1.AddHostID(pm1.Local().GetPrimary(), 1)
	require.NoError(t, err)

	var dispatcher2 Dispatcher
	dispatcher2.SetMode(uniproto.NewConnectionMode(0, 0))
	dispatcher2.RegisterProtocol(0, TestProtocolDescriptor, marshaller, marshaller)
	dispatcher2.Seal()

	ups2 := NewUnifiedServer(&dispatcher2, TestLogAdapter{t})
	ups2.SetConfig(ServerConfig{
		BindingAddress: Server2,
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	})
	ups2.SetPeerFactory(peerProfileFn)
	ups2.SetSignatureFactory(vf)
	ups2.SetTransportProvider(provider)

	ups2.StartListen()

	pm2 := ups2.PeerManager()
	_, err = pm2.AddHostID(pm2.Local().GetPrimary(), 2)
	require.NoError(t, err)

	conn21, err := pm2.Manager().ConnectPeer(pm1.Local().GetPrimary())
	require.NoError(t, err)
	require.NotNil(t, conn21)
	require.NoError(t, conn21.Transport().EnsureConnect())

	t.Run("sessionless", func(t *testing.T) {
		testStr := "sessionless msg"
		require.NoError(t, conn21.SendPacket(uniproto.Sessionless, &TestPacket{testStr}))

		method := <-ch

		require.Equal(t, "SendBytes", method)
	})

	t.Run("sessionfull", func(t *testing.T) {
		testStr := "sessionfull msg"

		require.NoError(t, conn21.SendPacket(uniproto.SessionfulSmall, &TestPacket{testStr}))

		method := <-ch
		require.Equal(t, "SendBytes", method)

		// extend message len for using Send instead SendBytes
		for i := 0; i < uniproto.MaxNonExcessiveLength; i++ {
			testStr += "-"
		}

		require.NoError(t, conn21.SendPacket(uniproto.SessionfulLarge, &TestPacket{testStr}))

		// if package is large, we try SendBytes with pre buffer first
		method = <-ch
		require.Equal(t, "SendBytes", method)
		// and when send another bytes
		method = <-ch
		require.Equal(t, "Send", method)
		// and when send signature, just read
		method = <-ch
		require.Equal(t, "Send", method)
	})
}

type TestOneWayTransport struct {
	l1.OneWayTransport
	ch chan string
}

func (t TestOneWayTransport) Send(payload io.WriterTo) error {
	t.ch <- "Send"
	return t.OneWayTransport.Send(payload)
}

func (t TestOneWayTransport) SendBytes(b []byte) error {
	t.ch <- "SendBytes"
	return t.OneWayTransport.SendBytes(b)
}
