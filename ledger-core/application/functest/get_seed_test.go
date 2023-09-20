// +build functest

package functest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
)

func TestGetSeed(t *testing.T) {
	seed1 := getSeed(t)
	seed2 := getSeed(t)

	require.NotEqual(t, seed1, seed2)

}

func getSeed(t testing.TB) string {
	body := getRPSResponseBody(t, launchnet.TestRPCUrl, postParams{
		"jsonrpc": "2.0",
		"method":  "node.getSeed",
		"id":      "",
	})
	getSeedResponse := &getSeedResponse{}
	unmarshalRPCResponse(t, body, getSeedResponse)
	require.NotNil(t, getSeedResponse.Result)
	return getSeedResponse.Result.Seed
}
