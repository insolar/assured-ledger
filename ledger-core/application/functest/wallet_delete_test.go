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

// Creates wallet, calls /wallet/delete, then calls /wallet/get_balance and gets error.
func TestWalletDelete(t *testing.T) {
	insrail.LogSkipCase(t, "C4859", "https://insolar.atlassian.net/browse/PLAT-414")

	ctx := context.Background()

	walletRef, err := testutils.CreateSimpleWallet(ctx)
	require.NoError(t, err, "failed to create wallet")

	deleteURL := testutils.GetURL(testutils.WalletDeletePath, "", "")
	rawResp, err := testutils.SendAPIRequest(ctx, deleteURL, testutils.WalletDeleteRequestBody{Ref: walletRef})
	require.NoError(t, err, "failed to send request or get response body")

	resp, err := testutils.UnmarshalWalletDeleteResponse(rawResp)
	require.NoError(t, err, "failed to unmarshal response")

	require.Empty(t, resp.Err, "problem during execute request")
	assert.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")

	getBalanceURL := testutils.GetURL(testutils.WalletGetBalancePath, "", "")
	_, getBalanceErr := testutils.GetWalletBalance(ctx, getBalanceURL, walletRef)
	require.Error(t, getBalanceErr)
	require.Contains(t, getBalanceErr, "todo")
}
