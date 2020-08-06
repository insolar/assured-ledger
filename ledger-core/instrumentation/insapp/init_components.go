// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"
	"io"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/application/api"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor/memstor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/logwatermill"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"
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
	appFn AppFactoryFunc,
	cryptographyService cryptography.Service,
	pcs cryptography.PlatformCryptographyScheme,
	keyStore cryptography.KeyStore,
	keyProcessor cryptography.KeyProcessor,
	certManager nodeinfo.CertificateManager,

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
		publisher = publisherWrapper(ctx, cm, cfg.Introspection, publisher)
	}


	nw, err := servicenetwork.NewServiceNetwork(cfg, cm)
	checkError(ctx, err, "failed to start Network")

	metricsComp := metrics.NewMetrics(cfg.Metrics, metrics.GetInsolarRegistry("virtual"), "virtual")

	affine := affinity.NewAffinityHelper(certManager.GetCertificate().GetNodeRef())
	pulses := memstor.NewStorageMem()

	availabilityChecker := api.NewNetworkChecker(cfg.AvailabilityChecker)

	API, err := api.NewRunner(
		&cfg.APIRunner,
		certManager,
		nw,
		nw,
		pulses,
		affine,
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
		affine,
		nw,
		availabilityChecker,
	)
	checkError(ctx, err, "failed to start AdminAPIRunner")

	APIWrapper := api.NewWrapper(API, AdminAPIRunner)

	pm := insconveyor.NewPulseManager()

	appComponents := AppComponents{
		BeatHistory: pulses,
		AffinityHelper: affine,
		MessageSender: messagesender.NewDefaultService(publisher, affine, pulses),
	}
	appComponent := appFn(cfg, appComponents)

	cm.Register(
		pcs,
		keyStore,
		cryptographyService,
		keyProcessor,
		certManager,
		APIWrapper,
		availabilityChecker,
		nw,
		pm,
		appComponent,
	)

	cm.Inject(
		publisher,
		affine,
		pulses,
		metricsComp,
		cryptographyService,
		keyProcessor,
	)

	pm.AddDispatcher(appComponent.GetBeatDispatcher())
	// ??? // this should be done after Init due to inject
	// pm.AddDispatcher(virtualDispatcher.FlowDispatcher)

	err = cm.Init(ctx)
	checkError(ctx, err, "failed to init components")

	stopWatermillFn := startWatermill(
		ctx, wmLogger, subscriber,
		nw.SendMessageHandler,
		appComponent.GetMessageHandler(),
	)

	return cm, stopWatermillFn
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
