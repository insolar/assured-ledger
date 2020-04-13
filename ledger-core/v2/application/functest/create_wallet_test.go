// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func getCount(args []string) (int, error) {
	for _, s := range args {
		if strings.Contains(s, "test.count") {
			countInfo := strings.Split(s, "=")
			return strconv.Atoi(countInfo[1])
		}
	}
	return 1, nil
}

// Creates wallets and check Reference format in response body.
func TestCreateWallet(t *testing.T) {
	count, err := getCount(os.Args)
	require.NoError(t, err, "failed to parse -test.count param")

	t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
		for i := 0; i < count; i++ {
			url := getURL(createWalletPath, "", "")
			rawResp, err := sendAPIRequest(url, nil)
			require.NoError(t, err, "failed to send request or get response body")

			resp, err := unmarshalCreateWalletResponse(rawResp)
			require.NoError(t, err, "failed to unmarshal response")
			require.Nil(t, resp.Err, "problem during execute request")
			require.Contains(t, resp.Ref, "insolar:", "wrong reference")
			require.NotEmpty(t, resp.TraceID, "traceID mustn't be empty")
		}
	})
}
