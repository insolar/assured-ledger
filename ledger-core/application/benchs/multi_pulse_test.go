// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package benchs

import (
	"fmt"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func BenchmarkMultiPulse(b *testing.B) {
	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {

			res := launchnet.CustomRunWithPulsar(numNodes, 0, 0, runBenchOnNetwork(numNodes))

			if res != 0 {
				b.Error("network run failed")
				b.Fatal("failed test run")
			}
		})
	}
}
