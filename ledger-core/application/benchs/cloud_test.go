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

	for numNodes := uint(2); numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			cloudRunner := launchnet.PrepareCloudRunner(
				launchnet.WithRunning(numNodes, 0, 0),
				launchnet.WithDefaultLogLevel(log.FatalLevel))
			if res := cloudRunner.Run(func(apiAddresses []string) int {
				runner := benchRunner{
					benchLimiterFactory: func(n int) benchLimiter {
						return &countedBenchLimiter{targetCount: int32(n), currentCount: 0}
					},
				}

				run := runner.runBenchOnNetwork(b)
				return run(apiAddresses)
			}); res != 0 {
				b.Fatal("Failed to run benchmark")
			}
		})
	}
}

func Benchmark_SinglePulseOnCloud_N(b *testing.B) {
	instestlogger.SetTestOutput(b)
	for numNodes := uint(2); numNodes <= 5; numNodes++ {
		b.Run(fmt.Sprintf("Nodes %d", numNodes), func(b *testing.B) {
			cloudRunner := launchnet.PrepareCloudRunner(
				launchnet.WithRunning(numNodes, 0, 0),
				launchnet.WithDefaultLogLevel(log.FatalLevel),
				launchnet.WithCloudFileLogging(),
				launchnet.WithPulsarMode(launchnet.ManualPulsar))
			if res := cloudRunner.Run(func(apiAddresses []string) int {
				runner := benchRunner{
					benchLimiterFactory: func(n int) benchLimiter {
						return &countedBenchLimiter{targetCount: int32(n), currentCount: 0}
					},
				}

				run := runner.runBenchOnNetwork(b)
				return run(apiAddresses)
			}); res != 0 {
				b.Fatal("Failed to run benchmark")
			}
		})
	}
}
