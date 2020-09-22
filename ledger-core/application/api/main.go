// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"crypto"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/insolar/rpc/v2"
	jsonrpc "github.com/insolar/rpc/v2/json2"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/application/api/seedmanager"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// Runner implements Component for API
type Runner struct {
	CertificateManager nodeinfo.CertificateManager
	// nolint
	NodeNetwork         beat.NodeNetwork
	CertificateGetter   nodeinfo.CertificateGetter
	PulseAccessor       beat.History
	JetCoordinator      affinity.Helper
	NetworkStatus       network.Status
	AvailabilityChecker AvailabilityChecker

	handler       http.Handler
	server        *http.Server
	logger        log.Logger
	rpcServer     *rpc.Server
	cfg           *configuration.APIRunner
	keyCache      map[string]crypto.PublicKey
	cacheLock     *sync.RWMutex
	SeedManager   *seedmanager.SeedManager
	SeedGenerator seedmanager.SeedGenerator
}

func checkConfig(cfg *configuration.APIRunner) error {
	if cfg == nil {
		return throw.New("[ checkConfig ] config is nil")
	}
	if cfg.Address == "" {
		return throw.New("[ checkConfig ] Address must not be empty")
	}
	if len(cfg.RPC) == 0 {
		return throw.New("[ checkConfig ] RPC must exist")
	}
	if len(cfg.SwaggerPath) == 0 {
		return throw.New("[ checkConfig ] Missing openAPI spec file path")
	}

	return nil
}

func (ar *Runner) registerPublicServices(rpcServer *rpc.Server) error {
	err := rpcServer.RegisterService(NewNodeService(ar), "node")
	if err != nil {
		return throw.W(err, "[ registerServices ] Can't RegisterService: node")
	}

	return nil
}

// NewRunner is C-tor for API Runner
func NewRunner(cfg *configuration.APIRunner,
	logger log.Logger,
	certificateManager nodeinfo.CertificateManager,
	// nolint
	nodeNetwork beat.NodeNetwork,
	certificateGetter nodeinfo.CertificateGetter,
	pulseAccessor beat.History,
	jetCoordinator affinity.Helper,
	networkStatus network.Status,
	availabilityChecker AvailabilityChecker,
) (*Runner, error) {

	err := checkConfig(cfg)
	if err != nil {
		return nil, throw.W(err, "[ NewAPIRunner ] Bad config")
	}

	rpcServer := rpc.NewServer()
	ar := Runner{
		CertificateManager:  certificateManager,
		NodeNetwork:         nodeNetwork,
		CertificateGetter:   certificateGetter,
		PulseAccessor:       pulseAccessor,
		JetCoordinator:      jetCoordinator,
		NetworkStatus:       networkStatus,
		AvailabilityChecker: availabilityChecker,
		server:              &http.Server{Addr: cfg.Address},
		logger:              logger,
		rpcServer:           rpcServer,
		cfg:                 cfg,
		keyCache:            make(map[string]crypto.PublicKey),
		cacheLock:           &sync.RWMutex{},
	}

	rpcServer.RegisterCodec(jsonrpc.NewCodec(), "application/json")

	if err := ar.registerPublicServices(rpcServer); err != nil {
		return nil, throw.W(err, "[ NewAPIRunner ] Can't register public services:")
	}

	// init handler
	hc := NewHealthChecker(ar.CertificateManager, ar.NodeNetwork)

	router := http.NewServeMux()
	ar.server.Handler = router
	ar.SeedManager = seedmanager.New()

	server, err := NewRequestValidator(cfg.SwaggerPath, ar.rpcServer)
	if err != nil {
		return nil, throw.W(err, "failed to prepare api validator")
	}

	router.HandleFunc("/healthcheck", hc.CheckHandler)
	router.Handle(ar.cfg.RPC, server)
	ar.handler = router

	return &ar, nil
}

// IsAPIRunner is implementation of APIRunner interface for component manager
func (ar *Runner) IsAPIRunner() bool {
	return true
}

// Handler returns root http handler.
func (ar *Runner) Handler() http.Handler {
	return ar.handler
}

// Start runs api server
func (ar *Runner) Start(ctx context.Context) error {
	logger := inslogger.FromContext(ctx)
	logger.Info("Starting ApiRunner ...")
	logger.Info("Config: ", ar.cfg)
	listener, err := net.Listen("tcp", ar.server.Addr)
	if err != nil {
		return throw.W(err, "Can't start listening")
	}
	go func() {
		if err := ar.server.Serve(listener); err != http.ErrServerClosed {
			logger.Error("Http server: ListenAndServe() error: ", err)
		}
	}()
	return nil
}

// Stop stops api server
func (ar *Runner) Stop(ctx context.Context) error {
	const timeOut = 5

	inslogger.FromContext(ctx).Infof("Shutting down server gracefully ...(waiting for %d seconds)", timeOut)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeOut)*time.Second)
	defer cancel()
	err := ar.server.Shutdown(ctxWithTimeout)
	if err != nil {
		return throw.W(err, "Can't gracefully stop API server")
	}

	ar.SeedManager.Stop()

	return nil
}
