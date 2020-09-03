// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"
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
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

type CertManagerFactory func(ctx context.Context, certPath string, comps PreComponents) nodeinfo.CertificateManager
type KeyStoreFactory func(path string) (cryptography.KeyStore, error)

type Server struct {
	cfgPath            string
	appFn              AppFactoryFunc
	multiFn            MultiNodeConfigFunc
	extra              []interface{}
	certManagerFactory CertManagerFactory
	keyStoreFactory    KeyStoreFactory
}

// New creates a one-node process.
func New(cfgPath string, appFn AppFactoryFunc, extraComponents ...interface{}) *Server {
	return &Server{
		cfgPath:            cfgPath,
		appFn:              appFn,
		extra:              extraComponents,
		certManagerFactory: initCertificateManager,
		keyStoreFactory:    keystore.NewKeyStore,
	}
}

func (s *Server) readConfig(cfgPath string) configuration.Configuration {
	cfgHolder := configuration.NewHolder(cfgPath)
	err := cfgHolder.Load()
	if err != nil {
		global.Warn("failed to load configuration from file: ", err.Error())
	}
	return *cfgHolder.Configuration
}

func (s *Server) SetKeyStoreFactory(factoryFn KeyStoreFactory) {
	s.keyStoreFactory = factoryFn
}

func (s *Server) SetCertManagerFactory(factoryFn CertManagerFactory) {
	s.certManagerFactory = factoryFn
}

func (s *Server) Serve() {
	baseCfg := s.readConfig(s.cfgPath)
	var configs []configuration.Configuration
	var networkFn NetworkInitFunc

	fmt.Println("Version: ", version.GetFullVersion())
	if s.multiFn != nil {
		fmt.Println("Starts with multi-node configuration base:\n", configuration.ToString(&baseCfg))
		configs, networkFn = s.multiFn(s.cfgPath, baseCfg)
	} else {
		configs = append(configs, baseCfg)
	}

	baseCtx, baseLogger := inslogger.InitGlobalNodeLogger(context.Background(), baseCfg.Log, "", "")

	n := len(configs)
	cms := make([]*component.Manager, 0, n)
	stops := make([]func(), 0, n)
	contexts := make([]context.Context, 0, n)

	for i, cfg := range configs {
		fmt.Printf("Starts with configuration [%d/%d]:\n%s\n", i+1, n, configuration.ToString(&configs[i]))

		cm, stopWatermill := s.StartComponents(baseCtx, cfg, networkFn,
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
		stops = append(stops, stopWatermill)
	}

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
			if err := cm.GracefulStop(contexts[i]); err != nil {
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
	certManager := s.certManagerFactory(ctx, cfg.CertificatePath, preComponents)

	nodeCert := certManager.GetCertificate()
	nodeRole := nodeCert.GetRole()
	nodeRef := nodeCert.GetNodeRef().String()

	ctx = loggerFn(ctx, cfg.Log, nodeRef, nodeRole.String())
	traceID := trace.RandID() + "_main"

	if cfg.Tracer.Jaeger.AgentEndpoint != "" {
		jaegerFlush := jaeger(ctx, cfg.Tracer.Jaeger, traceID, nodeRef, nodeRole.String())
		defer jaegerFlush()
	}

	return s.initComponents(ctx, cfg, networkFn, preComponents, certManager)
}
