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

// Creates wallet, calls /wallet/add_amount and checks it's response body.
func TestWalletAddAmount(t *testing.T) {
	insrail.LogCase(t, "C4921")

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
	insrail.LogCase(t, "C4922")

	walletRef, err := createSimpleWallet()
	require.NoError(t, err, "failed to create wallet")

	count := 10 // Number of concurrent requests per node.
	outChan := make(chan error)

	for i := 0; i < count; i++ {
		for _, port := range defaultPorts {
			go func(port, walletRef string) {
				addAmountURL := getURL(walletAddAmountPath, "", port)
				resultErr := addAmountToWallet(addAmountURL, walletRef, 100)
				// testing.T isn't goroutine safe, so that we will check responses in main goroutine
				outChan <- resultErr
			}(port, walletRef)
		}
	}

	for i := 0; i < count*len(defaultPorts); i++ {
		assert.NoError(t, <-outChan)
	}
	close(outChan)
}
