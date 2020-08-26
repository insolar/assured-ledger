package integration

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_VCachedMemoryRequest(t *testing.T) {
	insrail.LogSkipCase(t, "", "https://insolar.atlassian.net/browse/PLAT-747")
	defer commonTestUtils.LeakTester(t)
	testCases := []struct {
		name        string
		method      bool
		constructor bool
		pending     bool
	}{
		{name: "Object state created from constructor", constructor: true},
		{name: "Object state created from method", method: true},
		{name: "Object state created from pending", pending: true},
	}

	const (
		newState     = "new state"
		methodState  = "method state"
		pendingState = "pending state"
	)

	for _, cases := range testCases {
		t.Run(cases.name, func(t *testing.T) {

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, t, nil)
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			var (
				classA        = server.RandomGlobalWithPulse()
				objectGlobal  = reference.NewSelf(classA.GetLocal())
				prevPulse     = server.GetPulse().PulseNumber
				expectedState string
			)

			server.IncrementPulse(ctx)

			if cases.constructor {

				pl := utils.GenerateVCallRequestConstructor(server)
				pl.Caller = classA
				callOutgoing := pl.CallOutgoing

				result := requestresult.New([]byte(newState), objectGlobal)

				key := callOutgoing.String()
				runnerMock.AddExecutionMock(key).
					AddStart(nil, &execution.Update{
						Type:   execution.Done,
						Result: result,
					})
				runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
					Interference: pl.CallFlags.GetInterference(),
					State:        pl.CallFlags.GetState(),
				}, nil)

				server.SendPayload(ctx, pl)
				expectedState = newState

			} else {
				Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)
			}

			// call method
			if cases.method {
				pl := utils.GenerateVCallRequestMethod(server)
				pl.Callee = objectGlobal
				pl.CallSiteMethod = "ordered"
				callOutgoing := pl.CallOutgoing

				result := requestresult.New([]byte(methodState), objectGlobal)

				key := callOutgoing.String()
				runnerMock.AddExecutionMock(key).
					AddStart(nil, &execution.Update{
						Type:   execution.Done,
						Result: result,
					})
				runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
					Interference: pl.CallFlags.GetInterference(),
					State:        pl.CallFlags.GetState(),
				}, nil)

				server.SendPayload(ctx, pl)
				expectedState = methodState
			}

			// pending
			if cases.pending {
				pl := utils.GenerateVCallRequestMethod(server)
				pl.Callee = objectGlobal
				pl.CallSiteMethod = "Pending"
				callOutgoing := pl.CallOutgoing
				key := callOutgoing.String()

				synchronizeExecution := synchronization.NewPoint(1)
				defer synchronizeExecution.Done()

				// add ExecutionMocks to runnerMock
				{
					runnerMock.AddExecutionClassify(key, contract.MethodIsolation{pl.CallFlags.GetInterference(), pl.CallFlags.GetState()}, nil)

					requestResult := requestresult.New([]byte("call result"), gen.UniqueGlobalRef())

					newObjDescriptor := descriptor.NewObject(
						reference.Global{}, reference.Local{}, gen.UniqueGlobalRef(), []byte(""), false,
					)
					requestResult.SetAmend(newObjDescriptor, []byte(pendingState))

					objectExecutionMock := runnerMock.AddExecutionMock(key)
					objectExecutionMock.AddStart(
						func(_ execution.Context) {
							synchronizeExecution.Synchronize()
						},
						&execution.Update{
							Type:   execution.Done,
							Result: requestResult,
						},
					)
				}

				// add checks to typedChecker
				{
					typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
						assert.Equal(t, objectGlobal, report.Object)
						assert.Equal(t, payload.StateStatusReady, report.Status)
						assert.Zero(t, report.DelegationSpec)
						return false
					})
					typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
						p2 := server.GetPulse().PulseNumber

						assert.Equal(t, objectGlobal, request.Callee)
						assert.Zero(t, request.DelegationSpec)

						approver := gen.UniqueGlobalRef()

						firstTokenValue := payload.CallDelegationToken{
							TokenTypeAndFlags: payload.DelegationTokenTypeCall,
							PulseNumber:       p2,
							Callee:            request.Callee,
							Outgoing:          request.CallOutgoing,
							DelegateTo:        server.JetCoordinatorMock.Me(),
							Approver:          approver,
						}
						msg := payload.VDelegatedCallResponse{
							Callee:                 request.Callee,
							CallIncoming:           request.CallIncoming,
							ResponseDelegationSpec: firstTokenValue,
						}

						server.SendPayload(ctx, &msg)
						return false
					})
					typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
						assert.Equal(t, objectGlobal, finished.Callee)
						assert.NotEmpty(t, finished.LatestState)
						assert.Equal(t, []byte(pendingState), finished.LatestState.State)

						return false
					})
					typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
						assert.Equal(t, objectGlobal, res.Callee)
						assert.Equal(t, []byte("call result"), res.ReturnArguments)
						return false
					})
				}
				executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

				server.SendPayload(ctx, pl)

				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())

				tokenRequestDone := server.Journal.Wait(
					predicate.ChainOf(
						predicate.NewSMTypeFilter(&execute.SMDelegatedTokenRequest{}, predicate.AfterAnyStopOrError),
						predicate.NewSMTypeFilter(&execute.SMExecute{}, predicate.BeforeStep((&execute.SMExecute{}).StepWaitExecutionResult)),
					),
				)

				server.IncrementPulseAndWaitIdle(ctx)

				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, tokenRequestDone)

				synchronizeExecution.WakeUp()
				expectedState = pendingState
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

				{
					assert.Equal(t, 1, typedChecker.VCallResult.Count())
					assert.Equal(t, 1, typedChecker.VStateReport.Count())
					assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

					assert.Equal(t, 1, typedChecker.VDelegatedCallRequest.Count())
				}
			}

			var stateRef payload.Reference

			server.IncrementPulse(ctx)
			typedChecker.VStateReport.Set(func(rep *payload.VStateReport) bool {
				stateRef = rep.LatestValidatedState
				return false // no resend msg
			})

			typedChecker.VCachedMemoryResponse.Set(func(resp *payload.VCachedMemoryResponse) bool {
				require.Equal(t, objectGlobal, resp.Object)
				require.Equal(t, expectedState, resp.Memory)
				return false
			})

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			{
				cachReq := &payload.VCachedMemoryRequest{
					Object:  objectGlobal,
					StateID: stateRef,
				}
				server.SendPayload(ctx, cachReq)
			}
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VCachedMemoryResponse.Count())
			mc.Finish()

		})
	}

}
