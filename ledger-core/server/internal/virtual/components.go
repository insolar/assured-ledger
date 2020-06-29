// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtual

import (
	"context"
	"io"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/application/api"
	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodestorage"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/logwatermill"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/server/internal"
	"github.com/insolar/assured-ledger/ledger-core/virtual"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/pulsemanager"
)

type bootstrapComponents struct {
	CryptographyService        cryptography.Service
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme
	KeyStore                   cryptography.KeyStore
	KeyProcessor               cryptography.KeyProcessor
}

func initBootstrapComponents(ctx context.Context, cfg configuration.Configuration) bootstrapComponents {
	earlyComponents := component.NewManager(nil)
	earlyComponents.SetLogger(global.Logger())

	keyStore, err := keystore.NewKeyStore(cfg.KeysPath)
	checkError(ctx, err, "failed to load KeyStore: ")

	platformCryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
	keyProcessor := platformpolicy.NewKeyProcessor()

	cryptographyService := platformpolicy.NewCryptographyService()
	earlyComponents.Register(platformCryptographyScheme, keyStore)
	earlyComponents.Inject(cryptographyService, keyProcessor)

	return bootstrapComponents{
		CryptographyService:        cryptographyService,
		PlatformCryptographyScheme: platformCryptographyScheme,
		KeyStore:                   keyStore,
		KeyProcessor:               keyProcessor,
	}
}

func initCertificateManager(
	ctx context.Context,
	cfg configuration.Configuration,
	cryptographyService cryptography.Service,
	keyProcessor cryptography.KeyProcessor,
) *mandates.CertificateManager {
	var certManager *mandates.CertificateManager
	var err error

	publicKey, err := cryptographyService.GetPublicKey()
	checkError(ctx, err, "failed to retrieve node public key")

	certManager, err = mandates.NewManagerReadCertificate(publicKey, keyProcessor, cfg.CertificatePath)
	checkError(ctx, err, "failed to start Certificate")

	return certManager
}

// initComponents creates and links all insolard components
func initComponents(
	ctx context.Context,
	cfg configuration.Configuration,
	cryptographyService cryptography.Service,
	pcs cryptography.PlatformCryptographyScheme,
	keyStore cryptography.KeyStore,
	keyProcessor cryptography.KeyProcessor,
	certManager node.CertificateManager,

) (*component.Manager, func()) {
	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	// Watermill.
	var (
		wmLogger   *logwatermill.WatermillLogAdapter
		publisher  message.Publisher
		subscriber message.Subscriber
	)
	{
		wmLogger = logwatermill.NewWatermillLogAdapter(inslogger.FromContext(ctx))
		pubsub := gochannel.NewGoChannel(gochannel.Config{}, wmLogger)
		subscriber = pubsub
		publisher = pubsub
		// Wrapped watermill Publisher for introspection.
		publisher = internal.PublisherWrapper(ctx, cm, cfg.Introspection, publisher)
	}

	nw, err := servicenetwork.NewServiceNetwork(cfg, cm)
	checkError(ctx, err, "failed to start Network")

	metricsComp := metrics.NewMetrics(cfg.Metrics, metrics.GetInsolarRegistry("virtual"), "virtual")

	jc := jet.NewAffinityHelper(cfg.Ledger.LightChainLimit, certManager.GetCertificate().GetNodeRef())
	pulses := pulsestor.NewStorageMem()

	messageSender := messagesender.NewDefaultService(publisher, jc, pulses)

	runnerService := runner.NewService()
	err = runnerService.Init()
	checkError(ctx, err, "failed to initialize Runner Service")

	virtualDispatcher := virtual.NewDispatcher()

	virtualDispatcher.Runner = runnerService
	virtualDispatcher.MessageSender = messageSender
	virtualDispatcher.Affinity = jc
	virtualDispatcher.AuthenticationService = authentication.NewService(ctx, jc)

	availabilityChecker := api.NewNetworkChecker(cfg.AvailabilityChecker)

	API, err := api.NewRunner(
		&cfg.APIRunner,
		certManager,
		nw,
		nw,
		pulses,
		jc,
		nw,
		availabilityChecker,
	)
	checkError(ctx, err, "failed to start ApiRunner")

	AdminAPIRunner, err := api.NewRunner(
		&cfg.AdminAPIRunner,
		certManager,
		nw,
		nw,
		pulses,
		jc,
		nw,
		availabilityChecker,
	)
	checkError(ctx, err, "failed to start AdminAPIRunner")

	APIWrapper := api.NewWrapper(API, AdminAPIRunner)

	pm := pulsemanager.NewPulseManager()

	cm.Register(
		pcs,
		keyStore,
		cryptographyService,
		keyProcessor,
		certManager,
		virtualDispatcher,
		runnerService,
		APIWrapper,
		testwalletapi.NewTestWalletServer(cfg.TestWalletAPI, virtualDispatcher, pulses),
		availabilityChecker,
		nw,
		pm,
	)

	components := []interface{}{
		publisher,
		jc,
		pulses,
		nodestorage.NewStorage(),
	}
	components = append(components, []interface{}{
		metricsComp,
		cryptographyService,
		keyProcessor,
	}...)

	cm.Inject(components...)

	err = cm.Init(ctx)
	checkError(ctx, err, "failed to init components")

	// this should be done after Init due to inject
	pm.AddDispatcher(virtualDispatcher.FlowDispatcher)

	return cm, startWatermill(
		ctx, wmLogger, subscriber,
		nw.SendMessageHandler,
		virtualDispatcher.FlowDispatcher.Process,
	)
}

func startWatermill(
	ctx context.Context,
	logger watermill.LoggerAdapter,
	sub message.Subscriber,
	outHandler, inHandler message.NoPublishHandlerFunc,
) func() {
	inRouter, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}
	outRouter, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

	outRouter.AddNoPublisherHandler(
		"OutgoingHandler",
		defaults.TopicOutgoing,
		sub,
		outHandler,
	)

	inRouter.AddNoPublisherHandler(
		"IncomingHandler",
		defaults.TopicIncoming,
		sub,
		inHandler,
	)
	startRouter(ctx, inRouter)
	startRouter(ctx, outRouter)

	return stopWatermill(ctx, inRouter, outRouter)
}

func stopWatermill(ctx context.Context, routers ...io.Closer) func() {
	return func() {
		for _, r := range routers {
			err := r.Close()
			if err != nil {
				inslogger.FromContext(ctx).Error("Error while closing router", err)
			}
		}
	}
}

func startRouter(ctx context.Context, router *message.Router) {
	go func() {
		if err := router.Run(ctx); err != nil {
			inslogger.FromContext(ctx).Error("Error while running router", err)
		}
	}()
	<-router.Running()
}
