// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package deduplication

import (
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
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

	awaitStopFirstSM := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)
	awaitStopSecondSM := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

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
	testutils.WaitSignalsTimed(t, 10*time.Second, awaitStopSecondSM)

	// wakeup first SMExecute
	synchronizeExecution.WakeUp()

	testutils.WaitSignalsTimed(t, 10*time.Second, awaitStopFirstSM)
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
		executionMock.AddStart(nil, &execution.Update{
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

	awaitStopFirstSM := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	awaitStopSecondSM := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	{
		// send first call request
		server.SendPayload(ctx, &pl)
	}

	// await first SMExecute go completed work (after complete SMExecute publish result to table in SMObject)
	testutils.WaitSignalsTimed(t, 10*time.Second, awaitStopFirstSM)

	{
		// send second call request
		server.SendPayload(ctx, &pl)
	}

	// second SMExecute should send result again because she started after first machine complete
	testutils.WaitSignalsTimed(t, 10*time.Second, awaitStopSecondSM)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, 2, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

type DeduplicationDifferentPulsesCase struct {
	*utils.TestCase

	VState           payload.VStateReport
	vStateSendBefore bool

	VFindCallRequestExpected bool
	VFindCall                *payload.VFindCallResponse

	VDelegatedCall             *payload.VDelegatedCallRequest
	VDelegatedCallBadReference bool
	VDelegatedRequestFinished  *payload.VDelegatedRequestFinished

	VCallResultExpected    bool
	ExecutionExpected      bool
	ExecuteShouldHaveError string
	ExpectedResult         []byte
}

func (test *DeduplicationDifferentPulsesCase) TestRun(t *testing.T) {
	test.TestCase.Run(t, test.run)

	test.Name = test.Name + ", state already sent"
	test.vStateSendBefore = true
	test.TestCase.Run(t, test.run)
}

func (test DeduplicationDifferentPulsesCase) vCallResultCount() int {
	if test.VCallResultExpected {
		return 1
	}
	return 0
}

func (test DeduplicationDifferentPulsesCase) vStateRequestCount() int {
	if test.vStateSendBefore {
		return 0
	}
	return 1
}

func (test DeduplicationDifferentPulsesCase) vFindCallCount() int {
	if test.VFindCall != nil {
		return 1
	}
	return 0
}

func (test DeduplicationDifferentPulsesCase) vDelegatedCallResponseCount() int {
	if test.VDelegatedCall != nil {
		return 1
	}
	return 0
}

var (
	ExecutionResultFromExecutor     = []byte{1, 2, 3}
	ExecutionResultFromPreviousNode = []byte{2, 3, 4}
)

func (test *DeduplicationDifferentPulsesCase) run(t *testing.T) {
	var (
		isolation     = contract.ConstructorIsolation()
		class         = gen.UniqueGlobalRef()
		previousPulse = test.Server.GetPulse().PulseNumber
		outgoingLocal = gen.UniqueLocalRefWithPulse(previousPulse)
		outgoing      = reference.NewRecordOf(test.Server.GlobalCaller(), outgoingLocal)
		object        = reference.NewSelf(outgoingLocal)
		executeDone   = test.Server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		foundError    = synckit.ClosedChannel()

		ctx    = test.Context
		server = test.Server
	)

	if test.ExecuteShouldHaveError != "" {
		foundError = test.Server.Journal.Wait(func(event debuglogger.UpdateEvent) bool {
			if event.Data.Error != nil {
				stack := throw.DeepestStackTraceOf(event.Data.Error)
				return strings.Contains(stack.StackTraceAsText(), test.ExecuteShouldHaveError)
			}
			return false
		})
	}

	// populate needed VStateReport fields
	test.VState.Object = object
	test.VState.OrderedPendingEarliestPulse = previousPulse

	// populate needed VFindCallResponse fields
	if test.VFindCall != nil {
		test.VFindCall.Callee = object
		test.VFindCall.Outgoing = outgoing
	}

	// populate needed VDelegatedCallResponse fields
	if test.VDelegatedCall != nil {
		if test.VDelegatedCallBadReference {
			test.VDelegatedCall.CallOutgoing = test.Server.RandomGlobalWithPulse()
		} else {
			test.VDelegatedCall.CallOutgoing = outgoing
		}
		test.VDelegatedCall.Callee = object
		test.VDelegatedCall.CallFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)
	}

	if test.VDelegatedRequestFinished != nil {
		test.VDelegatedRequestFinished.CallOutgoing = outgoing
		test.VDelegatedRequestFinished.Callee = object
		test.VDelegatedRequestFinished.CallFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)
	}

	if test.ExecutionExpected {
		requestResult := requestresult.New(ExecutionResultFromExecutor, gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte(""))

		executionMock := test.Runner.AddExecutionMock(outgoing.String())
		executionMock.AddStart(nil, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	{
		test.TypedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
			assert.Equal(t, test.ExpectedResult, result.ReturnArguments)
			return false
		})
		test.TypedChecker.VStateRequest.Set(func(request *payload.VStateRequest) bool {
			VStateReportDone := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
			server.SendPayload(ctx, &test.VState)
			testutils.WaitSignalsTimed(t, 10*time.Second, VStateReportDone)

			assert.Equal(t, object, request.Object)

			return false
		})
		test.TypedChecker.VFindCallRequest.Set(func(request *payload.VFindCallRequest) bool {
			if test.VFindCall == nil {
				t.Fatal("unreachable")
			}

			assert.Equal(t, previousPulse, request.LookAt)
			assert.Equal(t, object, request.Callee)
			assert.Equal(t, outgoing, request.Outgoing)

			test.VFindCall.LookedAt = request.LookAt
			server.SendPayload(test.Context, test.VFindCall)

			return false
		})
		test.TypedChecker.VDelegatedCallResponse.Set(func(response *payload.VDelegatedCallResponse) bool {
			return false
		})
	}

	test.Server.IncrementPulseAndWaitIdle(test.Context)

	if test.vStateSendBefore {
		VStateReportDone := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, &test.VState)
		testutils.WaitSignalsTimed(t, 10*time.Second, VStateReportDone)
	}

	if test.VDelegatedCall != nil {
		server.SendPayload(ctx, test.VDelegatedCall)
	}

	if test.VDelegatedRequestFinished != nil {
		server.SendPayload(ctx, test.VDelegatedRequestFinished)
	}

	{
		pl := payload.VCallRequest{
			CallType:       payload.CTConstructor,
			CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
			Callee:         class,
			CallSiteMethod: "New",
			CallOutgoing:   outgoing,
		}

		server.SendPayload(ctx, &pl)
	}

	{
		testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
		testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone(), foundError)
	}

	{
		assert.Equal(t, test.vStateRequestCount(), test.TypedChecker.VStateRequest.Count())
		assert.Equal(t, test.vCallResultCount(), test.TypedChecker.VCallResult.Count())
		assert.Equal(t, test.vFindCallCount(), test.TypedChecker.VFindCallRequest.Count())
		assert.Equal(t, test.vDelegatedCallResponseCount(), test.TypedChecker.VDelegatedCallResponse.Count())
	}
}

