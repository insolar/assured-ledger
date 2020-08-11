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
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

type Server struct {
	cfgPath string
	appFn   AppFactoryFunc
	extra   []interface{}
}

func New(cfgPath string, appFn AppFactoryFunc, extraComponents ...interface{}) *Server {
	return &Server{
		cfgPath: cfgPath,
		appFn:   appFn,
		extra:   extraComponents,
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

func (s *Server) Serve() {
	cfg := s.readConfig(s.cfgPath)

	fmt.Println("Version: ", version.GetFullVersion())
	fmt.Println("Starts with configuration:\n", configuration.ToString(&cfg))

	ctx := context.Background()
	var logger log.Logger

	cm, stopWatermill := s.StartComponents(ctx, cfg, func(_ context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context {
		ctx, logger = inslogger.InitNodeLogger(ctx, cfg, nodeRef, nodeRole)
		global.InitTicker()
		return ctx
	})

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	var waitChannel = make(chan bool)

	go func() {
		sig := <-gracefulStop
		logger.Debug("caught sig: ", sig)

		logger.Warn("GRACEFUL STOP APP")
		// th.Leave(ctx, 10) TODO: is actual ??
		logger.Info("main leave ends ")

		err := cm.GracefulStop(ctx)
		checkError(ctx, err, "failed to graceful stop components")

		if stopWatermill != nil {
			stopWatermill()
		}

		err = cm.Stop(ctx)
		checkError(ctx, err, "failed to stop components")
		close(waitChannel)
	}()

	err := cm.Start(ctx)
	checkError(ctx, err, "failed to start components")
	fmt.Println("All components were started")
	<-waitChannel
}

type LoggerInitFunc = func(ctx context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context

func (s *Server) StartComponents(ctx context.Context, cfg configuration.Configuration, loggerFn LoggerInitFunc) (*component.Manager, func()) {
	preComponents := s.initBootstrapComponents(ctx, cfg)
	certManager := s.initCertificateManager(ctx, cfg, preComponents)

	nodeRole := certManager.GetCertificate().GetRole().String()
	nodeRef := certManager.GetCertificate().GetNodeRef().String()

	ctx = loggerFn(ctx, cfg.Log, nodeRef, nodeRole)
	traceID := trace.RandID() + "_main"

	if cfg.Tracer.Jaeger.AgentEndpoint != "" {
		jaegerFlush := jaeger(ctx, cfg.Tracer.Jaeger, traceID, nodeRef, nodeRole)
		defer jaegerFlush()
	}

	return s.initComponents(ctx, cfg, preComponents, certManager, nodeRole)
}

func checkError(ctx context.Context, err error, message string) {
	if err == nil {
		return
	}
	inslogger.FromContext(ctx).Fatalf("%v: %v", message, err.Error())
}

