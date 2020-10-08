// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"

	"github.com/stretchr/testify/require"
)

func TestGetStatus(t *testing.T) {
	status := getStatus(t)
	require.NotNil(t, status)

	numNodes, err := getNodesCount()
	require.NoError(t, err)

	require.Equal(t, "CompleteNetworkState", status.NetworkState)
	require.Equal(t, numNodes, status.WorkingListSize)
}

func getStatus(t testing.TB) statusResponse {
	body := getRPSResponseBody(t, launchnet.TestRPCUrl, postParams{
		"jsonrpc": "2.0",
		"method":  "node.getStatus",
		"id":      "1",
	})
	rpcStatusResponse := &rpcStatusResponse{}
	unmarshalRPCResponse(t, body, rpcStatusResponse)
	require.NotNil(t, rpcStatusResponse.Result)
	return rpcStatusResponse.Result
}
