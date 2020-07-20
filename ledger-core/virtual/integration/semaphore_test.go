// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func TestVirtual_SemaphoreLimitNotExceeded(t *testing.T) {
	t.Log("C5137")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, t, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)

	semaphoreParallelism := 3
	server.SetMaxParallelism(semaphoreParallelism)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		numObject = 40
		objects   = make([]reference.Global, 0, numObject)
	)

	// Create objects
	{
		for i := 0; i < numObject; i++ {
			objects = append(objects, reference.NewSelf(server.RandomLocalWithPulse()))
			Method_PrepareObject(ctx, server, payload.Ready, objects[i])
		}
	}

	var (
		class            = testwallet.GetClass()
		interferenceFlag = contract.CallIntolerable
		stateFlag        = contract.CallDirty
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

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		return false // no resend msg
	})

	// Send VCallRequests
	{
		for i := 0; i < numObject; i++ {
			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(interferenceFlag, stateFlag),
				Caller:              server.GlobalCaller(),
				Callee:              objects[i],
				CallSiteDeclaration: class,
				CallSiteMethod:      objects[i].String(),
				CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
			}
			server.SendPayload(ctx, &pl)
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

	testutils.WaitSignalsTimed(t, time.Second*10, typedChecker.VCallResult.Wait(ctx, numObject))

	mc.Finish()
}
