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
				return l1.MapOutputFactory(factory, func(transport l1.OneWayTransport) l1.OneWayTransport {
					fmt.Printf("create sesstion full connnection\n")
					return &testPausedSendOutputTransport{transport, pause}
				})
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
	srv2, err := h.createServiceWithProfile(&prf2, receiver2)
	require.NoError(t, err)
	srv3, err := h.createServiceWithProfile(&prf3, receiver3)
	require.NoError(t, err)

	fmt.Print("Start sending\n")

	for i := 0; i < numberMessage; i++ {
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

	receiver2 := func(a ReturnAddress, done nwapi.PayloadCompleteness, val interface{}) error {
		require.True(t, bool(done))
		s := val.(fmt.Stringer).String()

		fmt.Printf("%s\n", s)
		switch {
		case strings.HasPrefix(s, "1"):
			pause <- struct{}{}
			<-pause
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

	for i := 0; i < numberMessage; i++ {
		res := <-receivedFrom3
		require.Equal(t, fmt.Sprintf("3%d", i), res)
	}

	pause <- struct{}{}

	for i := 0; i < numberMessage; i++ {
		res := <-receivedFrom1
		require.Equal(t, fmt.Sprintf("1%d", i), res)
	}
}

type testPausedSendOutputTransport struct {
	l1.OneWayTransport
	pause chan struct{}
}

func (t testPausedSendOutputTransport) Send(payload io.WriterTo) error {
	t.pause <- struct{}{}
	<-t.pause
	return t.OneWayTransport.Send(payload)
}

func (t testPausedSendOutputTransport) SendBytes(b []byte) error {
	t.pause <- struct{}{}
	<-t.pause
	return t.OneWayTransport.SendBytes(b)
}
