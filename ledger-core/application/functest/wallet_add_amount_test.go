// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
)

// Creates wallet, calls /wallet/add_amount and checks it's response body.
func TestWalletAddAmount(t *testing.T) {
	insrail.LogCase(t, "C4921")

	ctx := context.Background()

	walletRef, err := testutils.CreateSimpleWallet(ctx)
	require.NoError(t, err, "failed to create wallet")

	addAmountURL := testutils.GetURL(testutils.WalletAddAmountPath, "", "")
	require.NoError(t, testutils.AddAmountToWallet(ctx, addAmountURL, walletRef, 100))
}

// Creates wallet and calls /wallet/add_amount concurrently.
func TestWalletAddAmountConcurrently(t *testing.T) {
	insrail.LogCase(t, "C4922")

	ctx := context.Background()

	walletRef, err := testutils.CreateSimpleWallet(ctx)
	require.NoError(t, err, "failed to create wallet")

	count := 10 // Number of concurrent requests per node.
	outChan := make(chan error)

	for i := 0; i < count; i++ {
		for _, port := range testutils.GetPorts() {
			go func(port, walletRef string) {
				addAmountURL := testutils.GetURL(testutils.WalletAddAmountPath, "", "")
				resultErr := testutils.AddAmountToWallet(ctx, addAmountURL, walletRef, 100)
				// testing.T isn't goroutine safe, so that we will check responses in main goroutine
				outChan <- resultErr
			}(port, walletRef)
		}
	}

	for i := 0; i < count*len(testutils.GetPorts()); i++ {
		assert.NoError(t, <-outChan)
	}
	close(outChan)
}
