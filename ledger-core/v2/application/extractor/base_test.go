// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package extractor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common/foundation"
)

func TestStringResponse(t *testing.T) {
	testValue := "test_string"

	data, err := foundation.MarshalMethodResult(testValue, nil)
	require.NoError(t, err)

	result, err := stringResponse(data)

	require.NoError(t, err)
	require.Equal(t, testValue, result)
}

func TestStringResponse_ErrorResponse(t *testing.T) {
	testValue := "test_string"
	contractErr := &foundation.Error{S: "Custom test error"}

	data, err := foundation.MarshalMethodResult(testValue, contractErr)
	require.NoError(t, err)

	result, err := stringResponse(data)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Has error in response")
	require.Contains(t, err.Error(), "Custom test error")
	require.Equal(t, "", result)
}

func TestStringResponse_UnmarshalError(t *testing.T) {
	testValue := "some_no_valid_data"

	data, err := insolar.Serialize(testValue)
	require.NoError(t, err)

	result, err := stringResponse(data)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Can't unmarshal")
	require.Equal(t, "", result)
}
