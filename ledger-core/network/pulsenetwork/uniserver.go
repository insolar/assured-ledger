// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsenetwork

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func NewPulsarUniserver() *uniserver.UnifiedServer {
	var dispatcher uniserver.Dispatcher

	vf := servicenetwork.TestVerifierFactory{}
	skBytes := [servicenetwork.TestDigestSize]byte{}
	sk := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), servicenetwork.TestSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1

	unifiedServer := uniserver.NewUnifiedServer(&dispatcher, servicenetwork.TestLogAdapter{context.Background()})
	unifiedServer.SetConfig(uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	})

	unifiedServer.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk)
		return nwapi.Address{}, nil
	})
	unifiedServer.SetSignatureFactory(vf)

	var desc = uniproto.Descriptor{
		SupportedPackets: uniproto.PacketDescriptors{
			uniproto.ProtocolTypePulsar: {Flags: uniproto.NoSourceID | uniproto.OptionalTarget | uniproto.DatagramAllowed | uniproto.DatagramOnly, LengthBits: 16},
		},
	}

	receiver := &uniproto.NoopReceiver{}
	dispatcher.SetMode(uniproto.NewConnectionMode(uniproto.AllowUnknownPeer, uniproto.ProtocolTypePulsar))
	dispatcher.RegisterProtocol(uniproto.ProtocolTypePulsar, desc, receiver, receiver)

	return unifiedServer
}
