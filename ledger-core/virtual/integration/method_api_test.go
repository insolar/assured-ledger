// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/investigation"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_Method_API(t *testing.T) {
	defer commontestutils.LeakTester(t)
	investigation.LogCase(t, "C4931")

	server, ctx := utils.NewServerWithErrorFilter(nil, t, func(s string) bool {
		return false
	})
	defer server.Stop()

	var (
		walletReference1 reference.Global
		walletReference2 reference.Global
	)
	{
		code, byteBuffer := server.CallAPICreateWallet(ctx)
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
		code, byteBuffer := server.CallAPICreateWallet(ctx)
		require.Equal(t, 200, code, string(byteBuffer))

		walletResponse, err := utils.UnmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)

		assert.Empty(t, walletResponse.Err)
		assert.NotEmpty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)

		walletReference2, err = reference.DefaultDecoder().Decode(walletResponse.Ref)
		require.NoError(t, err)
	}

	{
		code, byteBuffer := server.CallAPIAddAmount(ctx, walletReference1, 500)
		require.Equal(t, 200, code, string(byteBuffer))

		response, err := utils.UnmarshalWalletAddAmountResponse(byteBuffer)
		require.NoError(t, err)

		assert.Empty(t, response.Err)
		assert.NotEmpty(t, response.TraceID)
	}

	{
		code, byteBuffer := server.CallAPIGetBalance(ctx, walletReference1)
		require.Equal(t, 200, code, string(byteBuffer))

		response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
		require.NoError(t, err)

		assert.Empty(t, response.Err)
		assert.NotEmpty(t, response.TraceID)
		assert.Equal(t, uint(1000000500), response.Amount)
	}

	{
		{ // Transfer request
			code, byteBuffer := server.CallAPITransfer(ctx, walletReference1, walletReference2, 500)
			require.Equal(t, 200, code, string(byteBuffer))

			response, err := utils.UnmarshalWalletTransferResponse(byteBuffer)
			require.NoError(t, err)

			assert.Empty(t, response.Err)
			assert.NotEmpty(t, response.TraceID)
		}
		{ // GetBalance request
			code, byteBuffer := server.CallAPIGetBalance(ctx, walletReference1)
			require.Equal(t, 200, code, string(byteBuffer))

			response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
			require.NoError(t, err)

			assert.Empty(t, response.Err)
			assert.NotEmpty(t, response.TraceID)
			assert.Equal(t, uint(1000000000), response.Amount)
		}
		{ // Delete request
			foundError := server.Journal.Wait(func(event debuglogger.UpdateEvent) bool {
				if event.Data.Error != nil {
					return strings.Contains(event.Data.Error.Error(), "not implemented")
				}
				return false
			})

			server.CallAPIDelete(ctx, walletReference1)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone(), foundError)

			// TODO: change code when realization will be done and remove ErrorFilter on line: 28
			// TODO: after https://insolar.atlassian.net/browse/PLAT-416
			// code, byteBuffer := server.CallAPIDelete(ctx, walletReference1)
			// require.Equal(t, 200, code, string(byteBuffer))
			//
			// response, err := utils.UnmarshalWalletDeleteResponse(byteBuffer)
			// require.NoError(t, err)
			//
			// assert.Empty(t, response.Err)
			// assert.NotEmpty(t, response.TraceID)
		}
	}
}