func TestDeduplication_DifferentPulses_MissingState(t *testing.T) {
	var tests []utils.TestRunner

	tests = append(tests, &DeduplicationDifferentPulsesCase{
		TestCase: utils.NewTestCase("empty object, no pending executions"),
		VState: payload.VStateReport{
			Status:              payload.Missing,
			OrderedPendingCount: 0,
		},
		VCallResultExpected: true,
		ExecutionExpected:   true,
		ExpectedResult:      ExecutionResultFromExecutor,
	})

	utils.Suite{Parallel: false, Cases: tests, TestRailID: "C5012"}.Run(t)
}

func TestDeduplication_DifferentPulses_EmptyState(t *testing.T) {
	var tests []utils.TestRunner

	errorFragmentFindCall := "(*SMExecute).stepProcessFindCallResponse"
	filterErrorFindCall := func(s string) bool {
		return !strings.Contains(s, errorFragmentFindCall)
	}

	{
		vStateReportEmptyOnePendingRequest := payload.VStateReport{
			Status:              payload.Empty,
			OrderedPendingCount: 1,
		}

		tests = append(tests,
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("empty object, 1 pending, missing call").WithErrorFilter(filterErrorFindCall),
				VState:   vStateReportEmptyOnePendingRequest,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.MissingCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragmentFindCall,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("empty object, 1 pending, unknown call").WithErrorFilter(filterErrorFindCall),
				VState:   vStateReportEmptyOnePendingRequest,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.UnknownCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragmentFindCall,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("empty object, 1 pending, known call wo result"),
				VState:   vStateReportEmptyOnePendingRequest,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: nil,
				},
				VCallResultExpected: false,
				ExecutionExpected:   false,
			},
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("empty object, 1 pending, known call w result"),
				VState:   vStateReportEmptyOnePendingRequest,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: &payload.VCallResult{ReturnArguments: ExecutionResultFromPreviousNode},
				},
				VCallResultExpected: true,
				ExpectedResult:      ExecutionResultFromPreviousNode,
				ExecutionExpected:   false,
			})
	}

	utils.Suite{Parallel: false, Cases: tests, TestRailID: "C5006"}.Run(t)
}

