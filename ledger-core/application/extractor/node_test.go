// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package extractor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/foundation"
)

func TestNodeInfoResponse(t *testing.T) {
	testPK := "test_public_key"
	testRole := node.StaticRoleVirtual

	testValue := struct {
		PublicKey string
		Role      node.StaticRole
	}{
		PublicKey: testPK,
		Role:      testRole,
	}

	data, err := foundation.MarshalMethodResult(testValue, nil)
	require.NoError(t, err)

	pk, role, err := NodeInfoResponse(data)

	require.NoError(t, err)
	require.Equal(t, testPK, pk)
	require.Equal(t, testRole.String(), role)
}

func TestNodeInfoResponse_ErrorResponse(t *testing.T) {
	testPK := "test_public_key"
	testRole := node.StaticRoleVirtual

	testValue := struct {
		PublicKey string
		Role      node.StaticRole
	}{
		PublicKey: testPK,
		Role:      testRole,
	}
	contractErr := &foundation.Error{S: "Custom test error"}

	data, err := foundation.MarshalMethodResult(testValue, contractErr)
	require.NoError(t, err)

	pk, role, err := NodeInfoResponse(data)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Has error in response")
	require.Contains(t, err.Error(), "Custom test error")
	require.Equal(t, "", pk)
	require.Equal(t, "", role)
}

func TestNodeInfoResponse_UnmarshalError(t *testing.T) {
	testValue := "some_no_valid_data"

	data, err := insolar.Serialize(testValue)
	require.NoError(t, err)

	pk, role, err := NodeInfoResponse(data)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Can't unmarshal response")
	require.Equal(t, "", pk)
	require.Equal(t, "", role)
}
