package integration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_SemaphoreLimitNotExceeded(t *testing.T) {
	insrail.LogCase(t, "C5137")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, t, func(execution execution.Context) interface{} {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)

	semaphoreParallelism := 3
	server.SetMaxParallelism(semaphoreParallelism)

	server.Init(ctx)

	var (
		numObject = 40
		objects   = make([]reference.Global, 0, numObject)
	)

	// Create objects
	{
		pulse := server.GetPulse().PulseNumber
		server.IncrementPulseAndWaitIdle(ctx)

		for i := 0; i < numObject; i++ {
			objects = append(objects, gen.UniqueGlobalRefWithPulse(pulse))
			Method_PrepareObject(ctx, server, rms.StateStatusReady, objects[i], pulse)
		}
	}

	var (
		interferenceFlag = isolation.CallIntolerable
		stateFlag        = isolation.CallDirty
		numParallelExecs = int64(0)
	)

	syncChan := make(chan bool, semaphoreParallelism*2)
	// Add execution mocks
	{
		for i := 0; i < numObject; i++ {
			key := objects[i].String()
			runnerMock.AddExecutionMock(key).
				AddStart(func(ctx execution.Context) {
					lastNum := atomic.AddInt64(&numParallelExecs, 1)
					require.LessOrEqual(t, lastNum, int64(semaphoreParallelism))
					<-syncChan
					atomic.AddInt64(&numParallelExecs, -1)
				}, &execution.Update{
					Type:   execution.Done,
					Result: requestresult.New([]byte("345"), objects[0]),
				})
			runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
				Interference: interferenceFlag,
				State:        stateFlag,
			}, nil)
		}
	}

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		return false // no resend msg
	})

	// Send VCallRequests
	{
		for i := 0; i < numObject; i++ {
			pl := utils.GenerateVCallRequestMethod(server)
			pl.CallFlags = rms.BuildCallFlags(interferenceFlag, stateFlag)
			pl.Callee.Set(objects[i])
			pl.CallSiteMethod = objects[i].String()

			server.SendPayload(ctx, pl)
		}
	}

	// Wait after every execution to increase probability that multiple executions are in queue
	for i := 0; i < numObject; i++ {
		select {
		case syncChan <- true:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "timeout")
		}

	}

	commonTestUtils.WaitSignalsTimed(t, time.Second*10, typedChecker.VCallResult.Wait(ctx, numObject))

	mc.Finish()
}
