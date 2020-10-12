// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/insolar/component-manager"
	"github.com/insolar/insconfig"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/network/pulsenetwork"
	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

const EnvPrefix = "pulsard"

// Need to fix problem with start pulsar
func main() {
	jww.SetStdoutThreshold(jww.LevelDebug)
	var err error

	pCfg := configuration.NewPulsarConfiguration()
	paramsCfg := insconfig.Params{
		EnvPrefix:        EnvPrefix,
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(paramsCfg)
	err = insConfigurator.Load(&pCfg)
	if err != nil {
		global.Warn("failed to load configuration from file: ", err.Error())
	}

	ctx := context.Background()
	ctx, inslog := inslogger.InitGlobalNodeLogger(ctx, pCfg.Log, "", "pulsar")

	jaegerflush := func() {}
	if pCfg.Tracer.Jaeger.AgentEndpoint != "" {
		jconf := pCfg.Tracer.Jaeger
		global.Infof("Tracing enabled. Agent endpoint: '%s', collector endpoint: '%s'", jconf.AgentEndpoint, jconf.CollectorEndpoint)
		jaegerflush = instracer.ShouldRegisterJaeger(
			ctx,
			"pulsar",
			"pulsar",
			jconf.AgentEndpoint,
			jconf.CollectorEndpoint,
			jconf.ProbabilityRate)
	}
	defer jaegerflush()

	m := metrics.NewMetrics(pCfg.Metrics, metrics.GetInsolarRegistry("pulsar"), "pulsar")
	err = m.Init(ctx)
	if err != nil {
		global.Fatal("Couldn't init metrics:", err)
		os.Exit(1)
	}
	err = m.Start(ctx)
	if err != nil {
		global.Fatal("Couldn't start metrics:", err)
		os.Exit(1)
	}

	cm, server := initPulsar(ctx, pCfg)

	if pCfg.OneShot {
		nextPulseNumber := pulse.OfNow()
		err := server.Send(ctx, nextPulseNumber)
		if err != nil {
			panic(err)
		}
		// it's required since pulse is sent in goroutine
		time.Sleep(time.Second * 10)
		err = cm.Stop(ctx)
		if err != nil {
			inslog.Error(err)
		}
		return
	}

	pulseTicker := runPulsar(ctx, server, pCfg.Pulsar)

	defer func() {
		pulseTicker.Stop()
		err = cm.Stop(ctx)
		if err != nil {
			inslog.Error(err)
		}
	}()

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	<-gracefulStop
}

func initPulsar(ctx context.Context, cfg configuration.PulsarConfiguration) (*component.Manager, *pulsar.Pulsar) {
	fmt.Println("Version: ", version.GetFullVersion())
	fmt.Println("Starts with configuration:\n", configuration.ToString(cfg))

	keyStore, err := keystore.NewKeyStore(cfg.KeysPath)
	if err != nil {
		panic(err)
	}
	cryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
	cryptographyService := platformpolicy.NewCryptographyService()
	keyProcessor := platformpolicy.NewKeyProcessor()

	pulseDistributor, err := pulsenetwork.NewDistributor(cfg.Pulsar.PulseDistributor, createUniserver(100))
	if err != nil {
		panic(err)
	}

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	cm.Register(cryptographyScheme, keyStore, keyProcessor)
	cm.Inject(cryptographyService, pulseDistributor)

	if err = cm.Init(ctx); err != nil {
		panic(err)
	}

	if err = cm.Start(ctx); err != nil {
		panic(err)
	}

	server := pulsar.NewPulsar(
		cfg.Pulsar,
		cryptographyService,
		cryptographyScheme,
		keyProcessor,
		pulseDistributor,
		&entropygenerator.StandardEntropyGenerator{},
	)

	return cm, server
}

func runPulsar(ctx context.Context, server *pulsar.Pulsar, cfg configuration.Pulsar) *time.Ticker {
	nextPulseNumber := pulse.OfNow()
	err := server.Send(ctx, nextPulseNumber)
	if err != nil {
		panic(err)
	}

	pulseTicker := time.NewTicker(time.Duration(cfg.PulseTime) * time.Millisecond)
	go func() {
		for range pulseTicker.C {
			err := server.Send(ctx, server.LastPN()+pulse.Number(cfg.NumberDelta))
			if err != nil {
				panic(err)
			}
		}
	}()

	return pulseTicker
}

func createUniserver(id nwapi.ShortNodeID) *uniserver.UnifiedServer {
	var unifiedServer *uniserver.UnifiedServer
	var dispatcher uniserver.Dispatcher

	vf := servicenetwork.TestVerifierFactory{}
	skBytes := [servicenetwork.TestDigestSize]byte{}
	sk := cryptkit.NewSigningKey(longbits.CopyBytes(skBytes[:]), servicenetwork.TestSigningMethod, cryptkit.PublicAsymmetricKey)
	skBytes[0] = 1

	unifiedServer = uniserver.NewUnifiedServer(&dispatcher, servicenetwork.TestLogAdapter{context.Background()})
	unifiedServer.SetConfig(uniserver.ServerConfig{
		BindingAddress: "127.0.0.1:0",
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
	// datagramHandler.SetPacketProcessor(&pProcessor{})

	marshaller := &adapters.ConsensusProtocolMarshaller{HandlerAdapter: datagramHandler}
	dispatcher.SetMode(uniproto.NewConnectionMode(uniproto.AllowUnknownPeer, 0))
	dispatcher.RegisterProtocol(0, desc, marshaller, marshaller)

	return unifiedServer
}
