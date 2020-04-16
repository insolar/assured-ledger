// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"

	"github.com/stretchr/testify/require"
)

type contractInfo struct {
	reference *insolar.Reference
	testName  string
}

var contracts = map[string]*contractInfo{}

type postParams map[string]interface{}

type RPCResponseInterface interface {
	getRPCVersion() string
	getError() map[string]interface{}
}

type RPCResponse struct {
	RPCVersion string                 `json:"jsonrpc"`
	Error      map[string]interface{} `json:"error"`
}

func (r *RPCResponse) getRPCVersion() string {
	return r.RPCVersion
}

func (r *RPCResponse) getError() map[string]interface{} {
	return r.Error
}

type getSeedResponse struct {
	RPCResponse
	Result struct {
		Seed    string `json:"seed"`
		TraceID string `json:"traceID"`
	} `json:"result"`
}

type statusResponse struct {
	NetworkState    string `json:"networkState"`
	WorkingListSize int    `json:"workingListSize"`
}

type rpcStatusResponse struct {
	RPCResponse
	Result statusResponse `json:"result"`
}

func getRPSResponseBody(t testing.TB, URL string, postParams map[string]interface{}) []byte {
	jsonValue, _ := json.Marshal(postParams)

	postResp, err := http.Post(URL, "application/json", bytes.NewBuffer(jsonValue))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, postResp.StatusCode)
	body, err := ioutil.ReadAll(postResp.Body)
	require.NoError(t, err)
	return body
}

func unmarshalRPCResponse(t testing.TB, body []byte, response RPCResponseInterface) {
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)
	require.Equal(t, "2.0", response.getRPCVersion())
	require.Nil(t, response.getError())
}

func unmarshalCallResponse(t testing.TB, body []byte, response *requester.ContractResponse) {
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)
}
