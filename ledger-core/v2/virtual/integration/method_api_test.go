// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func TestVirtual_Method_API(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		// verify and decode incoming message
		assert.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])

		return nil
	}

	var (
		walletReference1 reference.Global
		walletReference2 reference.Global
	)
	{
		code, byteBuffer := server.CallAPICreateWallet()
		require.Equal(t, 200, code, string(byteBuffer))

		walletResponse, err := utils.UnmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, walletResponse.Err)
		assert.NotEmpty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)

		walletReference1, err = reference.DefaultDecoder().Decode(walletResponse.Ref)
		require.NoError(t, err)
	}
	{
		code, byteBuffer := server.CallAPICreateWallet()
		require.Equal(t, 200, code, string(byteBuffer))

		walletResponse, err := utils.UnmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, walletResponse.Err)
		assert.NotEmpty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)

		walletReference2, err = reference.DefaultDecoder().Decode(walletResponse.Ref)
		require.NoError(t, err)
	}

	t.Run("AddAmount", func(t *testing.T) {
		code, byteBuffer := server.CallAPIAddAmount(walletReference1, 500)
		require.Equal(t, 200, code, string(byteBuffer))

		response, err := utils.UnmarshalWalletAddAmountResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, response.Err)
		assert.NotEmpty(t, response.TraceID)
	})

	t.Run("GetBalance", func(t *testing.T) {
		code, byteBuffer := server.CallAPIGetBalance(walletReference1)
		require.Equal(t, 200, code, string(byteBuffer))

		response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, response.Err)
		assert.NotEmpty(t, response.TraceID)
		assert.Equal(t, uint(1000000500), response.Amount)
	})

	t.Run("Transfer", func(t *testing.T) {
		t.Skip("Not implemented yet")

		code, byteBuffer := server.CallAPITransfer(walletReference1, walletReference2, 500)
		require.Equal(t, 200, code, string(byteBuffer))

		response, err := utils.UnmarshalWalletTransferResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, response.Err)
		assert.NotEmpty(t, response.TraceID)
	})
}

// 10 parallel executions
func TestVirtual_Scenario1(t *testing.T) {
	t.Skip("hangs?")

	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

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
		code, byteBuffer := server.CallAPICreateWallet()
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
			code, byteBuffer := server.CallAPIAddAmount(walletReference, amount)
			if code != 200 {
				outChan <- errors.Errorf("bad code, expected 200 got %d", code)
				return
			}
			// testing.T isn't goroutine safe, so that we will check responses in main goroutine
			response, err := utils.UnmarshalWalletTransferResponse(byteBuffer)
			if err != nil {
				outChan <- errors.Wrap(err, "failed to parse response")
				return
			}

			if response.Err != "" {
				err := errors.New(response.Err)
				outChan <- errors.Wrap(err, "failed to execute contract")
				return
			}

			outChan <- nil
		}()
	}

	for i := 0; i < count; i++ {
		assert.NoError(t, <-outChan)
	}
	close(outChan)

	code, byteBuffer := server.CallAPIGetBalance(walletReference)
	require.Equal(t, 200, code, string(byteBuffer))

	response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
	require.NoError(t, err)
	assert.Empty(t, response.Err)
	assert.NotEmpty(t, response.TraceID)
	assert.Equal(t, expectedBalance, response.Amount)
}

// 10 sequential executions
func TestVirtual_Scenario2(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

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
		code, byteBuffer := server.CallAPICreateWallet()
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
		code, byteBuffer := server.CallAPIAddAmount(walletReference, amount)
		assert.Equal(t, 200, code)

		// testing.T isn't goroutine safe, so that we will check responses in main goroutine
		response, err := utils.UnmarshalWalletTransferResponse(byteBuffer)
		assert.NoError(t, err)
		assert.Empty(t, response.Err)
	}

	code, byteBuffer := server.CallAPIGetBalance(walletReference)
	require.Equal(t, 200, code, string(byteBuffer))

	response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
	require.NoError(t, err)
	assert.Empty(t, response.Err)
	assert.NotEmpty(t, response.TraceID)
	assert.Equal(t, expectedBalance, response.Amount)
}
