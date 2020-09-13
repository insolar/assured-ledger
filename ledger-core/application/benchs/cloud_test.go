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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func Benchmark_MultiPulseOnCloud_Timed(b *testing.B) {
	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			teardown, cloudRunner, err := runCloud(numNodes)
			if err != nil {
				b.Fatal("network run failed")
			}

			runner := &benchRunner{
				benchLimiter: &timedBenchLimiter{pulseTime: int(cloudRunner.ConfProvider.PulsarConfig.Pulsar.PulseTime)},
				n:            100,
			}

			apiAddrs := []string{}
			for _, el := range cloudRunner.ConfProvider.GetAppConfigs() {
				apiAddrs = append(apiAddrs, el.TestWalletAPI.Address)
			}

			run := runner.runBenchOnNetwork(b, numNodes)
			retCode := run(apiAddrs)
			if retCode != 0 {
				b.Fatal("failed test run")
			}

			teardown()
		})
	}
}

// TODO: Stop pulsar after network startup.
// This functionality should be done in PLAT-782
func Benchmark_SinglePulseOnCloud_N(b *testing.B) {
	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			b.ResetTimer()
			b.StopTimer()
			teardown, cloudRunner, err := runCloud(numNodes)
			if err != nil {
				b.Fatal("network run failed")
			}

			for i := 0; i <= b.N; i++ {
				runner := benchRunner{
					benchLimiter: &countedBenchLimiter{targetCount: int32(i + 1), currentCount: 0},
					n:            i + 1,
				}

				apiAddrs := []string{}
				for _, el := range cloudRunner.ConfProvider.GetAppConfigs() {
					apiAddrs = append(apiAddrs, el.TestWalletAPI.Address)
				}

				run := runner.runBenchOnNetwork(b, numNodes)
				retCode := run(apiAddrs)
				if retCode != 0 {
					b.Fatal("failed test run")
				}
			}

			teardown()
		})
	}
}

func runCloud(numNodes int) (func(), *launchnet.CloudRunner, error) {
	runner := &launchnet.CloudRunner{}
	runner.SetNumVirtuals(numNodes)
	runner.PrepareConfig()
	runner.ConfProvider.PulsarConfig.Pulsar.PulseTime = 3000
	runner.ConfProvider.BaseConfig.Log.Level = "Fatal"

	var apiAddresses []string
	for i, el := range runner.ConfProvider.GetAppConfigs() {
		runner.ConfProvider.GetAppConfigs()[i].Log.Level = "Fatal"
		apiAddresses = append(apiAddresses, el.TestWalletAPI.Address)
	}
	teardown, err := runner.SetupCloud()
	if err != nil {
		return func() {}, nil, throw.W(err, "Can't run cloud")
	}

	return teardown, runner, nil
}
