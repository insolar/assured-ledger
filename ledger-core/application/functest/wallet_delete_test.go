// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
)

// Creates wallet, calls /wallet/delete, then calls /wallet/get_balance and gets error.
func TestWalletDelete(t *testing.T) {
	insrail.LogSkipCase(t, "C4859", "https://insolar.atlassian.net/browse/PLAT-414")

	walletRef, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	deleteURL := getURL(walletDeletePath, "", "")
	rawResp, err := sendAPIRequest(deleteURL, walletDeleteRequestBody{Ref: walletRef})
	require.NoError(t, err, "failed to send request or get response body")

	resp, err := unmarshalWalletDeleteResponse(rawResp)
	require.NoError(t, err, "failed to unmarshal response")

	require.Empty(t, resp.Err, "problem during execute request")
	assert.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")

	getBalanceURL := getURL(walletGetBalancePath, "", "")
	_, getBalanceErr := getWalletBalance(getBalanceURL, walletRef)
	require.Error(t, getBalanceErr)
	require.Contains(t, getBalanceErr, "todo")
}
