// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testwalletapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

func mustConvertMapToJSON(data map[string]interface{}) []byte {
	jsonString, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		panic(err)
	}

	return jsonString
}

const (
	traceIDField = "traceID"
	errorField   = "error"
)

func create(w http.ResponseWriter, req *http.Request) {
	traceID := utils.RandTraceID()
	_, inslog := inslogger.WithTraceField(context.Background(), traceID)
	inslog.Info("Incoming request: ", req.URL)

	result := map[string]interface{}{
		"reference":  gen.Reference().String(),
		traceIDField: traceID,
		errorField:   nil,
	}
	rawJSON := mustConvertMapToJSON(result)
	_, err := w.Write(rawJSON)
	if err != nil {
		panic(err)
	}
}

type TransferParams struct {
	From   string
	To     string
	Amount uint
}

func (t *TransferParams) isValid() bool {
	return len(t.To) > 0 && len(t.From) > 0 && t.Amount > 0
}

const badRequestErrorPattern = "%s. " + traceIDField + ": %s"

func transfer(w http.ResponseWriter, req *http.Request) {
	traceID := utils.RandTraceID()
	_, inslog := inslogger.WithTraceField(context.Background(), traceID)
	inslog.Info("Incoming request: ", req.URL)

	params := TransferParams{}
	err := json.NewDecoder(req.Body).Decode(&params)
	if err != nil {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "Can't parse boby: "+err.Error(), traceID), http.StatusBadRequest)
		return
	}
	if !params.isValid() {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "invalid input params", traceID), http.StatusBadRequest)
		return
	}

	result := map[string]interface{}{
		traceIDField: traceID,
		errorField:   nil,
	}

	rawJSON := mustConvertMapToJSON(result)
	_, err = w.Write(rawJSON)
	if err != nil {
		panic(err)
	}
}

type GetBalanceParams struct {
	WalletRef string
}

func (g *GetBalanceParams) isValid() bool {
	return len(g.WalletRef) > 0
}

func getBalance(w http.ResponseWriter, req *http.Request) {
	traceID := utils.RandTraceID()
	_, inslog := inslogger.WithTraceField(context.Background(), traceID)
	inslog.Info("Incoming request: ", req.URL)

	params := GetBalanceParams{}
	err := json.NewDecoder(req.Body).Decode(&params)
	if err != nil {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "Can't parse boby: "+err.Error(), traceID), http.StatusBadRequest)
		return
	}
	if !params.isValid() {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "invalid input params", traceID), http.StatusBadRequest)
		return
	}

	result := map[string]interface{}{
		"amount":     1000,
		traceIDField: traceID,
		errorField:   nil,
	}

	rawJSON := mustConvertMapToJSON(result)
	_, err = w.Write(rawJSON)
	if err != nil {
		panic(err)
	}
}

type AddAmountParams struct {
	To     string
	Amount uint
}

func (a *AddAmountParams) isValid() bool {
	return a.Amount > 0 && len(a.To) > 0
}

func addAmount(w http.ResponseWriter, req *http.Request) {
	traceID := utils.RandTraceID()
	_, inslog := inslogger.WithTraceField(context.Background(), traceID)
	inslog.Info("Incoming request: ", req.URL)

	params := AddAmountParams{}
	err := json.NewDecoder(req.Body).Decode(&params)
	if err != nil {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "Can't parse boby: "+err.Error(), traceID), http.StatusBadRequest)
		return
	}
	if !params.isValid() {
		http.Error(w, fmt.Sprintf(badRequestErrorPattern, "invalid input params", traceID), http.StatusBadRequest)
		return
	}

	result := map[string]interface{}{
		traceIDField: traceID,
		errorField:   nil,
	}

	rawJSON := mustConvertMapToJSON(result)
	_, err = w.Write(rawJSON)
	if err != nil {
		panic(err)
	}
}
