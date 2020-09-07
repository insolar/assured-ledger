// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestController(t *testing.T) {
	const Server1 = "127.0.0.1:0"
	const Server2 = "127.0.0.1:0"

	vf := TestVerifierFactory{}
	skBytes := [testDigestSize]byte{}
	sk1 := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1
	sk2 := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)

	var receive1 atomickit.Uint32
	var receive2 atomickit.Uint32

	results := make(chan string, 2000)

	// var ctl1 Service
	controller1 := NewController(Protocol, TestDeserializationFactory{},
		func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
			receive1.Add(1)
			// t.Log(a.String(), "Ctl1:", v)
			// fmt.Println(a.String(), "Ctl1:", v)
			s := v.(fmt.Stringer).String()
			results <- s
			// s := v.(fmt.Stringer).String() + "-return"
			// return ctl1.ShipReturn(a, Shipment{Head: &TestString{s}})
			return nil
		}, nil, TestLogAdapter{t})

	// ctl1 = controller1.NewFacade()

	var dispatcher1 uniserver.Dispatcher
	controller1.RegisterWith(dispatcher1.RegisterProtocol)

	ups1 := uniserver.NewUnifiedServer(&dispatcher1, TestLogAdapter{t})
	ups1.SetConfig(uniserver.ServerConfig{
		BindingAddress: Server1,
		UDPMaxSize:     0,
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
	defer dispatcher1.Stop()

	pm1 := ups1.PeerManager()
	_, err := pm1.AddHostID(pm1.Local().GetPrimary(), 1)
	require.NoError(t, err)

	pr := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{}))
	dispatcher1.NextPulse(pr)

	/********************************/

	controller2 := NewController(Protocol, TestDeserializationFactory{},
		func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
			receive2.Add(1)
			s := v.(fmt.Stringer).String()
			// fmt.Println(a.String(), "Ctl2:", s)
			// t.Log(a.String(), "Ctl2:", s)
			results <- s
			return nil
		}, nil, TestLogAdapter{t})

	var dispatcher2 uniserver.Dispatcher
	dispatcher2.SetMode(uniproto.NewConnectionMode(0, Protocol))
	controller2.RegisterWith(dispatcher2.RegisterProtocol)
	dispatcher2.Seal()

	ups2 := uniserver.NewUnifiedServer(&dispatcher2, TestLogAdapter{t})
	ups2.SetConfig(uniserver.ServerConfig{
		BindingAddress: Server2,
		UDPMaxSize:     0,
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
	defer dispatcher2.Stop()

	pm2 := ups2.PeerManager()
	_, err = pm2.AddHostID(pm2.Local().GetPrimary(), 2)
	require.NoError(t, err)

	conn21, err := pm2.Manager().ConnectPeer(pm1.Local().GetPrimary())
	require.NoError(t, err)
	require.NotNil(t, conn21)
	require.NoError(t, conn21.Transport().EnsureConnect())

	ctl2 := controller2.NewFacade()
	// loopback
	err = ctl2.ShipTo(NewDirectAddress(2), Shipment{Head: &TestString{"loc0"}})
	require.NoError(t, err)

	err = ctl2.ShipTo(NewDirectAddress(1), Shipment{Head: &TestString{"rem0"}})
	require.NoError(t, err)

	require.Equal(t, "loc0", <-results)
	require.Equal(t, "rem0", <-results)
	// require.Equal(t, "rem0-return", <-results)

	sentMap := sync.Map{}
	totalCount := atomickit.Uint32{}

	for i := 1; i <= 1000; i++ {
		// loopback
		// err = ctl2.ShipTo(NewDirectAddress(2), Shipment{Head: &TestString{"loc" + strconv.Itoa(i)}})
		// require.NoError(t, err)

		key := "rem" + strconv.Itoa(i)
		sentMap.Store(key, nil)
		totalCount.Add(1)
		err = ctl2.ShipTo(NewDirectAddress(1), Shipment{Head: &TestString{key}})
		require.NoError(t, err)
	}

	go func() {
		for totalCount.Load() > 0 {
			key := <-results
			sentMap.Delete(key)
			totalCount.Sub(1)
		}
	}()

	time.Sleep(time.Second*10)

	if totalCount.Load() > 0 {
		sentMap.Range(func(key, value interface{}) bool {
			fmt.Println(key)
			return true
		})

		t.Fail()
	}
}
