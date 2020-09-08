// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testwalletapi

import (
	"context"
	"net"
	"net/http"

	jsoniter "github.com/json-iterator/go"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

type TestWalletServer struct {
	server   *http.Server
	feeder   conveyor.EventInputer
	accessor beat.History
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

func NewTestWalletServer(api configuration.TestWalletAPI, feeder conveyor.EventInputer, accessor beat.History) *TestWalletServer {
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
	httpServerMux.HandleFunc(walletLocation+"/get_balance_validated", s.GetBalanceValidated)
	httpServerMux.HandleFunc(walletLocation+"/add_amount", s.AddAmount)
	httpServerMux.HandleFunc(walletLocation+"/delete", s.Delete)
}

func (s *TestWalletServer) Start(ctx context.Context) error {
	s.RegisterHandlers(s.mux)
	s.server.Handler = s.NodeReadyMiddleware(s.mux)

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
	return s.server.Close()
}

// NodeReadyMiddleware returns 503 ServiceUnavailable until node is ready
func (s *TestWalletServer) NodeReadyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.accessor.LatestTimeBeat(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, err := w.Write([]byte(`{"error":"node is not ready"}`))
			if err != nil {
				panic(err)
			}
		}

		next.ServeHTTP(w, r)
	})
}
