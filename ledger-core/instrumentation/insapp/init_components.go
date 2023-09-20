package insapp

import (
	"context"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/application/api"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type AppInitializer struct {
	appFn        AppFactoryFunc
	extra        []interface{}
	confProvider ConfigurationProvider
}

type LoggerInitFunc = func(ctx context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context

func (s *AppInitializer) StartComponents(ctx context.Context, cfg configuration.Configuration,
	networkFn NetworkInitFunc, loggerFn LoggerInitFunc,
) (*component.Manager, func()) {
	preComponents := s.initBootstrapComponents(ctx, cfg)

	nodeCert := preComponents.CertificateManager.GetCertificate()
	nodeRole := nodeCert.GetRole()
	nodeRef := nodeCert.GetNodeRef().String()

	ctx = loggerFn(ctx, cfg.Log, nodeRef, nodeRole.String())
	traceID := trace.RandID() + "_main"

	if cfg.Tracer.Jaeger.AgentEndpoint != "" {
		jaegerFlush := jaeger(ctx, cfg.Tracer.Jaeger, traceID, nodeRef, nodeRole.String())
		defer jaegerFlush()
	}

	return s.initComponents(ctx, cfg, networkFn, preComponents)
}

func (s *AppInitializer) initBootstrapComponents(ctx context.Context, cfg configuration.Configuration) PreComponents {
	earlyComponents := component.NewManager(nil)
	logger := inslogger.FromContext(ctx)
	earlyComponents.SetLogger(logger)

	keyStore, err := s.confProvider.GetKeyStoreFactory()(cfg.KeysPath)
	checkError(ctx, err, "failed to load KeyStore: ")

	platformCryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
	keyProcessor := platformpolicy.NewKeyProcessor()

	cryptographyService := platformpolicy.NewCryptographyService()
	earlyComponents.Register(platformCryptographyScheme, keyStore)
	earlyComponents.Inject(cryptographyService, keyProcessor)

	publicKey, err := cryptographyService.GetPublicKey()
	checkError(ctx, err, "failed to retrieve node public key")

	certManager, err := s.confProvider.GetCertManagerFactory()(publicKey, keyProcessor, cfg.CertificatePath)
	checkError(ctx, err, "failed to start Certificate")

	return PreComponents{
		CryptographyService:        cryptographyService,
		PlatformCryptographyScheme: platformCryptographyScheme,
		KeyStore:                   keyStore,
		KeyProcessor:               keyProcessor,
		CryptoScheme:               legacyadapter.New(platformCryptographyScheme, keyProcessor, keyStore),
		CertificateManager:         certManager,
	}
}

// initComponents creates and links all insolard components
func (s *AppInitializer) initComponents(ctx context.Context, cfg configuration.Configuration, networkFn NetworkInitFunc,
	comps PreComponents) (*component.Manager, func()) {
	cm := component.NewManager(nil)
	logger := inslogger.FromContext(ctx)
	cm.SetLogger(logger)

	cm.Register(
		comps.PlatformCryptographyScheme,
		comps.KeyStore,
		comps.CryptographyService,
		comps.KeyProcessor,
		comps.CertificateManager,
	)

	var nw NetworkSupport
	var ns network.Status

	var pulses beat.History
	var addDispatcherFn func(beat.Dispatcher)

	if networkFn == nil {
		nsn, err := servicenetwork.NewServiceNetwork(cfg, cm)
		checkError(ctx, err, "failed to start ServiceNetwork")
		cm.Register(nsn)

		nw = nsn
		ns = nsn

		pulses = memstor.NewStorageMem()
		pm := NewPulseManager()
		cm.Register(pm)

		addDispatcherFn = pm.AddDispatcher
	} else {
		var err error
		nw, ns, err = networkFn(cfg, cm)
		checkError(ctx, err, "failed to start ServiceNetwork by factory")
		cm.Register(nw)
		addDispatcherFn = nw.AddDispatcher
		pulses = nw.GetBeatHistory()
	}

	nodeCert := comps.CertificateManager.GetCertificate()
	nodeRole := nodeCert.GetRole()

	roleName := nodeRole.String()
	metricsComp := metrics.NewMetrics(cfg.Metrics, metrics.GetInsolarRegistry(roleName), roleName)

	availabilityChecker := api.NewDummyNetworkChecker(cfg.AvailabilityChecker)

	mr := nw.CreateMessagesRouter(ctx)
	// TODO introspection support, cfg.Introspection

	cm.Register(
		pulses,
		availabilityChecker,
		metricsComp,
	)

	cm.Register(s.extra...)

	var appComponent AppComponent

	if s.appFn != nil {
		affine := affinity.NewAffinityHelper(nodeCert.GetNodeRef())
		cm.Register(affine)

		if ns != nil {
			API, err := api.NewRunner(&cfg.APIRunner, logger,
				comps.CertificateManager, nw, nw, pulses, affine, ns, availabilityChecker)
			checkError(ctx, err, "failed to start ApiRunner")

			AdminAPIRunner, err := api.NewRunner(&cfg.AdminAPIRunner, logger,
				comps.CertificateManager, nw, nw, pulses, affine, ns, availabilityChecker)
			checkError(ctx, err, "failed to start AdminAPIRunner")

			APIWrapper := api.NewWrapper(API, AdminAPIRunner)

			cm.Register(APIWrapper)
		}

		appComponents := AppComponents{
			LocalNodeRef:  nodeCert.GetNodeRef(),
			LocalNodeRole: nodeRole,
			Certificate:   comps.CertificateManager.GetCertificate(),

			BeatHistory:    pulses,
			AffinityHelper: affine,
			MessageSender:  mr.CreateMessageSender(affine, pulses),
			CryptoScheme:   comps.CryptoScheme,
		}

		var err error
		appComponent, err = s.appFn(ctx, cfg, appComponents)
		checkError(ctx, err, "failed to start AppCompartment")

		cm.Register(appComponent)
	}
	cm.Inject()

	err := cm.Init(ctx)
	checkError(ctx, err, "failed to init components")

	if appComponent == nil {
		return cm, nil
	}

	// must be after Init
	bd := appComponent.GetBeatDispatcher()
	addDispatcherFn(bd)

	stopFn := mr.SubscribeForMessages(bd.Process)

	return cm, stopFn
}

func checkError(ctx context.Context, err error, message string) {
	if err != nil {
		inslogger.FromContext(ctx).Fatalf("%v: %v", message, throw.ErrorWithStack(err))
	}
}
