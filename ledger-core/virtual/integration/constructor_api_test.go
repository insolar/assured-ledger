// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func Test_API_Create(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4837")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	code, byteBuffer := server.CallAPICreateWallet(ctx)
	if !assert.Equal(t, 200, code) {
		t.Log(string(byteBuffer))
	} else {
		walletResponse, err := utils.UnmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)

		assert.Empty(t, walletResponse.Err)
		assert.NotEmpty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)
	}
}
