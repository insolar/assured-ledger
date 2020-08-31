// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat/memstor"
	"github.com/insolar/assured-ledger/ledger-core/application/api"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/servicenetwork"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type preComponents struct {
	CryptographyService        cryptography.Service
	PlatformCryptographyScheme cryptography.PlatformCryptographyScheme
	KeyStore                   cryptography.KeyStore
	KeyProcessor               cryptography.KeyProcessor
	CryptoScheme               crypto.PlatformScheme
}

func (s *Server) initBootstrapComponents(ctx context.Context, cfg configuration.Configuration) preComponents {
	earlyComponents := component.NewManager(nil)
	logger := inslogger.FromContext(ctx)
	earlyComponents.SetLogger(logger)

	keyStore, err := keystore.NewKeyStore(cfg.KeysPath)
	checkError(ctx, err, "failed to load KeyStore: ")

	platformCryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
	keyProcessor := platformpolicy.NewKeyProcessor()

	cryptographyService := platformpolicy.NewCryptographyService()
	earlyComponents.Register(platformCryptographyScheme, keyStore)
	earlyComponents.Inject(cryptographyService, keyProcessor)

	return preComponents{
		CryptographyService:        cryptographyService,
		PlatformCryptographyScheme: platformCryptographyScheme,
		KeyStore:                   keyStore,
		KeyProcessor:               keyProcessor,
		CryptoScheme:               legacyadapter.New(platformCryptographyScheme, keyProcessor, keyStore),
	}
}

func (s *Server) initCertificateManager(ctx context.Context, cfg configuration.Configuration, comps preComponents) *mandates.CertificateManager {
	var certManager *mandates.CertificateManager
	var err error

	publicKey, err := comps.CryptographyService.GetPublicKey()
	checkError(ctx, err, "failed to retrieve node public key")

	certManager, err = mandates.NewManagerReadCertificate(publicKey, comps.KeyProcessor, cfg.CertificatePath)
	checkError(ctx, err, "failed to start Certificate")

	return certManager
}

// initComponents creates and links all insolard components
func (s *Server) initComponents(ctx context.Context, cfg configuration.Configuration, networkFn NetworkInitFunc,
	comps preComponents, certManager nodeinfo.CertificateManager,
) (*component.Manager, func()) {
	cm := component.NewManager(nil)
	logger := inslogger.FromContext(ctx)
	cm.SetLogger(logger)

	cm.Register(
		comps.PlatformCryptographyScheme,
		comps.KeyStore,
		comps.CryptographyService,
		comps.KeyProcessor,
		certManager,
	)

	var nw NetworkSupport
	var ns network.Status

	if networkFn == nil {
		nsn, err := servicenetwork.NewServiceNetwork(cfg, cm)
		checkError(ctx, err, "failed to start ServiceNetwork")
		cm.Register(nsn)

		nw = nsn
		ns = nsn
	} else {
		var err error
		nw, ns, err = networkFn(cfg, cm)
		checkError(ctx, err, "failed to start ServiceNetwork by factory")
		cm.Register(nw)
	}

	nodeCert := certManager.GetCertificate()
	nodeRole := nodeCert.GetRole()

	roleName := nodeRole.String()
	metricsComp := metrics.NewMetrics(cfg.Metrics, metrics.GetInsolarRegistry(roleName), roleName)

	pulses := memstor.NewStorageMem()
	pm := NewPulseManager()

	availabilityChecker := api.NewNetworkChecker(cfg.AvailabilityChecker)

	mr := nw.CreateMessagesRouter(ctx)
	// TODO introspection support, cfg.Introspection

	cm.Register(
		pulses,
		pm,
		availabilityChecker,
		metricsComp,
	)

	cm.Register(s.extra...)

	var appComponent AppComponent

	if s.appFn != nil {
		affine := affinity.NewAffinityHelper(certManager.GetCertificate().GetNodeRef())
		cm.Register(affine)

		if ns != nil {
			API, err := api.NewRunner(&cfg.APIRunner,
				certManager, nw, nw, pulses, affine, ns, availabilityChecker)
			checkError(ctx, err, "failed to start ApiRunner")

			AdminAPIRunner, err := api.NewRunner(&cfg.AdminAPIRunner,
				certManager, nw, nw, pulses, affine, ns, availabilityChecker)
			checkError(ctx, err, "failed to start AdminAPIRunner")

			APIWrapper := api.NewWrapper(API, AdminAPIRunner)

			cm.Register(APIWrapper)
		}

		appComponents := AppComponents{
			LocalNodeRef:  nodeCert.GetNodeRef(),
			LocalNodeRole: nodeRole,

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
	pm.AddDispatcher(bd)

	stopFn := mr.SubscribeForMessages(bd.Process)

	return cm, stopFn
}

func checkError(ctx context.Context, err error, message string) {
	if err != nil {
		inslogger.FromContext(ctx).Fatalf("%v: %v", message, throw.ErrorWithStack(err))
	}
}

