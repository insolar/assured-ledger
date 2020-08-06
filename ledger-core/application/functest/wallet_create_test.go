// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
)

// Creates wallets and check Reference format in response body.
func TestWalletCreate(t *testing.T) {
	insrail.LogCase(t, "C4854")

	status := getStatus(t)
	require.NotEqual(t, 0, status.WorkingListSize, "not enough nodes to run test")
	count := 2 * status.WorkingListSize

	t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
		for i := 0; i < count; i++ {
			url := getURL(walletCreatePath, "", "")
			rawResp, err := sendAPIRequest(url, nil)
			require.NoError(t, err, "failed to send request or get response body")

			resp, err := unmarshalWalletCreateResponse(rawResp)
			require.NoError(t, err, "failed to unmarshal response")
			require.Empty(t, resp.Err, "problem during execute request")
			require.Contains(t, resp.Ref, "insolar:", "wrong reference")
			require.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")
		}
	})
}
