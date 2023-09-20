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

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network/pulsenetwork"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

const EnvPrefix = "pulsard"

// Need to fix problem with start pulsar
func main() {
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
	signal.Notify(gracefulStop, os.Interrupt, syscall.SIGTERM)

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

	pulseDistributor, err := pulsenetwork.NewDistributor(cfg.Pulsar.PulseDistributor)
	if err != nil {
		panic(err)
	}

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	cm.Register(cryptographyScheme, keyStore, keyProcessor, transport.NewFactory(cfg.Pulsar.DistributionTransport))
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
