// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsenetwork

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

const (
	PULSENUMBER = pulse.MinTimePulse + 155
)

type pProcessor struct{}

func (pp *pProcessor) ProcessPacket(ctx context.Context, payload transport.PacketParser, from endpoints.Inbound) error {
	fmt.Printf("ProcessPacket from %s : %v", from.GetNameAddress(), payload)
	return nil
}

func createUniserver(id nwapi.ShortNodeID, address string) *uniserver.UnifiedServer {
	var unifiedServer *uniserver.UnifiedServer
	var dispatcher uniserver.Dispatcher

	vf := servicenetwork.TestVerifierFactory{}
	skBytes := [servicenetwork.TestDigestSize]byte{}
	sk := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), servicenetwork.TestSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1

	unifiedServer = uniserver.NewUnifiedServer(&dispatcher, servicenetwork.TestLogAdapter{context.Background()})
	unifiedServer.SetConfig(uniserver.ServerConfig{
		BindingAddress: address,
		UDPMaxSize:     1400,
		UDPParallelism: 1,
		PeerLimit:      -1,
	})

	unifiedServer.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk)
		peer.SetNodeID(id) // todo: ??
		return nwapi.NewHostID(nwapi.HostID(id)), nil
		// return nwapi.Address{}, nil
	})
	unifiedServer.SetSignatureFactory(vf)

	var desc = uniproto.Descriptor{
		SupportedPackets: uniproto.PacketDescriptors{
			0: {Flags: uniproto.NoSourceID | uniproto.OptionalTarget | uniproto.DatagramAllowed | uniproto.DatagramOnly, LengthBits: 16},
		},
	}

	datagramHandler := adapters.NewDatagramHandler()
	datagramHandler.SetPacketProcessor(&pProcessor{})

	marshaller := &adapters.ConsensusProtocolMarshaller{HandlerAdapter: datagramHandler}
	dispatcher.SetMode(uniproto.NewConnectionMode(uniproto.AllowUnknownPeer, 0))
	dispatcher.RegisterProtocol(0, desc, marshaller, marshaller)

	return unifiedServer
}

func TestDistributor_Distribute(t *testing.T) {
	instestlogger.SetTestOutput(t)

	nodeServ := createUniserver(1, "127.0.0.1:0")
	pulsarServ := createUniserver(2, "127.0.0.1:0")
	nodeServ.StartListen()
	pulsarServ.StartListen()

	ctx := context.Background()

	// handler := func(ctx context.Context, r network.ReceivedPacket) (network.Packet, error) {
	// 	global.Info("handle Pulse")
	// 	pulse := r.GetRequest().GetPulse()
	// 	assert.EqualValues(t, PULSENUMBER, pulse.Pulse.PulseNumber)
	// 	return nil, nil
	// }
	// n1.RegisterRequestHandler(types.Pulse, handler)

	// err = n1.Start(ctx)
	// require.NoError(t, err)
	// defer func() {
	// 	err = n1.Stop(ctx)
	// 	require.NoError(t, err)
	// }()

	address := nodeServ.PeerManager().Local().GetPrimary().String()

	pulsarCfg := configuration.NewPulsar()
	pulsarCfg.PulseDistributor.BootstrapHosts = []string{address}

	d, err := NewDistributor(pulsarCfg.PulseDistributor, pulsarServ)
	require.NoError(t, err)
	assert.NotNil(t, d)

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	key, err := platformpolicy.NewKeyProcessor().GeneratePrivateKey()
	require.NoError(t, err)

	cm.Inject(d, platformpolicy.NewPlatformCryptographyScheme(), keystore.NewInplaceKeyStore(key))
	err = cm.Init(ctx)
	require.NoError(t, err)
	err = cm.Start(ctx)
	require.NoError(t, err)
	defer func() {
		err = cm.Stop(ctx)
		require.NoError(t, err)
	}()

	d.Distribute(ctx, pulsar.PulsePacket{PulseNumber: PULSENUMBER})
	d.Distribute(ctx, pulsar.PulsePacket{PulseNumber: PULSENUMBER})
	d.Distribute(ctx, pulsar.PulsePacket{PulseNumber: PULSENUMBER})
}
