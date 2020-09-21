// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

var noopReceiver = func(_ ReturnAddress, _ nwapi.PayloadCompleteness, _ interface{}) error {
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func rndBytes(n int) []byte {
	key := make([]byte, n)

	rand.Read(key)

	return key
}

func TestShipToHead(t *testing.T) {
	payloadLen := 64

	head := TestString{string(rndBytes(payloadLen))}

	sh := Shipment{
		Head: &head,
	}

	var _, srv2 Service

	ch1 := make(chan string, 1)
	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		ch1 <- vo

		return nil
	}

	_, srv2, stop := startUniprotoServers(t, recv1, noopReceiver)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	expPayload := head.S
	actlPayload := <-ch1

	require.Equal(t, expPayload, actlPayload)
}

func TestShipToBody(t *testing.T) {
	payloadLen := 1024 * 1024 * 64

	body := TestString{string(rndBytes(payloadLen))}

	sh := Shipment{
		Body: &body,
	}

	var _, srv2 Service

	ch1 := make(chan string, 1)
	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		ch1 <- vo

		return nil
	}

	_, srv2, stop := startUniprotoServers(t, recv1, noopReceiver)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	expPayload := body.S
	actlPayload := <-ch1

	require.Equal(t, expPayload, actlPayload)
}

func TestShipToHeadAndBody(t *testing.T) {
	payloadLen := 1024 * 1024 * 64

	bytes := rndBytes(payloadLen)

	head := TestString{string(bytes[:64])}
	body := TestString{string(bytes)}

	sh := Shipment{
		Head: &head,
		Body: &body,
	}

	var srv1, srv2 Service

	ch1 := make(chan string, 2)

	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.False(t, bool(done))

		// Save received head
		ch1 <- vo

		err := srv1.PullBody(a, ShipmentRequest{
			ReceiveFn: func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
				vo := v.(fmt.Stringer).String()
				t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

				require.True(t, bool(done))

				// Save received body
				ch1 <- vo
				return nil
			},
		})

		require.NoError(t, err)

		return nil
	}

	srv1, srv2, stop := startUniprotoServers(t, recv1, noopReceiver)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	expHeadPayload := head.S
	expBodyPayload := body.S
	actlHeadPayload := <-ch1
	actlBodyPayload := <-ch1

	require.Equal(t, expHeadPayload, actlHeadPayload)
	require.Equal(t, expBodyPayload, actlBodyPayload)
}

func TestEchoHead(t *testing.T) {
	payloadLen := 64

	head := TestString{string(rndBytes(payloadLen))}

	sh := Shipment{
		Head: &head,
	}

	var srv1, srv2 Service

	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		err := srv1.ShipReturn(a, Shipment{
			Head: &TestString{S: vo + "echo1"},
		})

		require.NoError(t, err)

		return nil
	}

	ch2 := make(chan string, 1)

	recv2 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-2:"), len(vo))

		require.True(t, bool(done))

		ch2 <- vo

		return nil
	}

	srv1, srv2, stop := startUniprotoServers(t, recv1, recv2)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	expPayload := head.S + "echo1"
	actlPayload := <-ch2

	require.Equal(t, expPayload, actlPayload)
}

func TestEchoHeadAndBody(t *testing.T) {
	payloadLen := 1024 * 1024 * 64

	bytes := rndBytes(payloadLen)

	head := TestString{string(bytes[:64])}
	body := TestString{string(bytes)}

	sh := Shipment{
		Head: &head,
		Body: &body,
	}

	var srv1, srv2 Service

	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.False(t, bool(done))

		echoHead := TestString{S: vo + "echo1"}

		err := srv1.PullBody(a, ShipmentRequest{
			ReceiveFn: func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
				vo := v.(fmt.Stringer).String()

				require.True(t, bool(done))

				err := srv1.ShipReturn(a, Shipment{
					Head: &echoHead,
					Body: &TestString{S: vo + "echo1"},
				})

				require.NoError(t, err)

				return nil
			},
		})

		require.NoError(t, err)

		return nil
	}

	ch2 := make(chan string, 2)

	recv2 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-2:"), len(vo))

		require.False(t, bool(done))

		ch2 <- vo

		err := srv2.PullBody(a, ShipmentRequest{
			ReceiveFn: func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
				vo := v.(fmt.Stringer).String()
				t.Log(a.String(), fmt.Sprintf("ctrl-2:"), len(vo))

				require.True(t, bool(done))

				ch2 <- vo

				return nil
			},
		})

		require.NoError(t, err)

		return nil
	}

	srv1, srv2, stop := startUniprotoServers(t, recv1, recv2)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	expHeadPayload := head.S + "echo1"
	expBodyPayload := body.S + "echo1"
	actlHeadPayload := <-ch2
	actlBodyPayload := <-ch2

	require.Equal(t, expHeadPayload, actlHeadPayload)
	require.Equal(t, expBodyPayload, actlBodyPayload)
}

func TestShipToCancel(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-798")

	payloadLen := 64

	head := TestString{string(rndBytes(payloadLen))}

	ch := synckit.NewChainedCancel()
	sh := Shipment{
		Head:   &head,
		Cancel: ch,
	}

	ch.Cancel()

	var _, srv2 Service

	ch1 := make(chan string, 1)
	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		ch1 <- vo

		return nil
	}

	_, srv2, stop := startUniprotoServers(t, recv1, noopReceiver)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	_, ok := <-ch1

	require.False(t, ok)
}

