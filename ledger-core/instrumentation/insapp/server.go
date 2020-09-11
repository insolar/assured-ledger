// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"
	"crypto"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

type CertManagerFactory func(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, certPath string) (*mandates.CertificateManager, error)
type KeyStoreFactory func(path string) (cryptography.KeyStore, error)

type ConfigurationProvider interface {
	Config() configuration.Configuration
	GetKeyStoreFactory() KeyStoreFactory
	GetCertManagerFactory() CertManagerFactory
}

type Server struct {
	appFn        AppFactoryFunc
	multiFn      MultiNodeConfigFunc
	extra        []interface{}
	confProvider ConfigurationProvider
}

type defaultConfigurationProvider struct {
	config configuration.Configuration
}

func (cp defaultConfigurationProvider) Config() configuration.Configuration {
	return cp.config
}

func (cp defaultConfigurationProvider) GetCertManagerFactory() CertManagerFactory {
	return mandates.NewManagerReadCertificate
}

func (cp defaultConfigurationProvider) GetKeyStoreFactory() KeyStoreFactory {
	return keystore.NewKeyStore
}

// New creates a one-node process.
func New(cfg configuration.Configuration, appFn AppFactoryFunc, extraComponents ...interface{}) *Server {
	return &Server{
		confProvider: &defaultConfigurationProvider{config: cfg},
		appFn:        appFn,
		extra:        extraComponents,
	}
}

func (s *Server) Serve() {
	var (
		configs   []configuration.Configuration
		networkFn NetworkInitFunc
	)

	fmt.Println("Version: ", version.GetFullVersion())

	if s.multiFn != nil {
		baseConfig := s.confProvider.Config()
		fmt.Println("Starts with multi-node configuration base:\n", configuration.ToString(&baseConfig))
		configs, networkFn = s.multiFn(s.confProvider)
	} else {
		configs = append(configs, s.confProvider.Config())
	}

	baseCtx, baseLogger := inslogger.InitGlobalNodeLogger(context.Background(), s.confProvider.Config().Log, "", "")

	n := len(configs)

	cms := make([]*component.Manager, 0, n)
	stops := make([]func(), 0, n)
	contexts := make([]context.Context, 0, n)

	for i := range configs {
		cfg := configs[i]
		fmt.Printf("Starts with configuration [%d/%d]:\n%s\n", i+1, n, configuration.ToString(&cfg))

		cm, stopFunc := s.StartComponents(baseCtx, cfg, networkFn,
			func(_ context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context {
				ctx, logger := inslogger.InitNodeLogger(baseCtx, cfg, nodeRef, nodeRole)
				contexts = append(contexts, ctx)

				if n == 1 {
					baseCtx = ctx
					baseLogger = logger
					global.SetLogger(logger)
				}

				return ctx
			})
		cms = append(cms, cm)
		stops = append(stops, stopFunc)
	}

	s.confProvider = nil

	global.InitTicker()

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	var waitChannel = make(chan bool)

	go func() {
		defer close(waitChannel)

		sig := <-gracefulStop
		baseLogger.Debug("caught sig: ", sig)

		baseLogger.Info("stopping gracefully")

		for i, cm := range cms {
			if err := cm.GracefulStop(baseCtx); err != nil {
				baseLogger.Fatalf("graceful stop failed [%d]: %s", i, throw.ErrorWithStack(err))
			}
		}

		for _, stopFn := range stops {
			if stopFn != nil {
				stopFn()
			}
		}

		for i, cm := range cms {
			if err := cm.Stop(contexts[i]); err != nil {
				baseLogger.Fatalf("stop failed [%d]: %s", i, throw.ErrorWithStack(err))
			}
		}
	}()

	for i, cm := range cms {
		if err := cm.Start(contexts[i]); err != nil {
			baseLogger.Fatalf("start failed [%d]: %s", i, throw.ErrorWithStack(err))
		}
	}

	fmt.Println("All components were started")
	<-waitChannel
}

type LoggerInitFunc = func(ctx context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context

func (s *Server) StartComponents(ctx context.Context, cfg configuration.Configuration,
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
