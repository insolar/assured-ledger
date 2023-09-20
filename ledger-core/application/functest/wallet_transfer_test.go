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

// Creates wallets, calls /wallet/transfer and checks it's response body, also checks balances after transfer.
func TestWalletTransfer(t *testing.T) {
	insrail.LogCase(t, "C4856")

	var transferAmount uint = 100

	ctx := context.Background()

	walletRefFrom, err := testutils.CreateSimpleWallet(ctx)
	require.NoError(t, err, "failed to create wallet")

	walletRefTo, err := testutils.CreateSimpleWallet(ctx)
	require.NoError(t, err, "failed to create wallet")

	transferURL := testutils.GetURL(testutils.WalletTransferPath, "", "")
	rawResp, err := testutils.SendAPIRequest(ctx, transferURL, testutils.WalletTransferRequestBody{From: walletRefFrom, To: walletRefTo, Amount: transferAmount})
	require.NoError(t, err, "failed to send request or get response body")

	resp, err := testutils.UnmarshalWalletTransferResponse(rawResp)
	require.NoError(t, err, "failed to unmarshal response")
	require.Empty(t, resp.Err, "problem during execute request")
	assert.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")

	getBalanceURL := testutils.GetURL(testutils.WalletGetBalancePath, "", "")

	walletFromBalance, err := testutils.GetWalletBalance(ctx, getBalanceURL, walletRefFrom)
	require.NoError(t, err, "failed to get balance")
	require.Equal(t, testutils.StartBalance-transferAmount, walletFromBalance, "wrong balance")

	walletToBalance, err := testutils.GetWalletBalance(ctx, getBalanceURL, walletRefTo)
	require.NoError(t, err, "failed to get balance")
	require.Equal(t, testutils.StartBalance+transferAmount, walletToBalance, "wrong balance")
}
