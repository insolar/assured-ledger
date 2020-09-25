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
	"github.com/insolar/assured-ledger/ledger-core/log"
)

func Benchmark_MultiPulseOnCloud_Timed(b *testing.B) {
	b.Skip("changing of pulse doesn't work now")

	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			if res := launchnet.RunCloud(func(apiAddresses []string) int {
				runner := benchRunner{
					benchLimiterFactory: func(n int) benchLimiter {
						return &countedBenchLimiter{targetCount: int32(n), currentCount: 0}
					},
				}

				run := runner.runBenchOnNetwork(b)
				return run(apiAddresses)
			}, launchnet.WithNumVirtual(numNodes), launchnet.WithDefaultLogLevel(log.FatalLevel)); res != 0 {
				b.Fatal("Failed to run benchmark")
			}
		})
	}
}

func Benchmark_SinglePulseOnCloud_N(b *testing.B) {
	instestlogger.SetTestOutput(b)
	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			if res := launchnet.RunCloud(func(apiAddresses []string) int {
				runner := benchRunner{
					benchLimiterFactory: func(n int) benchLimiter {
						return &countedBenchLimiter{targetCount: int32(n), currentCount: 0}
					},
				}

				run := runner.runBenchOnNetwork(b)
				return run(apiAddresses)
			}, launchnet.WithNumVirtual(numNodes), launchnet.WithDefaultLogLevel(log.FatalLevel), launchnet.WithPulsarMode(launchnet.ManualPulsar)); res != 0 {
				b.Fatal("Failed to run benchmark")
			}
		})
	}
}
