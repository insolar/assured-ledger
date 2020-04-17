// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
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
	peerProfileFn = func(peer *Peer) error {
		peer.SetSignatureKey(sk)
		return nil
	}

	var protocols uniproto.Parser
	protocols.SignatureSizeHint = 32

	protocols.Protocols[0] = TestProtocolDescriptor
	protocols.Protocols[0].Receiver = marshaller

	ups1 := NewUnifiedProtocolServer(&protocols, 2)
	ups1.SetConfig(ServerConfig{
		BindingAddress: Server1,
		UdpMaxSize:     1400,
		PeerLimit:      -1,
	})
	ups1.SetPeerFactory(peerProfileFn)
	ups1.SetSignatureFactory(vf)

	ups1.StartListen()
	ups1.SetMode(AllowAll)

	pm1 := ups1.PeerManager()
	pm1.AddHostId(pm1.Local().GetPrimary(), 1)

	ups2 := NewUnifiedProtocolServer(&protocols, 2)
	ups2.SetConfig(ServerConfig{
		BindingAddress: Server2,
		UdpMaxSize:     1400,
		PeerLimit:      -1,
	})
	ups2.SetPeerFactory(peerProfileFn)
	ups2.SetSignatureFactory(vf)

	ups2.StartNoListen()

	pm2 := ups2.PeerManager()
	pm2.AddHostId(pm2.Local().GetPrimary(), 2)

	conn21, err := pm2.ConnectTo(nwapi.NewHostPort(Server1))
	require.NoError(t, err)
	require.NotNil(t, conn21)
	require.NoError(t, conn21.EnsureConnect())

	testStr := "short msg"
	msgBytes := marshaller.SerializeMsg(0, 0, pulse.MinTimePulse, testStr)

	require.NoError(t, conn21.UseSessionful(int64(len(msgBytes)), func(t l1.OutTransport) (bool, error) {
		return true, t.SendBytes(msgBytes)
	}))

	marshaller.Wait(0)

	require.Equal(t, testStr, marshaller.LastMsg)
	require.Equal(t, pulse.Number(pulse.MinTimePulse), marshaller.LastPacket.PulseNumber)

	testStr = strings.Repeat("long msg", 6553)
	msgBytes = marshaller.SerializeMsg(0, 0, pulse.MinTimePulse, testStr)

	require.NoError(t, conn21.UseSessionful(int64(len(msgBytes)), func(t l1.OutTransport) (bool, error) {
		return true, t.SendBytes(msgBytes)
	}))

	marshaller.Wait(1)

	require.Equal(t, testStr, marshaller.LastMsg)
	require.Equal(t, pulse.Number(pulse.MinTimePulse), marshaller.LastPacket.PulseNumber)
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
