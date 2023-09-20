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

func TestCreateUpdateWallet(t *testing.T) {
	insrail.LogCase(t, "C4857")
	var (
		ref    string
		amount uint = 100
	)

	ctx := context.Background()

	t.Log("1.Create wallet")
	{
		walletRef, err := testutils.CreateSimpleWallet(ctx)
		require.NoError(t, err)

		ref = walletRef
		require.NotEmpty(t, ref, "wallet reference mustn't be empty")
	}
	t.Log("2.Get wallet balance")
	{
		getBalanceURL := testutils.GetURL(testutils.WalletGetBalancePath, "", "")
		balance, err := testutils.GetWalletBalance(ctx, getBalanceURL, ref)
		require.NoError(t, err)
		require.Equal(t, testutils.StartBalance, balance, "wrong balance amount")
	}
	t.Log("3.Add amount to wallet")
	{
		addAmountURL := testutils.GetURL(testutils.WalletAddAmountPath, "", "")
		err := testutils.AddAmountToWallet(ctx, addAmountURL, ref, amount)
		require.NoError(t, err)
	}
	t.Log("4.Get wallet balance")
	{
		getBalanceURL := testutils.GetURL(testutils.WalletGetBalancePath, "", "")
		balance, err := testutils.GetWalletBalance(ctx, getBalanceURL, ref)
		require.NoError(t, err)
		require.Equal(t, testutils.StartBalance+amount, balance, "wrong balance amount")
	}
}

func TestGetUpdateBalanceConcurrently(t *testing.T) {
	insrail.LogCase(t, "C4858")
	var (
		ref             string
		count                = 10 // Number of concurrent requests per node.
		amount          uint = 100
		expectedBalance      = testutils.StartBalance + amount*uint(count*len(testutils.GetPorts()))
		outChan              = make(chan error)
	)

	ctx := context.Background()

	t.Log("1.Create wallet")
	{
		walletRef, err := testutils.CreateSimpleWallet(ctx)
		require.NoError(t, err)

		ref = walletRef
		require.NotEmpty(t, ref, "wallet reference mustn't be empty")
	}
	t.Log("2.Concurrent requests to /add_amount and /get_balance")
	{
		for i := 0; i < count; i++ {
			for _, port := range testutils.GetPorts() {
				go func(port string) {
					getBalanceURL := testutils.GetURL(testutils.WalletGetBalancePath, "", port)
					_, resultErr := testutils.GetWalletBalance(ctx, getBalanceURL, ref)
					// testing.T isn't goroutine safe, so that we will check responses in main goroutine
					outChan <- resultErr
				}(port)

				go func(port string) {
					addAmountURL := testutils.GetURL(testutils.WalletAddAmountPath, "", port)
					resultErr := testutils.AddAmountToWallet(ctx, addAmountURL, ref, amount)
					// testing.T isn't goroutine safe, so that we will check responses in main goroutine
					outChan <- resultErr
				}(port)
			}
		}

		for i := 0; i < count*len(testutils.GetPorts())*2; i++ {
			assert.NoError(t, <-outChan)
		}
		close(outChan)
	}
	t.Log("3.Check balance after all requests are done")
	{
		getBalanceURL := testutils.GetURL(testutils.WalletGetBalancePath, "", "")
		balance, err := testutils.GetWalletBalance(ctx, getBalanceURL, ref)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, balance, "wrong balance amount")
	}
}
