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

var (
	servers = make([]*UniprotoServer, 0)
)

func getServerByIndex(idx int) *UniprotoServer {
	if idx < 1 || idx > len(servers) {
		panic("")
	}

	return servers[idx-1]
}

type UniprotoServer struct {
	service Service
	key     cryptkit.SigningKey
	disp    *uniserver.Dispatcher
	mng     *uniserver.PeerManager
}

func createService(
	t testing.TB,
	receiverFn ReceiverFunc,
	config uniserver.ServerConfig,
	idWithPortFn func(nwapi.Address) bool,
) *UniprotoServer {
	println("idWithPortFn", idWithPortFn)
	controller := NewController(Protocol, TestDeserializationByteFactory{}, receiverFn, nil, TestLogAdapter{t})

	var dispatcher uniserver.Dispatcher
	controller.RegisterWith(dispatcher.RegisterProtocol)

	vf := TestVerifierFactory{}

	srv := uniserver.NewUnifiedServer(&dispatcher, TestLogAdapter{t})
	srv.SetConfig(config)
	srv.SetIdentityClassifier(idWithPortFn)

	// This is min value for NodeID, 0 is not allowed
	// con := 0

	peerFn := func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		idx := len(servers)
		// if con != len(servers)-1 {
		// 	panic("")
		// }

		peer.SetSignatureKey(servers[idx-1].key)
		peer.SetNodeID(nwapi.ShortNodeID(idx))

		println("setted idx %d, con %d", idx, len(servers))
		// println("idx %d, con %d", idx, con)
		// con++
		return nwapi.NewHostID(nwapi.HostID(idx)), nil
	}

	srv.SetPeerFactory(peerFn)
	srv.SetSignatureFactory(vf)

	srv.StartListen()
	dispatcher.SetMode(uniproto.AllowAll)

	manager := srv.PeerManager()
	_, err := manager.AddHostID(manager.Local().GetPrimary(), nwapi.HostID(len(servers)+1))
	require.NoError(t, err)

	pr := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{}))
	dispatcher.NextPulse(pr)

	skBytes := [testDigestSize]byte{}
	skBytes[0] = 1
	sk := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), testSigningMethod, cryptkit.PublicAsymmetricKey)

	for _, s := range servers {
		// println("len servers %d", len(servers))
		// println("con %d",con)
		// it is not worked, server.go:313: error {127.0.0.1:60936 127.0.0.1:60937};	remap to loopback
		con, err := manager.Manager().ConnectPeer(s.mng.Local().GetPrimary())

		require.NoError(t, err)
		require.NoError(t, con.Transport().EnsureConnect())
	}

	info := &UniprotoServer{
		service: controller.NewFacade(),
		key:     sk,
		disp:    &dispatcher,
		mng:     manager,
	}
	servers = append(servers, info)
	return info
}
