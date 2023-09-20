package extractor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
)

func TestInfoResponse(t *testing.T) {
	testValue, _ := json.Marshal(map[string]interface{}{
		"root_member": "test_root_member",
		"node_domain": "test_node_domain",
	})
	expectedValue := Info{}
	_ = json.Unmarshal(testValue, &expectedValue)

	data, err := foundation.MarshalMethodResult(testValue, nil)
	require.NoError(t, err)

	info, err := InfoResponse(data)

	require.NoError(t, err)
	require.Equal(t, &expectedValue, info)
}

func TestInfoResponse_ErrorResponse(t *testing.T) {
	contractErr := errors.New("Custom test error")

	data, err := foundation.MarshalMethodErrorResult(contractErr)
	require.NoError(t, err)

	info, err := InfoResponse(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Has error in response")
	require.Contains(t, err.Error(), "Custom test error")
	require.Nil(t, info)
}

func TestInfoResponse_UnmarshalError(t *testing.T) {
	testValue := "some_no_valid_data"

	data, err := insolar.Serialize(testValue)
	require.NoError(t, err)

	info, err := InfoResponse(data)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Can't unmarshal")
	require.Nil(t, info)
}
