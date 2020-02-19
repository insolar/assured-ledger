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

const NOTEXISTINGPUBLICKEY = "not_existing_public_key"

func getNodeRefSignedCall(t *testing.T, params map[string]interface{}) (string, error) {
	res, err := signedRequest(t, launchnet.TestRPCUrl, &launchnet.Root, "contract.getNodeRef", params)
	if err != nil {
		return "", err
	}
	return res.(string), nil
}

func TestGetNodeRefByPublicKey(t *testing.T) {
	const testRole = "light_material"
	ref, err := registerNodeSignedCall(t, map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	require.NoError(t, err)
	require.NotNil(t, ref)

	nodeRef, err := getNodeRefSignedCall(t, map[string]interface{}{"publicKey": TESTPUBLICKEY})
	require.NoError(t, err)
	require.Equal(t, ref, nodeRef)
}

func TestGetNodeRefByNotExistsPK(t *testing.T) {
	const testRole = "light_material"
	ref, err := registerNodeSignedCall(t, map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	require.NoError(t, err)
	require.NotNil(t, ref)

	_, err = signedRequestWithEmptyRequestRef(t, launchnet.TestRPCUrl, &launchnet.Root,
		"contract.getNodeRef", map[string]interface{}{"publicKey": NOTEXISTINGPUBLICKEY})
	data := checkConvertRequesterError(t, err).Data
	require.Contains(t, data.Trace, "not_existing_public_key")
}

func TestGetNodeRefInvalidParams(t *testing.T) {
	const testRole = "light_material"
	ref, err := registerNodeSignedCall(t, map[string]interface{}{"publicKey": TESTPUBLICKEY, "role": testRole})
	require.NoError(t, err)
	require.NotNil(t, ref)

	_, err = signedRequestWithEmptyRequestRef(t, launchnet.TestRPCUrl, &launchnet.Root,
		"contract.getNodeRef", map[string]interface{}{"publicKey": 123})
	data := checkConvertRequesterError(t, err).Data
	require.Contains(t, data.Trace, "failed to get 'publicKey' param")
}
