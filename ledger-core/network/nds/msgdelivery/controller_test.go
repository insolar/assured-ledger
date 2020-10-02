// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
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

func TestManyServersEcho(t *testing.T) {
	numberServers := 50
	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	chRes := make(chan string)

	for idx := 1; idx <= numberServers; idx++ {
		servId := idx

		_, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
			self := h.server(servId)
			v := val.(fmt.Stringer).String()

			switch {
			case strings.HasPrefix(v, "echo"):
				chRes <- v
			default:
				sh := Shipment{
					Head: &TestString{fmt.Sprintf("echo%s<-%d", v, servId)},
				}

				err := self.service.ShipReturn(a, sh)
				require.NoError(t, err)
			}
			return nil
		})

		require.NoError(t, err)
	}

	for sendIdx := 1; sendIdx <= numberServers; sendIdx++ {
		srv := h.server(sendIdx)

		text := fmt.Sprintf("%d", sendIdx)
		sh := Shipment{
			Head: &TestString{text},
		}

		for recvIdx := 1; recvIdx <= numberServers; recvIdx++ {
			if sendIdx == recvIdx {
				continue
			}

			err := srv.service.ShipTo(NewDirectAddress(nwapi.ShortNodeID(recvIdx)), sh)
			require.NoError(t, err)
			res := <-chRes
			require.Equal(t, fmt.Sprintf("echo%s<-%d", text, recvIdx), res)
		}
	}
}

func TestShipToHead(t *testing.T) {
	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 64
	head := TestString{string(rndBytes(payloadLen))}
	sh := Shipment{
		Head: &head,
	}

	ch := make(chan string)

	srv1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		ch <- vo

		return nil
	})
	require.NoError(t, err)

	srv2, err := h.createService(cfg, noopReceiver)
	require.NoError(t, err)

	err = srv2.service.ShipTo(srv1.directAddress(), sh)
	require.NoError(t, err)

	require.Equal(t, head.S, <-ch)
}

func TestShipToBody(t *testing.T) {
	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 1024 * 1024 * 64
	body := TestString{string(rndBytes(payloadLen))}
	sh := Shipment{
		Body: &body,
	}

	ch := make(chan string)

	srv1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		ch <- vo

		return nil
	})
	require.NoError(t, err)

	srv2, err := h.createService(cfg, noopReceiver)
	require.NoError(t, err)

	err = srv2.service.ShipTo(srv1.directAddress(), sh)
	require.NoError(t, err)

	expPayload := body.S
	actlPayload := <-ch

	require.Equal(t, expPayload, actlPayload)
}

func TestShipToHeadAndBody(t *testing.T) {
	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 1024 * 64
	bytes := rndBytes(payloadLen)
	head := TestString{string(bytes[:64])}
	body := TestString{string(bytes)}
	sh := Shipment{
		Head: &head,
		Body: &body,
	}

	var srv1, srv2 Service
	ch1 := make(chan string, 10)

	server1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
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
	})
	require.NoError(t, err)

	srv1 = server1.service

	server2, err := h.createService(cfg, noopReceiver)
	require.NoError(t, err)

	srv2 = server2.service

	err = srv2.ShipTo(server1.directAddress(), sh)
	require.NoError(t, err)

	expHeadPayload := head.S
	expBodyPayload := body.S
	actlHeadPayload := <-ch1
	actlBodyPayload := <-ch1

	require.Equal(t, expHeadPayload, actlHeadPayload)
	require.Equal(t, expBodyPayload, actlBodyPayload)
}

func TestEchoHead(t *testing.T) {
	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 64
	head := TestString{string(rndBytes(payloadLen))}
	sh := Shipment{
		Head: &head,
	}

	var srv1 Service

	server1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		err := srv1.ShipReturn(a, Shipment{
			Head: &TestString{S: vo + "echo1"},
		})

		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)

	srv1 = server1.service

	ch2 := make(chan string)

	server2, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-2:"), len(vo))

		require.True(t, bool(done))

		ch2 <- vo

		return nil
	})
	require.NoError(t, err)

	err = server2.service.ShipTo(server1.directAddress(), sh)
	require.NoError(t, err)

	expPayload := head.S + "echo1"
	actlPayload := <-ch2

	require.Equal(t, expPayload, actlPayload)
}

