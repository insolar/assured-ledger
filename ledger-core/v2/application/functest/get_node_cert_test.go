// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest

package functest

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/testutils/launchnet"

	"github.com/stretchr/testify/require"
)

func TestNodeCert(t *testing.T) {
	const TESTPUBLICKEY = "some_fancy_public_key"
	const testRole = "virtual"
	res, err := signedRequest(t, launchnet.TestRPCUrl, &launchnet.Root,
		"contract.registerNode", map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	require.NoError(t, err)

	body := getRPSResponseBody(t, launchnet.TestRPCUrl, postParams{
		"jsonrpc": "2.0",
		"method":  "cert.get",
		"id":      1,
		"params":  map[string]string{"ref": res.(string)},
	})

	require.NotEqual(t, "", string(body))
}