func TestDeduplication_DifferentPulses_EmptyState_WithDelegationToken(t *testing.T) {
	var tests []utils.TestRunner

	errorFragmentFindCall := "(*SMExecute).stepProcessFindCallResponse"
	filterErrorFindCall := func(s string) bool {
		return !strings.Contains(s, errorFragmentFindCall)
	}

	errorFragmentDeduplicate := "(*SMExecute).stepDeduplicate"
	filterErrorDeduplicate := func(s string) bool {
		return !strings.Contains(s, errorFragmentDeduplicate)
	}

	{
		vStateReportEmptyOnePendingRequest := payload.VStateReport{
			Status:              payload.Empty,
			OrderedPendingCount: 1,
		}

		tests = append(tests,
			&DeduplicationDifferentPulsesCase{
				TestCase:            utils.NewTestCase("empty object, 1 pending, with delegated request"),
				VState:              vStateReportEmptyOnePendingRequest,
				VDelegatedCall:      &payload.VDelegatedCallRequest{},
				VCallResultExpected: false,
				ExecutionExpected:   false,
			})

		tests = append(tests,
			&DeduplicationDifferentPulsesCase{
				TestCase:                  utils.NewTestCase("empty object, 1 pending, with finished delegated request, missing call").WithErrorFilter(filterErrorFindCall),
				VState:                    vStateReportEmptyOnePendingRequest,
				VDelegatedCall:            &payload.VDelegatedCallRequest{},
				VDelegatedRequestFinished: &payload.VDelegatedRequestFinished{},
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.MissingCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecuteShouldHaveError: errorFragmentFindCall,
				ExecutionExpected:      false,
			},
			&DeduplicationDifferentPulsesCase{
				TestCase:                  utils.NewTestCase("empty object, 1 pending, with finished delegated request, unknown call").WithErrorFilter(filterErrorFindCall),
				VState:                    vStateReportEmptyOnePendingRequest,
				VDelegatedCall:            &payload.VDelegatedCallRequest{},
				VDelegatedRequestFinished: &payload.VDelegatedRequestFinished{},
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.UnknownCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecuteShouldHaveError: errorFragmentFindCall,
				ExecutionExpected:      false,
			},
			&DeduplicationDifferentPulsesCase{
				TestCase:                  utils.NewTestCase("empty object, 1 pending, with finished delegated request, known call wo result").WithErrorFilter(filterErrorFindCall),
				VState:                    vStateReportEmptyOnePendingRequest,
				VDelegatedCall:            &payload.VDelegatedCallRequest{},
				VDelegatedRequestFinished: &payload.VDelegatedRequestFinished{},
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecuteShouldHaveError: errorFragmentFindCall,
				ExecutionExpected:      false,
			},
			&DeduplicationDifferentPulsesCase{
				TestCase:                  utils.NewTestCase("empty object, 1 pending, with finished delegated request, known call w result"),
				VState:                    vStateReportEmptyOnePendingRequest,
				VDelegatedCall:            &payload.VDelegatedCallRequest{},
				VDelegatedRequestFinished: &payload.VDelegatedRequestFinished{},
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: &payload.VCallResult{ReturnArguments: ExecutionResultFromPreviousNode},
				},
				VCallResultExpected: true,
				ExpectedResult:      ExecutionResultFromPreviousNode,
				ExecutionExpected:   false,
			})

		tests = append(tests,
			&DeduplicationDifferentPulsesCase{
				TestCase:                   utils.NewTestCase("empty object, 1 pending, with bad delegated request").WithErrorFilter(filterErrorDeduplicate),
				VState:                     vStateReportEmptyOnePendingRequest,
				VDelegatedCall:             &payload.VDelegatedCallRequest{},
				VDelegatedCallBadReference: true,
				VCallResultExpected:        false,
				ExecuteShouldHaveError:     errorFragmentDeduplicate,
				ExecutionExpected:          false,
			})
	}

	utils.Suite{Parallel: false, Cases: tests, TestRailID: "C5319"}.Run(t)
}

