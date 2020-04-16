// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Creates wallet, calls /wallet/get_balance and checks it's response body.
func TestWalletGetBalance(t *testing.T) {
	walletRef, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	getBalanceURL := getURL(walletGetBalancePath, "", "")
	rawResp, err := sendAPIRequest(getBalanceURL, walletGetBalanceRequestBody{Ref: walletRef})
	require.NoError(t, err, "failed to send request or get response body")

	resp, err := unmarshalWalletGetBalanceResponse(rawResp)
	require.NoError(t, err, "failed to unmarshal response")

	require.Empty(t, resp.Err, "problem during execute request")
	assert.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")
	assert.Equal(t, 1000, resp.Amount, "wrong amount")
}

// Creates wallet and calls /wallet/get_balance concurrently.
func TestWalletGetBalanceConcurrently(t *testing.T) {
	walletRef, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	var wg sync.WaitGroup
	count := 10 // Number of concurrent requests per node.

	type result struct {
		balance int
		err     error
	}
	responses := make([]result, 0, count*len(nodesPorts))

	for i := 0; i < count; i++ {
		for _, port := range nodesPorts {
			wg.Add(1)
			go func(wg *sync.WaitGroup, port string) {
				defer wg.Done()

				getBalanceURL := getURL(walletGetBalancePath, "", port)
				balance, err := getWalletBalance(getBalanceURL, walletRef)
				// testing.T isn't goroutine save, so that we will check responses in main goroutine
				responses = append(responses, result{balance: balance, err: err})
			}(&wg, port)
		}
	}

	wg.Wait()

	for _, res := range responses {
		assert.NoError(t, res.err)
		assert.Equal(t, 1000, res.balance, "wrong balance amount")
	}
}
