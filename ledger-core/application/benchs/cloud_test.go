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

func BenchmarkMultiPulseOnCloud(b *testing.B) {
	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			runner := launchnet.CloudRunner{}
			runner.SetNumVirtuals(numNodes)
			runner.PrepareConfig()
			runner.ConfProvider.PulsarConfig.Pulsar.PulseTime = 3000
			runner.ConfProvider.BaseConfig.Log.Level = "Warn"

			var apiAddresses []string
			for i, el := range runner.ConfProvider.GetAppConfigs() {
				runner.ConfProvider.GetAppConfigs()[i].Log.Level = "Warn"
				apiAddresses = append(apiAddresses, el.TestWalletAPI.Address)
			}
			teardown, err := runner.SetupCloud()
			if err != nil {
				b.Error("network run failed", err)
				b.Fatal("failed test run", err)
			}

			run := runBenchOnNetwork(b, numNodes)
			retCode := run(apiAddresses)
			if retCode != 0 {
				b.Error("network run failed")
				b.Fatal("failed test run")
			}

			teardown()
		})
	}
}