func TestEchoHeadAndBody(t *testing.T) {
	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 1024 * 1024 * 64
	bytes := rndBytes(payloadLen)
	head := TestString{string(bytes[:64])}
	body := TestString{string(bytes)}
	sh := Shipment{
		Head: &head,
		Body: &body,
	}

	var srv1, srv2 Service

	server1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.False(t, bool(done))

		echoHead := TestString{S: vo + "echo1"}

		err := srv1.PullBody(a, ShipmentRequest{
			ReceiveFn: func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
				vo := v.(fmt.Stringer).String()
				t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

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
	})
	require.NoError(t, err)
	srv1 = server1.service

	ch2 := make(chan string, 2)

	server2, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
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
	})
	require.NoError(t, err)
	srv2 = server2.service

	err = srv2.ShipTo(server1.directAddress(), sh)
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

	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 64
	head := TestString{string(rndBytes(payloadLen))}
	ch := synckit.NewChainedCancel()
	sh := Shipment{
		Head:   &head,
		Cancel: ch,
	}

	ch.Cancel()

	ch1 := make(chan string)

	server1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		ch1 <- vo

		return nil
	})
	require.NoError(t, err)

	server2, err := h.createService(cfg, noopReceiver)
	require.NoError(t, err)

	err = server2.service.ShipTo(server1.directAddress(), sh)
	require.NoError(t, err)

	_, ok := <-ch1

	require.False(t, ok)
}

func TestShipReturnCancel(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-798")

	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 64
	head := TestString{string(rndBytes(payloadLen))}
	sh := Shipment{
		Head: &head,
	}

	var srv1 Service

	server1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
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
	})
	require.NoError(t, err)

	srv1 = server1.service

	ch2 := make(chan string)

	server2, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-2:"), len(vo))

		require.True(t, bool(done))

		ch2 <- vo

		return nil
	})
	require.NoError(t, err)

	err = server2.service.ShipTo(server1.directAddress(), sh)
	require.NoError(t, err)

	_, ok := <-ch2

	require.False(t, ok)
}

func TestPullBodyCancel(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-798")

	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 1024 * 1024 * 64
	bytes := rndBytes(payloadLen)
	head := TestString{string(bytes[:64])}
	body := TestString{string(bytes)}
	sh := Shipment{
		Head: &head,
		Body: &body,
	}

	var srv1 Service

	ch1 := make(chan string)

	server1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
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
	})
	require.NoError(t, err)
	srv1 = server1.service

	server2, err := h.createService(cfg, noopReceiver)

	err = server2.service.ShipTo(server1.directAddress(), sh)
	require.NoError(t, err)

	_, ok := <-ch1

	require.False(t, ok)
}

func TestRejectBody(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-799")

	cfg := uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	}

	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	payloadLen := 1024 * 1024 * 512
	bytes := rndBytes(payloadLen)
	head := TestString{string(bytes[:64])}
	body := TestString{string(bytes)}

	sh := Shipment{
		Head: &head,
		Body: &body,
	}

	var srv1 Service

	ch1 := make(chan string, 2)

	server1, err := h.createService(cfg, func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
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
	})
	require.Error(t, err)
	srv1 = server1.service

	server2, err := h.createService(cfg, noopReceiver)
	require.Error(t, err)

	err = server2.service.ShipTo(server1.directAddress(), sh)
	require.NoError(t, err)

	expHeadPayload := head.S
	actlHeadPayload := <-ch1

	require.Equal(t, expHeadPayload, actlHeadPayload)
}

func TestShipToWithTTL(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-800")
}

func TestShipReturnWithTTL(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-800")
}
