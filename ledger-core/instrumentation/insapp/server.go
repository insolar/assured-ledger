// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"
	"crypto"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// New creates a one-node process.
func New(cfg configuration.Configuration, appFn AppFactoryFunc, extraComponents ...interface{}) *Server {
	return newServer(&AppInitializer{
		confProvider: &defaultConfigurationProvider{config: cfg},
		appFn:        appFn,
		extra:        extraComponents,
	})
}

func newServer(initer *AppInitializer) *Server {
	if initer == nil {
		panic(throw.IllegalValue())
	}

	return &Server{
		initer:    initer,
		waitStart: make(chan struct{}),
		waitStop:  make(chan struct{}),
		waitSig:   make(chan os.Signal, 1),
	}
}

type CertManagerFactoryFunc = func(crypto.PublicKey, cryptography.KeyProcessor, string) (*mandates.CertificateManager, error)
type KeyStoreFactoryFunc = func(string) (cryptography.KeyStore, error)

type ConfigurationProvider interface {
	Config() configuration.Configuration
	GetKeyStoreFactory() KeyStoreFactoryFunc
	GetCertManagerFactory() CertManagerFactoryFunc
}

type Server struct {
	initer    *AppInitializer
	multiFn   MultiNodeConfigFunc
	waitSig   chan os.Signal
	waitStop  chan struct{}
	waitStart chan struct{}
}

func (s *Server) GetInitializerForTest() *AppInitializer {
	if s.initer == nil {
		panic(throw.IllegalState())
	}
	initer := s.initer
	s.initer = nil

	return initer
}

func (s *Server) Serve() {
	if s.initer == nil {
		panic(throw.IllegalState())
	}
	initer := s.initer
	s.initer = nil

	var (
		baseCtx    context.Context
		baseLogger log.Logger
	)

	baseCfg := initer.confProvider.Config()
	if global.IsInitialized() {
		baseCtx, baseLogger = inslogger.InitNodeLoggerByGlobal("", "")
	} else {
		baseCtx, baseLogger = inslogger.InitGlobalNodeLogger(context.Background(), baseCfg.Log, "", "")
	}

	var ctl lifecycleController
	if s.multiFn != nil {
		ctl = s.serveMulti(baseCtx, baseLogger, initer)
	} else {
		ctl, baseLogger = s.serveMono(baseCtx, baseCfg, initer)
	}

	global.InitTicker()

	signal.Notify(s.waitSig, syscall.SIGTERM, os.Interrupt)

	go func() {
		defer close(s.waitStop)

		sig := <-s.waitSig
		baseLogger.Debug("caught sig: ", sig)
		if sig != syscall.SIGTERM {
			baseLogger.Info("stopping gracefully")

			ctl.StopGraceful(baseCtx, func(name string, err error) {
				baseLogger.Fatalf("graceful stop failed [%s]: %s", name, throw.ErrorWithStack(err))
			})
		}

		ctl.Stop(baseCtx, func(name string, err error) {
			baseLogger.Fatalf("stop failed [%s]: %s", name, throw.ErrorWithStack(err))
		})
	}()

	ctl.Start(baseCtx, func(name string, err error) {
		baseLogger.Fatalf("start failed [%s]: %s", name, throw.ErrorWithStack(err))
	})

	close(s.waitStart)
	<-s.waitStop
}

type errorFunc func(name string, err error)

type lifecycleController interface {
	Start(context.Context, errorFunc)
	Stop(context.Context, errorFunc)
	StopGraceful(context.Context, errorFunc)
}

func (s *Server) serveMono(baseCtx context.Context, cfg configuration.Configuration, initer *AppInitializer) (lifecycleController, log.Logger) {
	ctl := monoLifecycle{}

	var baseLogger log.Logger

	ctl.cm, ctl.stopFn = initer.StartComponents(baseCtx, cfg, nil,
		func(_ context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context {
			ctx, logger := inslogger.InitNodeLogger(baseCtx, cfg, nodeRef, nodeRole)
			baseCtx = ctx
			baseLogger = logger
			global.SetLogger(logger)
			return ctx
		})
	return ctl, baseLogger
}

func (s *Server) serveMulti(baseCtx context.Context, baseLogger log.Logger, initer *AppInitializer) lifecycleController {
	configs, networkFn := s.multiFn(initer.confProvider)

	ctl := &multiLifecycle{
		baseCtx:   baseCtx,
		initer:    initer,
		apps:      make(map[string]*appEntry, len(configs)),
		networkFn: networkFn,
		loggerFn: func(baseCtx context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context {
			ctx, _ := inslogger.InitNodeLogger(baseCtx, cfg, nodeRef, nodeRole)
			return ctx
		},
	}

	for i := range configs {
		ctl.addInitApp(configs[i], baseLogger)
	}

	return ctl
}

func (s *Server) Stop() {
	select {
	case s.waitSig <- syscall.SIGQUIT:
	default:
	}

	<-s.waitStop
}

func (s *Server) WaitStarted() {
	<-s.waitStart
}

type defaultConfigurationProvider struct {
	config configuration.Configuration
}

func (cp defaultConfigurationProvider) Config() configuration.Configuration {
	return cp.config
}

func (cp defaultConfigurationProvider) GetCertManagerFactory() CertManagerFactoryFunc {
	return mandates.NewManagerReadCertificate
}

func (cp defaultConfigurationProvider) GetKeyStoreFactory() KeyStoreFactoryFunc {
	return keystore.NewKeyStore
}
