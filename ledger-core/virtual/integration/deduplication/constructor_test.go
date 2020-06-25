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

func TestConstructor_SamePulse(t *testing.T) {
	var (
		synchronizeExecution *synchronization.Point
		firstMessage         bool
	)

	tests := []struct {
		name          string
		resultCount   int
		testRailID    string
		skipped       string
		executionFn   func(ctx execution.Context)
		sendMessageFn func()
	}{
		{
			name:        "while executing",
			resultCount: 1,
			testRailID:  "C4998",
			skipped:     "",
			executionFn: func(ctx execution.Context) {
				synchronizeExecution.Synchronize()
			},
		},
		{
			name:        "after execution",
			resultCount: 2,
			testRailID:  "C5005",
			skipped:     "https://insolar.atlassian.net/browse/PLAT-386",
			sendMessageFn: func() {
				if firstMessage {
					synchronizeExecution.Synchronize()
					firstMessage = false
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// can't run in parallel, got shared state
			t.Log(test.testRailID)
			if test.skipped != "" {
				t.Skip(test.skipped)
			}

			mc := minimock.NewController(t)
			// prepare state
			synchronizeExecution = synchronization.NewPoint(1)
			firstMessage = false

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

			runnerMock := logicless.NewServiceMock(ctx, mc, nil)
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				isolation = contract.ConstructorIsolation()
				outgoing  = server.RandomLocalWithPulse()
				class     = gen.UniqueGlobalRef()
			)

			pl := payload.VCallRequest{
				CallType:       payload.CTConstructor,
				CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
				Callee:         class,
				CallSiteMethod: "New",
				CallOutgoing:   outgoing,
			}

			{
				requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
				requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte("234"))

				executionMock := runnerMock.AddExecutionMock(utils.CalculateOutgoing(pl).String())
				executionMock.AddStart(test.executionFn, &execution.Update{
					Type:   execution.Done,
					Result: requestResult,
				})
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
				if test.sendMessageFn != nil {
					test.sendMessageFn()
				}
				return false
			})

			{
				server.SendPayload(ctx, &pl)
			}

			testutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())

			{
				server.SendPayload(ctx, &pl)
			}

			synchronizeExecution.WakeUp()

			testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			{
				assert.Equal(t, test.resultCount, typedChecker.VCallResult.Count())
			}

			mc.Finish()

		})
	}
}