func TestDeduplication_DifferentPulses_ReadyState(t *testing.T) {
	var tests []utils.TestRunner

	errorFragment := "(*SMExecute).stepProcessFindCallResponse"
	filterError := func(s string) bool {
		return !strings.Contains(s, errorFragment)
	}

	{
		vStateReportReadyNoPendingRequests := payload.VStateReport{
			Status:              payload.Ready,
			OrderedPendingCount: 0,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{State: []byte("123")},
			},
		}

		tests = append(tests,
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("no pending requests, missing call").WithErrorFilter(filterError),
				VState:   vStateReportReadyNoPendingRequests,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.MissingCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragment,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("no pending requests, unknown call").WithErrorFilter(filterError),
				VState:   vStateReportReadyNoPendingRequests,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.UnknownCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragment,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("no pending requests, known call wo result").WithErrorFilter(filterError),
				VState:   vStateReportReadyNoPendingRequests,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragment,
			},
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("no pending requests, known call w result"),
				VState:   vStateReportReadyNoPendingRequests,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: &payload.VCallResult{ReturnArguments: ExecutionResultFromPreviousNode},
				},
				VCallResultExpected: true,
				ExecutionExpected:   false,
				ExpectedResult:      ExecutionResultFromPreviousNode,
			},
		)
	}

	{
		vStateReportReadyOnePendingRequest := payload.VStateReport{
			Status:              payload.Ready,
			OrderedPendingCount: 1,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{State: []byte("123")},
			},
		}

		tests = append(tests,
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("one pending request, missing call").WithErrorFilter(filterError),
				VState:   vStateReportReadyOnePendingRequest,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.MissingCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragment,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("one pending requests, unknown call").WithErrorFilter(filterError),
				VState:   vStateReportReadyOnePendingRequest,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.UnknownCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragment,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("one pending requests, known call wo result").WithErrorFilter(filterError),
				VState:   vStateReportReadyOnePendingRequest,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragment,
			},
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("one pending requests, known call w result"),
				VState:   vStateReportReadyOnePendingRequest,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: &payload.VCallResult{ReturnArguments: ExecutionResultFromPreviousNode},
				},
				VCallResultExpected: true,
				ExecutionExpected:   false,
				ExpectedResult:      ExecutionResultFromPreviousNode,
			},
		)
	}

	utils.Suite{Parallel: false, Cases: tests, TestRailID: "C5007"}.Run(t)
}

func TestDeduplication_DifferentPulses_InactiveState(t *testing.T) {
	var tests []utils.TestRunner

	{
		vStateReportInactive := payload.VStateReport{
			Status:              payload.Inactive,
			OrderedPendingCount: 0,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{State: []byte("123")},
			},
		}

		tests = append(tests,
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("missing call"),
				VState:   vStateReportInactive,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.MissingCall,
					CallResult: nil,
				},
				VCallResultExpected: false,
				ExecutionExpected:   false,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("unknown call"),
				VState:   vStateReportInactive,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.UnknownCall,
					CallResult: nil,
				},
				VCallResultExpected: false,
				ExecutionExpected:   false,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("known call wo result"),
				VState:   vStateReportInactive,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: nil,
				},
				VCallResultExpected: false,
				ExecutionExpected:   false,
			},
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("known call w result"),
				VState:   vStateReportInactive,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: &payload.VCallResult{ReturnArguments: ExecutionResultFromPreviousNode},
				},
				VCallResultExpected: true,
				ExecutionExpected:   false,
				ExpectedResult:      ExecutionResultFromPreviousNode,
			},
		)
	}

	utils.Suite{
		Parallel:   false,
		Cases:      tests,
		TestRailID: "C5008",
		Skipped:    "https://insolar.atlassian.net/browse/PLAT-416",
	}.Run(t)
}
