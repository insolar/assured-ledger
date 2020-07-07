// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package deduplication

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func TestConstructor_SamePulse_WhileExecution(t *testing.T) {
	t.Log("C4998")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		isolation = contract.ConstructorIsolation()
		outgoing  = server.BuildRandomOutgoingWithPulse()
		class     = gen.UniqueGlobalRef()
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		return false
	})

	synchronizeExecution := synchronization.NewPoint(1)

	executionFn := func(ctx execution.Context) {
		synchronizeExecution.Synchronize()
	}

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte("234"))

		executionMock := runnerMock.AddExecutionMock(outgoing.String())
		executionMock.AddStart(executionFn, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
	}

	{
		// send first call request
		server.SendPayload(ctx, &pl)
	}

	// await first SMExecute go to step execute (in this point machine is still not publish result to table in SMObject)
	testutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())

	{
		// send second call request
		server.SendPayload(ctx, &pl)
	}

	// second SMExecute should stop in deduplication algorithm and she should not send result because she started during execution first machine
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&execute.SMExecute{}, 1))

	// wakeup first SMExecute
	synchronizeExecution.WakeUp()

	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&execute.SMExecute{}, 1))
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

func TestConstructor_SamePulse_AfterExecution(t *testing.T) {
	t.Log("C5005")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		isolation = contract.ConstructorIsolation()
		outgoing  = server.BuildRandomOutgoingWithPulse()
		class     = gen.UniqueGlobalRef()
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		return false
	})

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte("234"))

		executionMock := runnerMock.AddExecutionMock(outgoing.String())
		executionMock.AddStart(func(ctx execution.Context) {

		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
	}

	{
		// send first call request
		server.SendPayload(ctx, &pl)
	}

	// await first SMExecute go completed work (after complete SMExecute publish result to table in SMObject)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&execute.SMExecute{}, 1))

	{
		// send second call request
		server.SendPayload(ctx, &pl)
	}

	// second SMExecute should send result again because she started after first machine complete
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&execute.SMExecute{}, 1))
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, 2, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}
