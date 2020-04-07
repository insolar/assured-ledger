// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package testwalletapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

func mustConvertMapToJson(data map[string]interface{}) []byte {
	jsonString, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		panic(err)
	}

	return jsonString
}

const traceIDField = "traceID"

func create(w http.ResponseWriter, req *http.Request) {
	traceID := utils.RandTraceID()
	_, inslog := inslogger.WithTraceField(context.Background(), traceID)
	inslog.Info("Incoming request: ", req.URL)

	result := map[string]interface{}{
		"reference":  gen.Reference().String(),
		traceIDField: traceID,
	}
	rawJson := mustConvertMapToJson(result)
	_, err := w.Write(rawJson)
	if err != nil {
		panic(err)
	}
}

func transfer(w http.ResponseWriter, req *http.Request) {
	traceID := utils.RandTraceID()
	_, inslog := inslogger.WithTraceField(context.Background(), traceID)
	inslog.Info("Incoming request: ", req.URL)

	result := map[string]interface{}{
		traceIDField: traceID,
	}

	rawJson := mustConvertMapToJson(result)
	_, err := w.Write(rawJson)
	if err != nil {
		panic(err)
	}
}

func getBalance(w http.ResponseWriter, req *http.Request) {
	traceID := utils.RandTraceID()
	_, inslog := inslogger.WithTraceField(context.Background(), traceID)
	inslog.Info("Incoming request: ", req.URL)

	result := map[string]interface{}{
		traceIDField: traceID,
		"amount":     1000,
	}

	rawJson := mustConvertMapToJson(result)
	_, err := w.Write(rawJson)
	if err != nil {
		panic(err)
	}
}

func addAmount(w http.ResponseWriter, req *http.Request) {
	traceID := utils.RandTraceID()
	_, inslog := inslogger.WithTraceField(context.Background(), traceID)
	inslog.Info("Incoming request: ", req.URL)

	result := map[string]interface{}{
		traceIDField: traceID,
	}

	rawJson := mustConvertMapToJson(result)
	_, err := w.Write(rawJson)
	if err != nil {
		panic(err)
	}
}
