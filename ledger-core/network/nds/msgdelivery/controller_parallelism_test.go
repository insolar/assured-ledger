// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"fmt"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/stretchr/testify/require"
	"io"
	"strconv"
	"strings"
	"testing"
)

func TestParallelismSend(t *testing.T) {
	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	prf := UnitProtoServerProfile{
		config: &uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		desFactory: &TestDeserializationStringFactory{},
	}

	prf1 := prf
	prf2 := prf
	prf3 := prf

	pause := make(chan struct{})

	var (
		srv1 *UnitProtoServer
		srv2 *UnitProtoServer
		srv3 *UnitProtoServer
	)

	prf1.provider = uniserver.MapTransportProvider(&uniserver.DefaultTransportProvider{},
		func(provider l1.SessionlessTransportProvider) l1.SessionlessTransportProvider {
			return l1.MapSessionlessProvider(provider, func(factory l1.OutTransportFactory) l1.OutTransportFactory {
				return l1.MapOutputFactory(factory, func(transport l1.OneWayTransport) l1.OneWayTransport {
					fmt.Printf("create sesstion less connnection\n")
					return &testPausedSendOutputTransport{transport, pause}
				})
			})
		},
		func(provider l1.SessionfulTransportProvider) l1.SessionfulTransportProvider {
			return l1.MapSessionFullProvider(provider, func(factory l1.OutTransportFactory) l1.OutTransportFactory {
				return &testPauseSendOutputFactory{
					pure: factory,
					paused: l1.MapOutputFactory(factory, func(transport l1.OneWayTransport) l1.OneWayTransport {
						fmt.Printf("create sesstion full connnection\n")
						return &testPausedSendOutputTransport{transport, pause}
					}),
					needPaused: func(addr nwapi.Address) bool {
						switch {
						case srv2.ingoing == addr:
							return true
						default:
							return false
						}
					},
				}

			})
		})

	numberMessage := 1000

	resultsOnServ2 := make(chan string, numberMessage)
	resultsOnServ3 := make(chan string, numberMessage)

	receiver2 := func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		require.True(t, bool(done))
		s := val.(fmt.Stringer).String()

		resultsOnServ2 <- s
		return nil
	}

	receiver3 := func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		require.True(t, bool(done))
		s := val.(fmt.Stringer).String()

		resultsOnServ3 <- s
		return nil
	}

	srv1, err := h.createServiceWithProfile(&prf1, noopReceiver)
	require.NoError(t, err)
	srv2, err = h.createServiceWithProfile(&prf2, receiver2)
	require.NoError(t, err)
	srv3, err = h.createServiceWithProfile(&prf3, receiver3)
	require.NoError(t, err)

	fmt.Print("Start sending\n")

	for i := 0; i < 1; i++ {
		err := srv1.service.ShipTo(srv2.directAddress(), Shipment{Head: &TestString{fmt.Sprintf("%d", i)}})
		require.NoError(t, err)
	}

	<-pause

	for i := 0; i < numberMessage; i++ {
		err := srv1.service.ShipTo(srv3.directAddress(), Shipment{Head: &TestString{fmt.Sprintf("%d", i)}})
		require.NoError(t, err)
	}

	select {
	case res, ok := <-resultsOnServ2:
		require.True(t, ok)
		t.Fatalf("unexpected string received:%s", res)
	default:
		// channel is empty, this okay
	}

	for i := 0; i < numberMessage; i++ {
		res := <-resultsOnServ3
		require.Equal(t, fmt.Sprintf("%d", i), res)
	}

	pause <- struct{}{}

	for i := 0; i < numberMessage; i++ {
		res := <-resultsOnServ2
		require.Equal(t, fmt.Sprintf("%d", i), res)
	}
}

func TestParallelismReceive(t *testing.T) {
	h := NewUnitProtoServersHolder(TestLogAdapter{t: t})
	defer h.stop()

	prf := UnitProtoServerProfile{
		config: &uniserver.ServerConfig{
			BindingAddress: "127.0.0.1:0",
			UDPMaxSize:     0,
			UDPParallelism: 4,
			PeerLimit:      -1,
		},
		desFactory: &TestDeserializationStringFactory{},
	}

	prf1 := prf
	prf2 := prf
	prf3 := prf

	pause := make(chan struct{})

	numberMessage := 1000

	receivedFrom1 := make(chan string, numberMessage)
	receivedFrom3 := make(chan string, numberMessage)

	first := true

	receiver2 := func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		require.True(t, bool(done))
		s := val.(fmt.Stringer).String()

		switch {
		case strings.HasPrefix(s, "1"):
			if first {
				pause <- struct{}{}
				<-pause
				first = false
			}

			receivedFrom1 <- s
		case strings.HasPrefix(s, "3"):
			receivedFrom3 <- s
		}

		return nil
	}

	srv1, err := h.createServiceWithProfile(&prf1, noopReceiver)
	require.NoError(t, err)
	srv2, err := h.createServiceWithProfile(&prf2, receiver2)
	require.NoError(t, err)
	srv3, err := h.createServiceWithProfile(&prf3, noopReceiver)
	require.NoError(t, err)

	for i := 0; i < numberMessage; i++ {
		err := srv1.service.ShipTo(srv2.directAddress(), Shipment{Head: &TestString{fmt.Sprintf("1%d", i)}})
		require.NoError(t, err)
	}

	<-pause

	for i := 0; i < numberMessage; i++ {
		err := srv3.service.ShipTo(srv2.directAddress(), Shipment{Head: &TestString{fmt.Sprintf("3%d", i)}})
		require.NoError(t, err)
	}

	select {
	case res, ok := <-receivedFrom1:
		require.True(t, ok)
		t.Fatalf("unexpected string received:%s", res)
	default:
		// channel is empty, this okay
	}

	checker := func(from string, ch chan string) {
		res := make(map[int]string)

		for i := 0; i < numberMessage; i++ {
			val := <-ch

			val0 := strings.TrimPrefix(val, from)

			parseInt, e := strconv.ParseInt(val0, 10, 32)
			require.NoError(t, e)
			res[int(parseInt)] = val
		}

		for i := 0; i < numberMessage; i++ {
			val := res[i]
			require.Equal(t, fmt.Sprintf("%s%d", from, i), val)
		}
	}

	checker("3", receivedFrom3)

	pause <- struct{}{}

	checker("1", receivedFrom1)
}

type testPausedSendOutputTransport struct {
	l1.OneWayTransport
	pause chan struct{}
}

func (t testPausedSendOutputTransport) Send(payload io.WriterTo) error {
	//t.pause <- struct{}{}
	fmt.Printf("paused Send\n")
	<-t.pause
	return t.OneWayTransport.Send(payload)
}

func (t testPausedSendOutputTransport) SendBytes(b []byte) error {
	t.pause <- struct{}{}
	fmt.Printf("paused SendBytes\n")
	<-t.pause
	return t.OneWayTransport.SendBytes(b)
}

type testPauseSendOutputFactory struct {
	pure       l1.OutTransportFactory
	paused     l1.OutTransportFactory
	needPaused func(addr nwapi.Address) bool
}

func (t testPauseSendOutputFactory) Close() error {
	return t.pure.Close()
}

func (t testPauseSendOutputFactory) ConnectTo(address nwapi.Address) (l1.OneWayTransport, error) {
	if t.needPaused(address) {
		return t.paused.ConnectTo(address)
	}

	return t.pure.ConnectTo(address)
}
