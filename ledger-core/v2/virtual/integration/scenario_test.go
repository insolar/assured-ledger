// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

// 10 parallel executions
func TestVirtual_Scenario1(t *testing.T) {
	t.Parallel()

	t.Log("C4932")

	server, ctx := utils.NewServer(t, nil)
	defer server.Stop()

	var (
		count                = 10 // Number of concurrent requests
		amount          uint = 100
		expectedBalance      = 1000000000 + uint(count)*amount
		outChan              = make(chan error, count)
	)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		// verify and decode incoming message
		assert.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])

		return nil
	}

	var (
		walletReference reference.Global
	)
	{
		code, byteBuffer := server.CallAPICreateWallet(ctx)
		require.Equal(t, 200, code, string(byteBuffer))

		walletResponse, err := utils.UnmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, walletResponse.Err)
		assert.NotEmpty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)

		walletReference, err = reference.DefaultDecoder().Decode(walletResponse.Ref)
		require.NoError(t, err)
	}

	for i := 0; i < count; i++ {
		go func() {
			code, byteBuffer := server.CallAPIAddAmount(ctx, walletReference, amount)
			if code != 200 {
				outChan <- throw.Errorf("bad code, expected 200 got %d", code)
				return
			}
			// testing.T isn't goroutine safe, so that we will check responses in main goroutine
			response, err := utils.UnmarshalWalletTransferResponse(byteBuffer)
			if err != nil {
				outChan <- throw.W(err, "failed to parse response")
				return
			}

			if response.Err != "" {
				err := throw.New(response.Err)
				outChan <- throw.W(err, "failed to execute contract")
				return
			}

			outChan <- nil
		}()
	}

	for i := 0; i < count; i++ {
		assert.NoError(t, <-outChan)
	}
	close(outChan)

	code, byteBuffer := server.CallAPIGetBalance(ctx, walletReference)
	require.Equal(t, 200, code, string(byteBuffer))

	response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
	require.NoError(t, err)
	assert.Empty(t, response.Err)
	assert.NotEmpty(t, response.TraceID)
	assert.Equal(t, expectedBalance, response.Amount)
}

// 10 sequential executions
func TestVirtual_Scenario2(t *testing.T) {
	t.Parallel()

	t.Log("C4933")

	server, ctx := utils.NewServer(t, nil)
	defer server.Stop()

	var (
		count                = 10 // Number of concurrent requests
		amount          uint = 100
		expectedBalance      = 1000000000 + uint(count)*amount
	)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		// verify and decode incoming message
		assert.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])

		return nil
	}

	var (
		walletReference reference.Global
	)
	{
		code, byteBuffer := server.CallAPICreateWallet(ctx)
		require.Equal(t, 200, code, string(byteBuffer))

		walletResponse, err := utils.UnmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, walletResponse.Err)
		assert.NotEmpty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)

		walletReference, err = reference.DefaultDecoder().Decode(walletResponse.Ref)
		require.NoError(t, err)
	}

	for i := 0; i < count; i++ {
		code, byteBuffer := server.CallAPIAddAmount(ctx, walletReference, amount)
		assert.Equal(t, 200, code)

		// testing.T isn't goroutine safe, so that we will check responses in main goroutine
		response, err := utils.UnmarshalWalletTransferResponse(byteBuffer)
		assert.NoError(t, err)
		assert.Empty(t, response.Err)
	}

	code, byteBuffer := server.CallAPIGetBalance(ctx, walletReference)
	require.Equal(t, 200, code, string(byteBuffer))

	response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
	require.NoError(t, err)
	assert.Empty(t, response.Err)
	assert.NotEmpty(t, response.TraceID)
	assert.Equal(t, expectedBalance, response.Amount)
}
