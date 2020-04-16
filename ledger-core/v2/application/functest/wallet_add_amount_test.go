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

// Creates wallet, calls /wallet/add_amount and checks it's response body.
func TestWalletAddAmount(t *testing.T) {
	walletRef, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	addAmountURL := getURL(walletAddAmountPath, "", "")
	rawResp, err := sendAPIRequest(addAmountURL, walletAddAmountRequestBody{To: walletRef, Amount: 100})
	require.NoError(t, err, "failed to send request or get response body")

	resp, err := unmarshalWalletAddAmountResponse(rawResp)
	require.NoError(t, err, "failed to unmarshal response")

	require.Empty(t, resp.Err, "problem during execute request")
	require.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")
}

// Creates wallet and calls /wallet/add_amount concurrently.
func TestWalletAddAmountConcurrently(t *testing.T) {
	walletRef, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	var wg sync.WaitGroup
	count := 10 // Number of concurrent requests per node.
	responses := make([]error, 0, count*len(nodesPorts))

	for i := 0; i < count; i++ {
		for _, port := range nodesPorts {
			wg.Add(1)
			go func(wg *sync.WaitGroup, port string) {
				defer wg.Done()

				addAmountURL := getURL(walletAddAmountPath, "", port)
				err := addAmountToWallet(addAmountURL, walletRef, 100)
				// testing.T isn't goroutine save, so that we will check responses in main goroutine
				responses = append(responses, err)
			}(&wg, port)
		}
	}

	wg.Wait()

	for _, res := range responses {
		assert.NoError(t, res)
	}
}
