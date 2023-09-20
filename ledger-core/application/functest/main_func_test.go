// +build functest,!cloud,!cloud_with_consensus

package functest

import (
	"os"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func getNodesCount() (int, error) {
	return launchnet.GetNodesCount()
}

func TestMain(m *testing.M) {
	instestlogger.SetTestOutputWithStub()

	os.Exit(launchnet.Run(func() int {
		return m.Run()
	}))
}
