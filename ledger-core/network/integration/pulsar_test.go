// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/pulsenetwork"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type TestPulsar interface {
	Start(ctx context.Context, bootstrapHosts []string) error
	Pause()
	Continue()
	component.Stopper
}

func NewTestPulsar(requestsTimeoutMs, pulseDelta int) (TestPulsar, error) {

	return &testPulsar{
		generator:         &entropygenerator.StandardEntropyGenerator{},
		reqTimeoutMs:      int32(requestsTimeoutMs),
		pulseDelta:        uint16(pulseDelta),
		cancellationToken: make(chan struct{}),
	}, nil
}

type testPulsar struct {
	distributor pulsar.PulseDistributor
	generator   entropygenerator.EntropyGenerator
	cm          *component.Manager

	activityMutex sync.Mutex

	reqTimeoutMs int32
	pulseDelta   uint16

	cancellationToken chan struct{}

	unifiedServer *uniserver.UnifiedServer
	dispatcher    uniserver.Dispatcher
}

func (tp *testPulsar) InitUniserver(ctx context.Context) {
	vf := servicenetwork.TestVerifierFactory{}
	skBytes := [servicenetwork.TestDigestSize]byte{}
	sk := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), servicenetwork.TestSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1

	tp.unifiedServer = uniserver.NewUnifiedServer(&tp.dispatcher, servicenetwork.TestLogAdapter{ctx})
	tp.unifiedServer.SetConfig(uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
		UDPMaxSize:     1400,
		UDPParallelism: 2,
		PeerLimit:      -1,
	})

	tp.unifiedServer.SetPeerFactory(func(peer *uniserver.Peer) (remapTo nwapi.Address, err error) {
		peer.SetSignatureKey(sk)
		// peer.SetNodeID(2) // todo: ??
		// return nwapi.NewHostID(2), nil
		return nwapi.Address{}, nil
	})
	tp.unifiedServer.SetSignatureFactory(vf)

	var desc = uniproto.Descriptor{
		SupportedPackets: uniproto.PacketDescriptors{
			0: {Flags: uniproto.NoSourceID | uniproto.OptionalTarget | uniproto.DatagramAllowed, LengthBits: 16},
		},
	}

	datagramHandler := adapters.NewDatagramHandler()
	marshaller := &adapters.ConsensusProtocolMarshaller{HandlerAdapter: datagramHandler}
	tp.dispatcher.SetMode(uniproto.NewConnectionMode(uniproto.AllowUnknownPeer, 0))
	tp.dispatcher.RegisterProtocol(0, desc, marshaller, marshaller)
}

func (tp *testPulsar) Start(ctx context.Context, bootstrapHosts []string) error {
	var err error

	tp.InitUniserver(ctx)
	// tp.unifiedServer.StartListen()

	distributorCfg := configuration.PulseDistributor{
		BootstrapHosts:      bootstrapHosts,
		PulseRequestTimeout: tp.reqTimeoutMs,
	}

	key, err := platformpolicy.NewKeyProcessor().GeneratePrivateKey()
	if err != nil {
		return err
	}

	tp.distributor, err = pulsenetwork.NewDistributor(distributorCfg, tp.unifiedServer)
	if err != nil {
		return errors.W(err, "Failed to create pulse distributor")
	}

	tp.cm = component.NewManager(nil)
	tp.cm.SetLogger(global.Logger())

	tp.cm.Register(platformpolicy.NewPlatformCryptographyScheme(), keystore.NewInplaceKeyStore(key))

	cfg := configuration.NewHostNetwork()
	cfg.Transport.Protocol = "udp"

	tp.cm.Inject(tp.distributor)

	if err = tp.cm.Init(ctx); err != nil {
		return errors.W(err, "Failed to init test pulsar components")
	}
	if err = tp.cm.Start(ctx); err != nil {
		return errors.W(err, "Failed to start test pulsar components")
	}

	go tp.distribute(ctx)
	return nil
}

func (tp *testPulsar) Pause() {
	tp.activityMutex.Lock()
}

func (tp *testPulsar) Continue() {
	tp.activityMutex.Unlock()
}

func (tp *testPulsar) distribute(ctx context.Context) {
	timeNow := time.Now()
	pulseNumber := pulse.OfTime(timeNow)

	pls := pulsar.PulsePacket{
		PulseNumber:      pulseNumber,
		Entropy:          tp.generator.GenerateEntropy(),
		NextPulseNumber:  pulseNumber.Next(tp.pulseDelta),
		PrevPulseNumber:  pulseNumber - pulse.Number(tp.pulseDelta),
		EpochPulseNumber: pulseNumber.AsEpoch(),
		OriginID:         [16]byte{206, 41, 229, 190, 7, 240, 162, 155, 121, 245, 207, 56, 161, 67, 189, 0},
	}

	var err error
	pls.Signs, err = getPSC(pls)
	if err != nil {
		global.Errorf("[ distribute ]", err)
	}

	for {
		select {
		case <-time.After(time.Duration(tp.pulseDelta) * time.Second):
			go func(pulse pulsar.PulsePacket) {
				tp.activityMutex.Lock()
				defer tp.activityMutex.Unlock()

				pulse.PulseTimestamp = time.Now().UnixNano()

				tp.distributor.Distribute(ctx, pulse)
			}(pls)

			pls = tp.incrementPulse(pls)
		case <-tp.cancellationToken:
			return
		}
	}
}

func (tp *testPulsar) incrementPulse(pulse pulsar.PulsePacket) pulsar.PulsePacket {
	newPulseNumber := pulse.PulseNumber.Next(tp.pulseDelta)
	newPulse := pulsar.PulsePacket{
		PulseNumber:      newPulseNumber,
		Entropy:          tp.generator.GenerateEntropy(),
		NextPulseNumber:  newPulseNumber.Next(tp.pulseDelta),
		PrevPulseNumber:  pulse.PulseNumber,
		EpochPulseNumber: pulse.EpochPulseNumber,
		OriginID:         pulse.OriginID,
		PulseTimestamp:   time.Now().UnixNano(),
		Signs:            pulse.Signs,
	}
	var err error
	newPulse.Signs, err = getPSC(newPulse)
	if err != nil {
		global.Errorf("[ incrementPulse ]", err)
	}
	return newPulse
}

func getPSC(pulse pulsar.PulsePacket) (map[string]pulsar.SenderConfirmation, error) {
	proc := platformpolicy.NewKeyProcessor()
	key, err := proc.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	pem, err := proc.ExportPublicKeyPEM(proc.ExtractPublicKey(key))
	if err != nil {
		return nil, err
	}
	result := make(map[string]pulsar.SenderConfirmation)
	psc := pulsar.SenderConfirmation{
		PulseNumber:     pulse.PulseNumber,
		ChosenPublicKey: string(pem),
		Entropy:         pulse.Entropy,
	}

	payload := pulsar.PulseSenderConfirmationPayload{SenderConfirmation: psc}
	hasher := platformpolicy.NewPlatformCryptographyScheme().IntegrityHasher()
	hash, err := payload.Hash(hasher)
	if err != nil {
		return nil, err
	}
	service := platformpolicy.NewKeyBoundCryptographyService(key)
	sign, err := service.Sign(hash)
	if err != nil {
		return nil, err
	}

	psc.Signature = sign.Bytes()
	result[string(pem)] = psc

	return result, nil
}

func (tp *testPulsar) Stop(ctx context.Context) error {
	if err := tp.cm.Stop(ctx); err != nil {
		return errors.W(err, "Failed to stop test pulsar components")
	}
	close(tp.cancellationToken)
	tp.unifiedServer.Stop()
	return nil
}
