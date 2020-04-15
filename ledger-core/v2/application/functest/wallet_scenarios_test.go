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

func TestCreateUpdateWallet(t *testing.T) {
	var ref string

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
		require.Equal(t, 1000, balance, "wrong balance amount")
	}
	t.Log("3.Add amount to wallet")
	{
		addAmountURL := getURL(walletAddAmountPath, "", "")
		err := addAmountToWallet(addAmountURL, ref, 100)
		require.NoError(t, err)
	}
	t.Log("4.Get wallet balance")
	{
		getBalanceURL := getURL(walletGetBalancePath, "", "")
		balance, err := getWalletBalance(getBalanceURL, ref)
		require.NoError(t, err)
		require.Equal(t, 1100, balance, "wrong balance amount")
	}
}

func TestGetUpdateBalanceConcurrently(t *testing.T) {
	var (
		ref             string
		wg              sync.WaitGroup
		count           = 10 // Number of concurrent requests per node.
		amount          = 100
		expectedBalance = 1000 + amount*count*len(nodesPorts)
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
			for _, port := range nodesPorts {
				wg.Add(1)
				go func(wg *sync.WaitGroup, port string) {
					defer wg.Done()

					getBalanceURL := getURL(walletGetBalancePath, "", port)
					_, err := getWalletBalance(getBalanceURL, ref)
					assert.NoError(t, err)
				}(&wg, port)

				wg.Add(1)
				go func(wg *sync.WaitGroup, port string) {
					defer wg.Done()

					addAmountURL := getURL(walletAddAmountPath, "", port)
					err := addAmountToWallet(addAmountURL, ref, uint(amount))
					assert.NoError(t, err)
				}(&wg, port)
			}
		}
	}
	t.Log("3.Check balance after all requests are done")
	{
		wg.Wait()

		getBalanceURL := getURL(walletGetBalancePath, "", "")
		balance, err := getWalletBalance(getBalanceURL, ref)
		require.NoError(t, err)
		require.Equal(t, expectedBalance, balance, "wrong balance amount")
	}
}
