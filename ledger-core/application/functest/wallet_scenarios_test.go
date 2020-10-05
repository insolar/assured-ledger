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

func TestCreateUpdateWallet(t *testing.T) {
	insrail.LogCase(t, "C4857")
	var (
		ref    string
		amount uint = 100
	)

	t.Log("1.Create wallet")
	{
		walletRef, err := createSimpleWallet()
		require.NoError(t, err)

		ref = walletRef
		require.NotEmpty(t, ref, "wallet reference mustn't be empty")
	}
	t.Log("2.Get wallet balance")
	{
		getBalanceURL := getURL(walletGetBalancePath, "", "")
		balance, err := getWalletBalance(getBalanceURL, ref)
		require.NoError(t, err)
		require.Equal(t, startBalance, balance, "wrong balance amount")
	}
	t.Log("3.Add amount to wallet")
	{
		addAmountURL := getURL(walletAddAmountPath, "", "")
		err := addAmountToWallet(addAmountURL, ref, amount)
		require.NoError(t, err)
	}
	t.Log("4.Get wallet balance")
	{
		getBalanceURL := getURL(walletGetBalancePath, "", "")
		balance, err := getWalletBalance(getBalanceURL, ref)
		require.NoError(t, err)
		require.Equal(t, startBalance+amount, balance, "wrong balance amount")
	}
}

func TestGetUpdateBalanceConcurrently(t *testing.T) {
	insrail.LogCase(t, "C4858")
	var (
		ref             string
		count                = 10 // Number of concurrent requests per node.
		amount          uint = 100
		expectedBalance      = startBalance + amount*uint(count*len(defaultPorts))
		outChan              = make(chan error)
	)

	t.Log("1.Create wallet")
	{
		walletRef, err := createSimpleWallet()
		require.NoError(t, err)

		ref = walletRef
		require.NotEmpty(t, ref, "wallet reference mustn't be empty")
	}
	t.Log("2.Concurrent requests to /add_amount and /get_balance")
	{
		for i := 0; i < count; i++ {
			for _, port := range defaultPorts {
				go func(port, ref string) {
					getBalanceURL := getURL(walletGetBalancePath, "", port)
					_, resultErr := getWalletBalance(getBalanceURL, ref)
					// testing.T isn't goroutine safe, so that we will check responses in main goroutine
					outChan <- resultErr
				}(port, ref)

				go func(port, ref string, amount uint) {
					addAmountURL := getURL(walletAddAmountPath, "", port)
					resultErr := addAmountToWallet(addAmountURL, ref, amount)
					// testing.T isn't goroutine safe, so that we will check responses in main goroutine
					outChan <- resultErr
				}(port, ref, amount)
			}
		}

		for i := 0; i < count*len(defaultPorts)*2; i++ {
			assert.NoError(t, <-outChan)
		}
		close(outChan)
	}
	t.Log("3.Check balance after all requests are done")
	{
		getBalanceURL := getURL(walletGetBalancePath, "", "")
		balance, err := getWalletBalance(getBalanceURL, ref)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, balance, "wrong balance amount")
	}
}
