// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package headless

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/server/internal"
	"github.com/insolar/assured-ledger/ledger-core/v2/version"
)

type Server struct {
	cfgPath string
}

func New(cfgPath string) *Server {
	return &Server{
		cfgPath: cfgPath,
	}
}

func (s *Server) Serve() {
	cfgHolder := configuration.NewHolder(s.cfgPath)
	err := cfgHolder.Load()
	if err != nil {
		global.Warn("failed to load configuration from file: ", err.Error())
	}

	cfg := cfgHolder.Configuration

	fmt.Println("Version: ", version.GetFullVersion())
	fmt.Println("Starts with configuration:\n", configuration.ToString(cfgHolder.Configuration))

	ctx := context.Background()
	bootstrapComponents := initBootstrapComponents(ctx, *cfg)
	certManager := initCertificateManager(
		ctx,
		*cfg,
		bootstrapComponents.CryptographyService,
		bootstrapComponents.KeyProcessor,
	)

	nodeRole := certManager.GetCertificate().GetRole().String()
	nodeRef := certManager.GetCertificate().GetNodeRef().String()

	traceID := trace.RandID() + "_main"
	ctx, logger := inslogger.InitNodeLogger(ctx, cfg.Log, nodeRef, nodeRole)
	global.InitTicker()

	if cfg.Tracer.Jaeger.AgentEndpoint != "" {
		jaegerFlush := internal.Jaeger(ctx, cfg.Tracer.Jaeger, traceID, nodeRef, nodeRole)
		defer jaegerFlush()
	}

	cm := initComponents(
		ctx,
		*cfg,
		bootstrapComponents.CryptographyService,
		bootstrapComponents.PlatformCryptographyScheme,
		bootstrapComponents.KeyStore,
		bootstrapComponents.KeyProcessor,
		certManager,
	)

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

		err = cm.GracefulStop(ctx)
		checkError(ctx, err, "failed to graceful stop components")

		err = cm.Stop(ctx)
		checkError(ctx, err, "failed to stop components")
		close(waitChannel)
	}()

	err = cm.Start(ctx)
	checkError(ctx, err, "failed to start components")
	fmt.Println("All components were started")
	<-waitChannel
}

func checkError(ctx context.Context, err error, message string) {
	if err == nil {
		return
	}
	inslogger.FromContext(ctx).Fatalf("%v: %v", message, err.Error())
}
