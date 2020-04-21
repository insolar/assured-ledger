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
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

type TestWalletServer struct {
	server   *http.Server
	feeder   conveyor.EventInputer
	accessor pulse.Accessor

	jsonCodec jsoniter.API
}

// Aliases VCallRequest.CallSiteMethod for call testwallet.Wallet contract methods
const (
	create     = "New"     // Create wallet
	getBalance = "Balance" // Get wallet balance
	addAmount  = "Accept"  // Add money to wallet
)

func NewTestWalletServer(api configuration.TestWalletAPI, feeder conveyor.EventInputer, accessor pulse.Accessor) *TestWalletServer {
	return &TestWalletServer{
		server:    &http.Server{Addr: api.Address},
		feeder:    feeder,
		accessor:  accessor,
		jsonCodec: jsoniter.ConfigCompatibleWithStandardLibrary,
	}
}

func registerHandlers(server *TestWalletServer) {
	walletLocation := "/wallet"
	http.HandleFunc(walletLocation+"/create", server.Create)
	http.HandleFunc(walletLocation+"/transfer", server.Transfer)
	http.HandleFunc(walletLocation+"/get_balance", server.GetBalance)
	http.HandleFunc(walletLocation+"/add_amount", server.AddAmount)
}

func (s *TestWalletServer) Start(ctx context.Context) error {
	registerHandlers(s)

	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return errors.Wrap(err, "Can't start listening")
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
		return errors.Wrap(err, "Can't gracefully stop API server")
	}
	return nil
}
