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

// Creates wallet, calls /wallet/get_balance and checks it's response body.
func TestWalletGetBalance(t *testing.T) {
	insrail.LogCase(t, "C4855")

	walletRef, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	getBalanceURL := getURL(walletGetBalancePath, "", "")
	rawResp, err := sendAPIRequest(getBalanceURL, walletGetBalanceRequestBody{Ref: walletRef})
	require.NoError(t, err, "failed to send request or get response body")

	resp, err := unmarshalWalletGetBalanceResponse(rawResp)
	require.NoError(t, err, "failed to unmarshal response")

	require.Empty(t, resp.Err, "problem during execute request")
	assert.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")
	assert.Equal(t, startBalance, resp.Amount, "wrong amount")
}

// Creates wallet and calls /wallet/get_balance concurrently.
func TestWalletGetBalanceConcurrently(t *testing.T) {
	insrail.LogCase(t, "C4920")

	walletRef, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	count := 10 // Number of concurrent requests per node.

	type result struct {
		balance uint
		err     error
	}
	outChan := make(chan result)

	for i := 0; i < count; i++ {
		for _, port := range defaultPorts {
			go func(port, walletRef string) {
				getBalanceURL := getURL(walletGetBalancePath, "", port)
				balance, err := getWalletBalance(getBalanceURL, walletRef)
				// testing.T isn't goroutine safe, so that we will check responses in main goroutine
				outChan <- result{balance: balance, err: err}
			}(port, walletRef)
		}
	}

	for i := 0; i < count*len(defaultPorts); i++ {
		res := <-outChan
		assert.NoError(t, res.err)
		assert.Equal(t, startBalance, res.balance, "wrong balance amount")
	}
	close(outChan)
}
