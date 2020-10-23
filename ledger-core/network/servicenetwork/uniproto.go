// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type TestLogAdapter struct {
	Ctx context.Context
}

func (t TestLogAdapter) LogError(err error) {
	if network.IsConnectionClosed(err) {
		return
	}
	inslogger.FromContext(t.Ctx).Error(err)
}

func (t TestLogAdapter) LogTrace(m interface{}) {
	inslogger.FromContext(t.Ctx).Error(m)
}

func (n *ServiceNetwork) initUniproto(ctx context.Context) msgdelivery.Service {
	vf := TestVerifierFactory{}
	skBytes := [TestDigestSize]byte{}
	sk := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), TestSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1

	recv1 := func(a msgdelivery.ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		_, err := n.processIncoming(ctx, v.([]byte))
		if err != nil {
			inslogger.FromContext(ctx).Error(err)
			return err
		}
		return nil
	}

	ctrl := msgdelivery.NewController(
		msgdelivery.Protocol,
		TestDeserializationFactory{},
		recv1,
		nil,
		TestLogAdapter{ctx},
	)

	n.msgdeliveryService = ctrl.NewFacade()

	ctrl.RegisterWith(n.dispatcher.RegisterProtocol)

	n.unifiedServer = uniserver.NewUnifiedServer(&n.dispatcher, TestLogAdapter{ctx})
	n.unifiedServer.SetConfig(uniserver.ServerConfig{
		BindingAddress: n.cfg.Host.Transport.Address,
		// BindingAddress: n.cfg.Host.Transport.Address,
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	})

	n.unifiedServer.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk)
		// peer.SetNodeID(2) // todo: ??
		// return nwapi.NewHostID(2), nil
		return nwapi.Address{}, nil
	})
	n.unifiedServer.SetSignatureFactory(vf)

	// pr := pulse.NewOnePulseRange(pulse.NewFirstPulsarData(5, longbits.Bits256{}))
	// dispatcher.NextPulse(pr)

	return n.msgdeliveryService
}
