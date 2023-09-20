// +build functest

package functest

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/application/api/requester"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/stretchr/testify/require"
)

type contractInfo struct {
	reference reference.Global
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

const (
	apiTimeout = 10 * time.Second
)

func getRPSResponseBody(t testing.TB, URL string, postParams map[string]interface{}) []byte {
	jsonValue, _ := json.Marshal(postParams)

	request, err := http.NewRequest(http.MethodPost, URL, bytes.NewBuffer(jsonValue))
	require.NoError(t, err)
	request.Header["Content-type"] = []string{"application/json"}
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	request = request.WithContext(ctx)

	postResp, err := http.DefaultClient.Do(request)
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

func getTestNodesSetup() (numVirtual uint, numLight uint, numHeavy uint) {
	// default num nodes
	{
		numVirtual, numLight, numHeavy = 5, 0, 0
	}

	if numVirtualStr := os.Getenv("NUM_DISCOVERY_VIRTUAL_NODES"); numVirtualStr != "" {
		num, err := strconv.Atoi(numVirtualStr)
		if err != nil {
			panic(err)
		}
		numVirtual = uint(num)
	}
	if numLightStr := os.Getenv("NUM_DISCOVERY_LIGHT_NODES"); numLightStr != "" {
		num, err := strconv.Atoi(numLightStr)
		if err != nil {
			panic(err)
		}
		numLight = uint(num)
	}
	if numHeavyStr := os.Getenv("NUM_DISCOVERY_HEAVY_NODES"); numHeavyStr != "" {
		num, err := strconv.Atoi(numHeavyStr)
		if err != nil {
			panic(err)
		}
		numHeavy = uint(num)
	}
	return numVirtual, numLight, numHeavy
}
