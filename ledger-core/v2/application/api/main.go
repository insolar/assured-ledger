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

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/api/seedmanager"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

// Runner implements Component for API
type Runner struct {
	CertificateManager node.CertificateManager
	// nolint
	NodeNetwork         network.NodeNetwork
	CertificateGetter   node.CertificateGetter
	PulseAccessor       pulsestor.Accessor
	JetCoordinator      jet.Coordinator
	NetworkStatus       node.NetworkStatus
	AvailabilityChecker AvailabilityChecker

	handler       http.Handler
	server        *http.Server
	rpcServer     *rpc.Server
	cfg           *configuration.APIRunner
	keyCache      map[string]crypto.PublicKey
	cacheLock     *sync.RWMutex
	SeedManager   *seedmanager.SeedManager
	SeedGenerator seedmanager.SeedGenerator
}

func checkConfig(cfg *configuration.APIRunner) error {
	if cfg == nil {
		return errors.New("[ checkConfig ] config is nil")
	}
	if cfg.Address == "" {
		return errors.New("[ checkConfig ] Address must not be empty")
	}
	if len(cfg.RPC) == 0 {
		return errors.New("[ checkConfig ] RPC must exist")
	}
	if len(cfg.SwaggerPath) == 0 {
		return errors.New("[ checkConfig ] Missing openAPI spec file path")
	}

	return nil
}

func (ar *Runner) registerPublicServices(rpcServer *rpc.Server) error {
	err := rpcServer.RegisterService(NewNodeService(ar), "node")
	if err != nil {
		return errors.Wrap(err, "[ registerServices ] Can't RegisterService: node")
	}

	return nil
}

// NewRunner is C-tor for API Runner
func NewRunner(cfg *configuration.APIRunner,
	certificateManager node.CertificateManager,
	// nolint
	nodeNetwork network.NodeNetwork,
	certificateGetter node.CertificateGetter,
	pulseAccessor pulsestor.Accessor,
	jetCoordinator jet.Coordinator,
	networkStatus node.NetworkStatus,
	availabilityChecker AvailabilityChecker,
) (*Runner, error) {

	err := checkConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "[ NewAPIRunner ] Bad config")
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
		rpcServer:           rpcServer,
		cfg:                 cfg,
		keyCache:            make(map[string]crypto.PublicKey),
		cacheLock:           &sync.RWMutex{},
	}

	rpcServer.RegisterCodec(jsonrpc.NewCodec(), "application/json")

	if err := ar.registerPublicServices(rpcServer); err != nil {
		return nil, errors.Wrap(err, "[ NewAPIRunner ] Can't register public services:")
	}

	// init handler
	hc := NewHealthChecker(ar.CertificateManager, ar.NodeNetwork, ar.PulseAccessor)

	router := http.NewServeMux()
	ar.server.Handler = router
	ar.SeedManager = seedmanager.New()

	server, err := NewRequestValidator(cfg.SwaggerPath, ar.rpcServer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare api validator")
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
		return errors.Wrap(err, "Can't start listening")
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
		return errors.Wrap(err, "Can't gracefully stop API server")
	}

	ar.SeedManager.Stop()

	return nil
}
