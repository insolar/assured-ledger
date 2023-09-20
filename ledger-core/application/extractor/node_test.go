package extractor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
)

func TestNodeInfoResponse(t *testing.T) {
	testPK := "test_public_key"
	testRole := member.PrimaryRoleVirtual

	testValue := struct {
		PublicKey string
		Role      member.PrimaryRole
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
	testRole := member.PrimaryRoleVirtual

	testValue := struct {
		PublicKey string
		Role      member.PrimaryRole
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
