// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

func TestController(t *testing.T) {
	const Server1 = "127.0.0.1:10001"
	const Server2 = "127.0.0.1:10002"

	addr1 := nwapi.NewHostPort(Server1)
	addr2 := nwapi.NewHostPort(Server2)

	vf := TestVerifierFactory{}
	sk := cryptkit.NewSignatureKey(longbits.Zero(testDigestSize), testSignatureMethod, cryptkit.PublicAsymmetricKey)

	controller1 := NewController(Protocol, TestDeserializationFactory{},
		func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
			fmt.Println(a.String(), v)
			return nil
		}, nil)

	var dispatcher1 uniserver.Dispatcher
	controller1.RegisterWith(dispatcher1.RegisterProtocol)

	ups1 := uniserver.NewUnifiedServer(&dispatcher1, 2)
	ups1.SetConfig(uniserver.ServerConfig{
		BindingAddress: Server1,
		UdpMaxSize:     1400,
		PeerLimit:      -1,
	})
	ups1.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk)
		peer.SetNodeID(2)
		return addr2, nil
	})
	ups1.SetSignatureFactory(vf)
	//	ups1.Set

	ups1.StartListen()
	dispatcher1.SetMode(uniproto.AllowAll)

	pm1 := ups1.PeerManager()
	_, err := pm1.AddHostID(pm1.Local().GetPrimary(), 1)
	require.NoError(t, err)

	pr := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{}))
	dispatcher1.NextPulse(pr)

	//********************************

	controller2 := NewController(Protocol, TestDeserializationFactory{},
		func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error {
			fmt.Println(a.String(), v)
			return nil
		}, nil)

	var dispatcher2 uniserver.Dispatcher
	dispatcher2.SetMode(uniproto.NewConnectionMode(0, Protocol))
	controller2.RegisterWith(dispatcher2.RegisterProtocol)
	dispatcher2.Seal()

	ups2 := uniserver.NewUnifiedServer(&dispatcher2, 2)
	ups2.SetConfig(uniserver.ServerConfig{
		BindingAddress: Server2,
		UdpMaxSize:     1400,
		PeerLimit:      -1,
	})
	ups2.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk)
		peer.SetNodeID(1)
		return addr1, nil
	})
	ups2.SetSignatureFactory(vf)

	ups2.StartNoListen()
	dispatcher2.NextPulse(pr)

	pm2 := ups2.PeerManager()
	_, err = pm2.AddHostID(pm2.Local().GetPrimary(), 2)
	require.NoError(t, err)

	conn21, err := pm2.Manager().ConnectPeer(addr1)
	require.NoError(t, err)
	require.NotNil(t, conn21)
	require.NoError(t, conn21.Transport().EnsureConnect())

	ctl2 := controller2.NewFacade()

	// loopback
	err = ctl2.ShipTo(NewDirectAddress(2), Shipment{Head: &TestString{"abc1"}})
	require.NoError(t, err)

	err = ctl2.ShipTo(NewDirectAddress(1), Shipment{Head: &TestString{"abc2"}})
	require.NoError(t, err)

	time.Sleep(time.Hour)
}
