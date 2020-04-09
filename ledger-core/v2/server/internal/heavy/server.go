// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package heavy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/assured-ledger/ledger-core/v2/application"
	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/server/internal"
	"github.com/insolar/assured-ledger/ledger-core/v2/version"
)

type Server struct {
	cfgPath        string
	genesisCfgPath string
}

func New(cfgPath string, genesisCfgPath string) *Server {
	return &Server{
		cfgPath:        cfgPath,
		genesisCfgPath: genesisCfgPath,
	}
}

func (s *Server) Serve() {
	cfgHolder := configuration.NewHolder(s.cfgPath)
	err := cfgHolder.Load()
	if err != nil {
		global.Fatalf("failed to load configuration: %v", err.Error())
	}

	b, err := ioutil.ReadFile(s.genesisCfgPath)
	if err != nil {
		global.Fatalf("failed to load genesis configuration from file: %v", s.genesisCfgPath)
	}
	var genesisCfg application.GenesisHeavyConfig
	err = json.Unmarshal(b, &genesisCfg)
	if err != nil {
		global.Fatalf("failed to pares genesis configuration from file: %v", s.genesisCfgPath)
	}

	cfg := cfgHolder.Configuration

	fmt.Println("Version: ", version.GetFullVersion())
	fmt.Println("Starts with configuration:\n", configuration.ToString(cfgHolder.Configuration))

	var (
		ctx         = context.Background()
		mainTraceID = utils.RandTraceID() + "_main"
		logger      log.Logger
	)
	{
		var (
			nodeRole      = "heavy_material"
			nodeReference = ""
		)
		certManager, err := initTemporaryCertificateManager(ctx, cfg)
		if err != nil {
			global.Warn("Failed to initialize nodeRef, nodeRole fields: ", err.Error())
		} else {
			nodeRole = certManager.GetCertificate().GetRole().String()
			nodeReference = certManager.GetCertificate().GetNodeRef().String()
		}

		ctx, logger = inslogger.InitNodeLogger(ctx, cfg.Log, nodeReference, nodeRole)
		global.InitTicker()
	}

	cmp, err := newComponents(ctx, *cfg, genesisCfg)
	fatal(ctx, err, "failed to create components")

	if cfg.Tracer.Jaeger.AgentEndpoint != "" {
		jaegerFlush := internal.Jaeger(ctx, cfg.Tracer.Jaeger, mainTraceID, cmp.NodeRef, cmp.NodeRole)
		defer jaegerFlush()
	}

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	var waitChannel = make(chan bool)

	go func() {
		sig := <-gracefulStop
		logger.Debug("caught sig: ", sig)

		logger.Warn("GRACEFUL STOP APP")
		err = cmp.Stop(ctx)
		fatal(ctx, err, "failed to graceful stop components")
		close(waitChannel)
	}()

	err = cmp.Start(ctx)
	fatal(ctx, err, "failed to start components")
	fmt.Println("All components were started")
	<-waitChannel
}

func fatal(ctx context.Context, err error, message string) {
	if err == nil {
		return
	}
	inslogger.FromContext(ctx).Fatalf("%v: %v", message, err.Error())
}
