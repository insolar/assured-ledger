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
	component2 "github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp/component"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp/internal/headless"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp/internal/virtual"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lmnapp"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

type Server struct {
	cfg       configuration.Configuration
	headless  bool
	appFn     component2.AppFactoryFunc
	networkFn NetworkInitFunc
	extra     []interface{}

	cm       *component.Manager
	stopFunc func()

	ctx    context.Context
	logger log.Logger
}

// New creates a one-node process.
func New(cfg configuration.Configuration, extraComponents ...interface{}) *Server {
	return &Server{
		cfg:   cfg,
		extra: extraComponents,
	}
}

// NewWithNetworkFn creates a one-node process with given networkFn
func NewWithNetworkFn(cfg configuration.Configuration, networkFn NetworkInitFunc, extraComponents ...interface{}) *Server {
	return &Server{
		cfg:       cfg,
		extra:     extraComponents,
		networkFn: networkFn,
	}
}

// NewHeadless creates a one-node headless process.
func NewHeadless(cfg configuration.Configuration, extraComponents ...interface{}) *Server {
	extraComponents = append(extraComponents, &headless.AppComponent{})
	return &Server{
		cfg:      cfg,
		headless: true,
		extra:    extraComponents,
	}
}

func (s *Server) Serve() {
	fmt.Println("Version: ", version.GetFullVersion())
	s.ctx, s.logger = inslogger.InitGlobalNodeLogger(context.Background(), s.cfg.Log, "", "")

	fmt.Printf("Starts with configuration: \n%s\n", configuration.ToString(s.cfg))

	s.cm, s.stopFunc = s.StartComponents(s.ctx, s.cfg, s.networkFn,
		func(_ context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context {
			s.ctx, s.logger = inslogger.InitNodeLogger(s.ctx, cfg, nodeRef, nodeRole)

			global.SetLogger(s.logger)

			return s.ctx
		})

	global.InitTicker()

	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	var waitChannel = make(chan bool)

	go func() {
		defer close(waitChannel)

		sig := <-gracefulStop
		s.logger.Debug("caught sig: ", sig)

		s.logger.Info("stopping gracefully")

		if err := s.cm.GracefulStop(s.ctx); err != nil {
			s.logger.Fatalf("graceful stop failed: %s", throw.ErrorWithStack(err))
		}

		s.stopFunc()

		if err := s.cm.Stop(s.ctx); err != nil {
			s.logger.Fatalf("stop failed [%d]: %s", throw.ErrorWithStack(err))
		}
	}()

	if err := s.cm.Start(s.ctx); err != nil {
		s.logger.Fatalf("start failed: %s", throw.ErrorWithStack(err))
	}

	fmt.Println("All components were started")
	<-waitChannel
}

type LoggerInitFunc = func(ctx context.Context, cfg configuration.Log, nodeRef, nodeRole string) context.Context

func (s *Server) StartComponents(ctx context.Context, cfg configuration.Configuration,
	networkFn NetworkInitFunc, loggerFn LoggerInitFunc,
) (*component.Manager, func()) {
	preComponents := s.initBootstrapComponents(ctx, cfg)
	certManager := s.initCertificateManager(ctx, cfg, preComponents)

	nodeCert := certManager.GetCertificate()
	nodeRole := nodeCert.GetRole()
	nodeRef := nodeCert.GetNodeRef().String()

	if !s.headless {
		switch nodeRole {
		case member.PrimaryRoleVirtual:
			s.appFn = virtual.AppFactory
		case member.PrimaryRoleLightMaterial:
			s.appFn = lmnapp.AppFactory
		default:
			panic("unknown role")
		}
	}

	ctx = loggerFn(ctx, cfg.Log, nodeRef, nodeRole.String())
	traceID := trace.RandID() + "_main"

	if cfg.Tracer.Jaeger.AgentEndpoint != "" {
		jaegerFlush := jaeger(ctx, cfg.Tracer.Jaeger, traceID, nodeRef, nodeRole.String())
		defer jaegerFlush()
	}

	return s.initComponents(ctx, cfg, networkFn, preComponents, certManager)
}
