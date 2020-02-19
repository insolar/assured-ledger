// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package foundation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimPublicKey(t *testing.T) {
	for _, tc := range []struct {
		input  string
		result string
	}{
		{
			input:  "asDafasf",
			result: "asDafasf",
		},
		{
			input:  "-----BEGIN RSA PUBLIC KEY-----\naSDafasf\n-----END RSA PUBLIC KEY-----",
			result: "aSDafasf",
		},
	} {
		require.Equal(t, tc.result, TrimPublicKey(tc.input))
	}
}
