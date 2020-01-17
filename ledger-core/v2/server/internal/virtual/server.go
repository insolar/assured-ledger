//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package virtual

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
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
	cfgHolder := configuration.NewHolder()
	var err error
	if len(s.cfgPath) != 0 {
		err = cfgHolder.LoadFromFile(s.cfgPath)
	} else {
		err = cfgHolder.Load()
	}
	if err != nil {
		global.Warn("failed to load configuration from file: ", err.Error())
	}

	cfg := &cfgHolder.Configuration

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

	traceID := utils.RandTraceID() + "_main"
	ctx, logger := inslogger.InitNodeLogger(ctx, cfg.Log, nodeRef, nodeRole)
	global.InitTicker()

	if cfg.Tracer.Jaeger.AgentEndpoint != "" {
		jaegerFlush := internal.Jaeger(ctx, cfg.Tracer.Jaeger, traceID, nodeRef, nodeRole)
		defer jaegerFlush()
	}

	cm, stopWatermill := initComponents(
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

		stopWatermill()

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
