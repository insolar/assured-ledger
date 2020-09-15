// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func createService(
	t testing.TB,
	index nwapi.HostID,
	receiverFn func(a ReturnAddress, _ nwapi.PayloadCompleteness, v interface{}) error,
	config uniserver.ServerConfig,
	idWithPortFn func(nwapi.Address) bool) ServiceTest {

	controller := NewController(Protocol, TestDeserializationByteFactory{}, receiverFn, nil, TestLogAdapter{t})

	var dispatcher uniserver.Dispatcher
	controller.RegisterWith(dispatcher.RegisterProtocol)

	vf := TestVerifierFactory{}

	srv := uniserver.NewUnifiedServer(&dispatcher, TestLogAdapter{t})
	srv.SetConfig(config)
	srv.SetIdentityClassifier(idWithPortFn)

	peerFn := func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		idx := connectionCount
		connectionCount++
		if connectionCount > len(StartedServers) {
			panic("")
		}

		peer.SetSignatureKey(StartedServers[idx].key)
		peer.SetNodeID(nwapi.ShortNodeID(idx))

		return nwapi.NewHostID(nwapi.HostID(idx)), nil
	}

	srv.SetPeerFactory(peerFn)
	srv.SetSignatureFactory(vf)

	srv.StartListen()
	dispatcher.SetMode(uniproto.AllowAll)

	manager := srv.PeerManager()
	_, err := manager.AddHostID(manager.Local().GetPrimary(), index)
	require.NoError(t, err)

	pr := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{}))
	dispatcher.NextPulse(pr)

	skBytes := [testDigestSize]byte{}
	skBytes[0] = 1
	sk := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)

	serviceInfo := ServiceTest{
		service: controller.NewFacade(),
		key:     sk,
		disp:    dispatcher,
		mng:     manager,
	}
	StartedServers[index] = serviceInfo
	return serviceInfo
}

type ServiceTest struct {
	service Service
	key     cryptkit.SigningKey
	disp    uniserver.Dispatcher
	mng     *uniserver.PeerManager
}
