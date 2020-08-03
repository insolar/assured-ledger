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

// Creates wallets, calls /wallet/transfer and checks it's response body, also checks balances after transfer.
func TestWalletTransfer(t *testing.T) {
	insrail.LogCase(t, "C4856")

	var transferAmount uint = 100

	walletRefFrom, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	walletRefTo, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	transferURL := getURL(walletTransferPath, "", "")
	rawResp, err := sendAPIRequest(transferURL, walletTransferRequestBody{From: walletRefFrom, To: walletRefTo, Amount: transferAmount})
	require.NoError(t, err, "failed to send request or get response body")

	resp, err := unmarshalWalletTransferResponse(rawResp)
	require.NoError(t, err, "failed to unmarshal response")
	require.Empty(t, resp.Err, "problem during execute request")
	assert.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")

	getBalanceURL := getURL(walletGetBalancePath, "", "")

	walletFromBalance, err := getWalletBalance(getBalanceURL, walletRefFrom)
	require.NoError(t, err, "failed to get balance")
	require.Equal(t, startBalance-transferAmount, walletFromBalance, "wrong balance")

	walletToBalance, err := getWalletBalance(getBalanceURL, walletRefTo)
	require.NoError(t, err, "failed to get balance")
	require.Equal(t, startBalance+transferAmount, walletToBalance, "wrong balance")
}
