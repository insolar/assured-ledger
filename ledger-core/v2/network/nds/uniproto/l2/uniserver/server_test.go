// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

func TestServer(t *testing.T) {
	const Server1 = "127.0.0.1:10001"
	const Server2 = "127.0.0.1:10002"

	marshaller := &TestProtocolMarshaller{}

	vf := TestVerifierFactory{}
	sk := cryptkit.NewSignatureKey(longbits.Zero(testDigestSize), testSignatureMethod, cryptkit.PublicAsymmetricKey)

	var peerProfileFn OfflinePeerFactoryFunc
	peerProfileFn = func(peer *Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk)
		return nwapi.Address{}, nil
	}

	var dispatcher1 Dispatcher
	dispatcher1.RegisterProtocol(0, TestProtocolDescriptor, marshaller, marshaller)

	buffer1 := NewReceiveBuffer(5, 0, 2, &dispatcher1)
	buffer1.RunWorkers(1, false)

	ups1 := NewUnifiedServer(&buffer1, 2)
	ups1.SetConfig(ServerConfig{
		BindingAddress: Server1,
		UdpMaxSize:     1400,
		PeerLimit:      -1,
	})
	ups1.SetPeerFactory(peerProfileFn)
	ups1.SetSignatureFactory(vf)

	ups1.StartListen()
	dispatcher1.SetMode(uniproto.AllowAll)

	pm1 := ups1.PeerManager()
	_, err := pm1.AddHostID(pm1.Local().GetPrimary(), 1)
	require.NoError(t, err)

	var dispatcher2 Dispatcher
	dispatcher2.SetMode(uniproto.NewConnectionMode(0, 0))
	dispatcher2.RegisterProtocol(0, TestProtocolDescriptor, marshaller, marshaller)
	dispatcher2.Seal()

	ups2 := NewUnifiedServer(&dispatcher2, 2)
	ups2.SetConfig(ServerConfig{
		BindingAddress: Server2,
		UdpMaxSize:     1400,
		PeerLimit:      -1,
	})
	ups2.SetPeerFactory(peerProfileFn)
	ups2.SetSignatureFactory(vf)

	ups2.StartNoListen()

	pm2 := ups2.PeerManager()
	_, err = pm2.AddHostID(pm2.Local().GetPrimary(), 2)
	require.NoError(t, err)

	conn21, err := pm2.Manager().ConnectPeer(nwapi.NewHostPort(Server1))
	require.NoError(t, err)
	require.NotNil(t, conn21)
	require.NoError(t, conn21.Transport().EnsureConnect())

	t.Run("small", func(t *testing.T) {
		testStr := "short msg"
		msgBytes := marshaller.SerializeMsg(0, 0, pulse.MinTimePulse, testStr)

		require.NoError(t, conn21.Transport().UseSessionful(int64(len(msgBytes)), func(t l1.OutTransport) (bool, error) {
			return true, t.SendBytes(msgBytes)
		}))

		marshaller.Wait(0)
		marshaller.Count.Store(0)

		require.Equal(t, testStr, marshaller.LastMsg)
		require.Equal(t, pulse.Number(pulse.MinTimePulse), marshaller.LastPacket.PulseNumber)

		testStr += "2"
		require.NoError(t, conn21.SendPacket(uniproto.SessionfulSmall, &TestPacket{testStr}))

		marshaller.Wait(0)
		marshaller.Count.Store(0)

		require.Equal(t, testStr, marshaller.LastMsg)
		require.Equal(t, pulse.Number(pulse.MinTimePulse), marshaller.LastPacket.PulseNumber)
	})

	t.Run("large", func(t *testing.T) {

		testStr := strings.Repeat("long msg", 6553)
		msgBytes := marshaller.SerializeMsg(0, 0, pulse.MinTimePulse, testStr)

		require.NoError(t, conn21.Transport().UseSessionful(int64(len(msgBytes)), func(t l1.OutTransport) (bool, error) {
			return true, t.SendBytes(msgBytes)
		}))

		marshaller.Wait(0)
		marshaller.Count.Store(0)

		require.Equal(t, testStr, marshaller.LastMsg)
		require.Equal(t, pulse.Number(pulse.MinTimePulse), marshaller.LastPacket.PulseNumber)

		testStr += "2"
		require.NoError(t, conn21.SendPacket(uniproto.SessionfulLarge, &TestPacket{testStr}))

		marshaller.Wait(0)
		marshaller.Count.Store(0)

		require.Equal(t, testStr, marshaller.LastMsg)
		require.Equal(t, pulse.Number(pulse.MinTimePulse), marshaller.LastPacket.PulseNumber)
	})
}

func TestHTTPLikeness(t *testing.T) {
	h := uniproto.Header{}
	require.Equal(t, uniproto.ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("GET /0123456789ABCDEF")))

	require.Equal(t, uniproto.ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("PUT /0123456789ABCDEF")))

	require.Equal(t, uniproto.ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("POST /0123456789ABCDEF")))
	require.Equal(t, uniproto.ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("POST 0123456789ABCDEF")))

	require.Equal(t, uniproto.ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("HEAD /0123456789ABCDEF")))
	require.Equal(t, uniproto.ErrPossibleHTTPRequest, h.DeserializeMinFromBytes([]byte("HEAD 0123456789ABCDEF")))
}
