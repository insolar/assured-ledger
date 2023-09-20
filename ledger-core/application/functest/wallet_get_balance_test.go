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

// Creates wallet, calls /wallet/get_balance and checks it's response body.
func TestWalletGetBalance(t *testing.T) {
	insrail.LogCase(t, "C4855")

	ctx := context.Background()

	walletRef, err := testutils.CreateSimpleWallet(ctx)
	require.NoError(t, err, "failed to create wallet")

	getBalanceURL := testutils.GetURL(testutils.WalletGetBalancePath, "", "")
	rawResp, err := testutils.SendAPIRequest(ctx, getBalanceURL, testutils.WalletGetBalanceRequestBody{Ref: walletRef})
	require.NoError(t, err, "failed to send request or get response body")

	resp, err := testutils.UnmarshalWalletGetBalanceResponse(rawResp)
	require.NoError(t, err, "failed to unmarshal response")

	require.Empty(t, resp.Err, "problem during execute request")
	assert.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")
	assert.Equal(t, testutils.StartBalance, resp.Amount, "wrong amount")
}

// Creates wallet and calls /wallet/get_balance concurrently.
func TestWalletGetBalanceConcurrently(t *testing.T) {
	insrail.LogCase(t, "C4920")

	ctx := context.Background()

	walletRef, err := testutils.CreateSimpleWallet(ctx)
	require.NoError(t, err, "failed to create wallet")

	count := 10 // Number of concurrent requests per node.

	type result struct {
		balance uint
		err     error
	}
	outChan := make(chan result)

	for i := 0; i < count; i++ {
		for _, port := range testutils.GetPorts() {
			go func(port string) {
				getBalanceURL := testutils.GetURL(testutils.WalletGetBalancePath, "", port)
				balance, err := testutils.GetWalletBalance(ctx, getBalanceURL, walletRef)
				// testing.T isn't goroutine safe, so that we will check responses in main goroutine
				outChan <- result{balance: balance, err: err}
			}(port)
		}
	}

	for i := 0; i < count*len(testutils.GetPorts()); i++ {
		res := <-outChan
		assert.NoError(t, res.err)
		assert.Equal(t, testutils.StartBalance, res.balance, "wrong balance amount")
	}
	close(outChan)
}
