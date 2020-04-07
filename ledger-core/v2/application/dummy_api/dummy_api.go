// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dummy_api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/pkg/errors"
)

type DummyAPI struct {
	server *http.Server
}

func mustConvertMapToJson(data map[string]interface{}) []byte {
	jsonString, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return jsonString
}

func create(w http.ResponseWriter, req *http.Request) {
	result := map[string]interface{}{"reference": gen.Reference()}
	rawJson := mustConvertMapToJson(result)
	_, err := w.Write(rawJson)
	if err != nil {
		panic(err)
	}
}

func NewDummyAPI(api configuration.DummyAPI) *DummyAPI {
	return &DummyAPI{
		server: &http.Server{Addr: api.Address},
	}
}

func registerHandlers() {
	walletLocation := "/wallet"
	http.HandleFunc(walletLocation+"/create", create)
	http.HandleFunc(walletLocation+"/transfer", create)
	http.HandleFunc(walletLocation+"/get_balance", create)
	http.HandleFunc(walletLocation+"/add_amount", create)
}

func (d *DummyAPI) Start(ctx context.Context) error {
	registerHandlers()

	listener, err := net.Listen("tcp", d.server.Addr)
	if err != nil {
		return errors.Wrap(err, "Can't start listening")
	}
	go func() {
		if err := d.server.Serve(listener); err != http.ErrServerClosed {
			inslogger.FromContext(ctx).Error("Http server: ListenAndServe() error: ", err)
		}
	}()

	return nil
}

func (d *DummyAPI) Stop(ctx context.Context) error {
	const timeOut = 5

	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeOut)*time.Second)
	defer cancel()
	err := d.server.Shutdown(ctxWithTimeout)
	if err != nil {
		return errors.Wrap(err, "Can't gracefully stop API server")
	}
	return nil
}
