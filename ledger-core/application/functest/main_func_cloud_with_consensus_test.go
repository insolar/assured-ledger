// +build functest,!cloud,cloud_with_consensus

package functest

import (
	"os"
	"testing"

	"github.com/google/gops/agent"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func init() {
	// starts gops agent https://github.com/google/gops on default addr (127.0.0.1:0)
	if err := agent.Listen(agent.Options{}); err != nil {
		panic(err)
	}
}

var (
	numNodes int
)

func getNodesCount() (int, error) {
	return numNodes, nil
}

func TestMain(m *testing.M) {
	instestlogger.SetTestOutputWithStub()

	numVirtual, numLight, numHeavy := getTestNodesSetup()

	numNodes = numVirtual + numLight + numHeavy

	os.Exit(launchnet.RunCloudWithConsensus(numVirtual, numLight, numHeavy, func() int {
		return m.Run()
	}))
}
