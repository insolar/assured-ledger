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

func Benchmark_MultiPulseFullNetwork_Timed(b *testing.B) {
	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			runner := &benchRunner{
				benchLimiter: &timedBenchLimiter{pulseTime: launchnet.GetPulseTime()},
				n:            100,
			}

			res := launchnet.CustomRunWithPulsar(numNodes, 0, 0, runner.runBenchOnNetwork(b, numNodes))
			if res != 0 {
				b.Error("network run failed")
				b.Fatal("failed test run")
			}
		})
	}
}

func Benchmark_SinglePulseFullNetwork_N(b *testing.B) {
	instestlogger.SetTestOutput(b)

	for numNodes := 2; numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			b.ResetTimer()
			for i := 1; i <= b.N; i++ {
				runner := &benchRunner{
					benchLimiter: &countedBenchLimiter{targetCount: int32(i), currentCount: 0},
					n:            i,
				}
				b.StopTimer()
				res := launchnet.CustomRunWithPulsar(numNodes, 0, 0, runner.runBenchOnNetwork(b, numNodes))
				if res != 0 {
					b.Error("network run failed")
					b.Fatal("failed test run")
				}
			}
		})
	}
}