func TestShipReturnCancel(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-798")

	payloadLen := 64

	head := TestString{string(rndBytes(payloadLen))}

	sh := Shipment{
		Head: &head,
	}

	var srv1, srv2 Service

	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		ch := synckit.NewChainedCancel()
		ch.Cancel()

		err := srv1.ShipReturn(a, Shipment{
			Head:   &TestString{S: vo + "echo1"},
			Cancel: ch,
		})

		require.NoError(t, err)

		return nil
	}

	ch2 := make(chan string, 1)

	recv2 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-2:"), len(vo))

		require.True(t, bool(done))

		ch2 <- vo

		return nil
	}

	srv1, srv2, stop := startUniprotoServers(t, recv1, recv2)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	_, ok := <-ch2

	require.False(t, ok)
}

func TestPullBodyCancel(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-798")

	payloadLen := 1024 * 1024 * 64

	bytes := rndBytes(payloadLen)

	head := TestString{string(bytes[:64])}
	body := TestString{string(bytes)}

	sh := Shipment{
		Head: &head,
		Body: &body,
	}

	var srv1, srv2 Service

	ch1 := make(chan string, 1)

	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.False(t, bool(done))

		ch := synckit.NewChainedCancel()
		ch.Cancel()

		err := srv1.PullBody(a, ShipmentRequest{
			ReceiveFn: func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
				vo := v.(fmt.Stringer).String()
				t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

				require.True(t, bool(done))

				ch1 <- vo
				return nil
			},
			Cancel: ch,
		})

		require.NoError(t, err)

		return nil
	}

	srv1, srv2, stop := startUniprotoServers(t, recv1, noopReceiver)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	_, ok := <-ch1

	require.False(t, ok)
}

func TestRejectBody(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-799")

	payloadLen := 1024 * 1024 * 512

	bytes := rndBytes(payloadLen)

	head := TestString{string(bytes[:64])}
	body := TestString{string(bytes)}

	sh := Shipment{
		Head: &head,
		Body: &body,
	}

	var srv1, srv2 Service

	ch1 := make(chan string, 2)

	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.False(t, bool(done))

		err := srv1.PullBody(a, ShipmentRequest{
			ReceiveFn: func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
				vo := v.(fmt.Stringer).String()
				t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

				require.True(t, bool(done))

				ch1 <- vo
				return nil
			},
		})
		require.NoError(t, err)

		err = srv1.RejectBody(a)
		require.NoError(t, err)

		err = srv1.PullBody(a, ShipmentRequest{
			ReceiveFn: noopReceiver,
		})

		// TODO err maybe in ReceiveFn
		require.Error(t, err)

		// Save received head
		ch1 <- vo

		return nil
	}

	srv1, srv2, stop := startUniprotoServers(t, recv1, noopReceiver)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	expHeadPayload := head.S
	actlHeadPayload := <-ch1

	require.Equal(t, expHeadPayload, actlHeadPayload)
}

func TestShipReturnWithTTL(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-800")
}

func startUniprotoServers(t *testing.T, recv1, recv2 ReceiverFunc) (Service, Service, func()) {
	const Server1 = "127.0.0.1:0"
	const Server2 = "127.0.0.1:0"

	vf := TestVerifierFactory{}
	skBytes := [testDigestSize]byte{}
	sk1 := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1
	sk2 := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)

	var srv1 Service
	ctrl1 := NewController(
		Protocol,
		TestDeserializationFactory{},
		recv1,
		nil,
		TestLogAdapter{t},
	)

	srv1 = ctrl1.NewFacade()

	var dispatcher1 uniserver.Dispatcher
	ctrl1.RegisterWith(dispatcher1.RegisterProtocol)

	ups1 := uniserver.NewUnifiedServer(&dispatcher1, TestLogAdapter{t})
	ups1.SetConfig(uniserver.ServerConfig{
		BindingAddress: Server1,
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	})

	ups1.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk2)
		peer.SetNodeID(2)
		return nwapi.NewHostID(2), nil
	})
	ups1.SetSignatureFactory(vf)

	ups1.StartListen()
	dispatcher1.SetMode(uniproto.AllowAll)

	pm1 := ups1.PeerManager()
	_, err := pm1.AddHostID(pm1.Local().GetPrimary(), 1)
	require.NoError(t, err)

	pr := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{}))
	dispatcher1.NextPulse(pr)

	/********************************/

	ctrl2 := NewController(
		Protocol,
		TestDeserializationFactory{},
		recv2,
		nil,
		TestLogAdapter{t},
	)

	srv2 := ctrl2.NewFacade()

	var dispatcher2 uniserver.Dispatcher
	dispatcher2.SetMode(uniproto.NewConnectionMode(0, Protocol))
	ctrl2.RegisterWith(dispatcher2.RegisterProtocol)
	dispatcher2.Seal()

	ups2 := uniserver.NewUnifiedServer(&dispatcher2, TestLogAdapter{t})
	ups2.SetConfig(uniserver.ServerConfig{
		BindingAddress: Server2,
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	})

	ups2.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk1)
		peer.SetNodeID(1)
		return nwapi.NewHostID(1), nil
	})
	ups2.SetSignatureFactory(vf)

	ups2.StartListen()
	dispatcher2.NextPulse(pr)

	pm2 := ups2.PeerManager()
	_, err = pm2.AddHostID(pm2.Local().GetPrimary(), 2)
	require.NoError(t, err)

	conn21, err := pm2.Manager().ConnectPeer(pm1.Local().GetPrimary())
	require.NoError(t, err)
	require.NotNil(t, conn21)
	require.NoError(t, conn21.Transport().EnsureConnect())

	require.NoError(t, err)

	return srv1, srv2, func() {
		dispatcher2.Stop()
		dispatcher1.Stop()
	}
}
