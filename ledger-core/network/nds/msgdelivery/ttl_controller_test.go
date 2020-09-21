// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestShipToWithTTL(t *testing.T) {

	head := TestString{string(rndBytes(64))}
	sleepChan := make(chan string, 1)
	const Server1 = "127.0.0.1:0"
	const Server2 = "127.0.0.1:0"
	pr := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{}))

	sh := Shipment{
		Head: &head,
		TTL:  1,
		PN:   pr.LeftBoundNumber(),
	}

	vf := TestVerifierFactory{}
	skBytes := [testDigestSize]byte{}
	sk1 := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1
	sk2 := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)

	ctrl1 := NewController(
		Protocol,
		TestDeserializationFactory{},
		func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
			require.FailNow(t, "it is not recend")
			return nil
		},
		nil,
		TestLogAdapter{t},
	)

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

	provider := uniserver.MapTransportProvider(&uniserver.DefaultTransportProvider{},
		func(lessProvider l1.SessionlessTransportProvider) l1.SessionlessTransportProvider {
			return l1.MapSessionlessProvider(lessProvider, func(factory l1.OutTransportFactory) l1.OutTransportFactory {
				return l1.MapOutputFactory(factory, func(transport l1.OneWayTransport) l1.OneWayTransport {
					return &TestOneWayTransport{transport, sleepChan}
				})
			})
		},
		func(fullProvider l1.SessionfulTransportProvider) l1.SessionfulTransportProvider {
			return l1.MapSessionFullProvider(fullProvider, func(factory l1.OutTransportFactory) l1.OutTransportFactory {
				return l1.MapOutputFactory(factory, func(transport l1.OneWayTransport) l1.OneWayTransport {
					return &TestOneWayTransport{transport, sleepChan}
				})
			})
		})

	ups1.SetTransportProvider(provider)
	ups1.StartListen()
	dispatcher1.SetMode(uniproto.AllowAll)

	pm1 := ups1.PeerManager()
	_, err := pm1.AddHostID(pm1.Local().GetPrimary(), 1)
	require.NoError(t, err)

	dispatcher1.NextPulse(pr)

	/********************************/

	ctrl2 := NewController(
		Protocol,
		TestDeserializationFactory{},
		noopReceiver,
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
	require.NoError(t, conn21.Transport().EnsureConnect()) // -end of create servers

	err = srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	pn := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(10, longbits.Bits256{}))
	dispatcher1.NextPulse(pn)
	// dispatcher1.NextPulse(pn)
	// dispatcher2.NextPulse(pn)
	dispatcher2.NextPulse(pn)

	sleepChan <- ""

	dispatcher1.Stop()
	dispatcher2.Stop()
}

type TestOneWayTransport struct {
	l1.OneWayTransport
	ch chan string
}

func (t TestOneWayTransport) Send(payload io.WriterTo) error {
	<-t.ch
	return t.OneWayTransport.Send(payload)
}
