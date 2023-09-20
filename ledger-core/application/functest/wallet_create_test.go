// +build functest

package functest

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
)

// Creates wallets and check Reference format in response body.
func TestWalletCreate(t *testing.T) {
	insrail.LogCase(t, "C4854")

	status := getStatus(t)
	require.NotEqual(t, 0, status.WorkingListSize, "not enough nodes to run test")
	count := 2 * status.WorkingListSize

	ctx := context.Background()

	t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
		for i := 0; i < count; i++ {
			_, err := testutils.CreateSimpleWallet(ctx)
			require.NoError(t, err, "failed to send request or get response body")
		}
	})
}
