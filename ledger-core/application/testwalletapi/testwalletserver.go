// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testwalletapi

import (
	"context"
	"net"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

type TestWalletServer struct {
	server   *http.Server
	feeder   conveyor.EventInputer
	accessor pulsestor.Accessor
	mux      *http.ServeMux

	jsonCodec jsoniter.API
}

// Aliases VCallRequest.CallSiteMethod for call testwallet.Wallet contract methods
const (
	create     = "New"        // Create wallet
	getBalance = "GetBalance" // Get wallet balance
	addAmount  = "Accept"     // Add money to wallet
	transfer   = "Transfer"   // Transfer money between wallets
)

func NewTestWalletServer(api configuration.TestWalletAPI, feeder conveyor.EventInputer, accessor pulsestor.Accessor) *TestWalletServer {
	return &TestWalletServer{
		server:    &http.Server{Addr: api.Address},
		mux:       http.NewServeMux(),
		feeder:    feeder,
		accessor:  accessor,
		jsonCodec: jsoniter.ConfigCompatibleWithStandardLibrary,
	}
}

func (s *TestWalletServer) RegisterHandlers(httpServerMux *http.ServeMux) {
	walletLocation := "/wallet"
	httpServerMux.HandleFunc(walletLocation+"/create", s.Create)
	httpServerMux.HandleFunc(walletLocation+"/transfer", s.Transfer)
	httpServerMux.HandleFunc(walletLocation+"/get_balance", s.GetBalance)
	httpServerMux.HandleFunc(walletLocation+"/add_amount", s.AddAmount)
	httpServerMux.HandleFunc(walletLocation+"/delete", s.Delete)
}

func (s *TestWalletServer) Start(ctx context.Context) error {
	s.RegisterHandlers(s.mux)
	s.server.Handler = s.mux

	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return errors.W(err, "Can't start listening")
	}

	go func() {
		if err := s.server.Serve(listener); err != http.ErrServerClosed {
			inslogger.FromContext(ctx).Error("Http server: ListenAndServe() error: ", err)
		}
	}()

	return nil
}

func (s *TestWalletServer) Stop(ctx context.Context) error {
	const timeOut = 5

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut*time.Second)
	defer cancel()
	err := s.server.Shutdown(ctxWithTimeout)
	if err != nil {
		return errors.W(err, "Can't gracefully stop API server")
	}
	return nil
}
