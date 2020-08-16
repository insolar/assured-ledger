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
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestConstructor_SamePulse_WhileExecution(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4998")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		return false
	})

	synchronizeExecution := synchronization.NewPoint(1)

	executionFn := func(ctx execution.Context) {
		synchronizeExecution.Synchronize()
	}

	pl := utils.GenerateVCallRequestConstructor(server)

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), pl.Callee, []byte("234"))

		executionMock := runnerMock.AddExecutionMock(pl.CallOutgoing.String())
		executionMock.AddStart(executionFn, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	awaitStopFirstSM := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)
	awaitStopSecondSM := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	{
		// send first call request
		server.SendPayload(ctx, pl)
	}

	// await first SMExecute go to step execute (in this point machine is still not publish result to table in SMObject)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())

	{
		// send second call request
		server.SendPayload(ctx, pl)
	}

	// second SMExecute should stop in deduplication algorithm and she should not send result because she started during execution first machine
	commontestutils.WaitSignalsTimed(t, 10*time.Second, awaitStopSecondSM)

	// wakeup first SMExecute
	synchronizeExecution.WakeUp()

	commontestutils.WaitSignalsTimed(t, 10*time.Second, awaitStopFirstSM)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

func TestConstructor_SamePulse_AfterExecution(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5005")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		return false
	})

	pl := utils.GenerateVCallRequestConstructor(server)

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), pl.Callee, []byte("234"))

		executionMock := runnerMock.AddExecutionMock(pl.CallOutgoing.String())
		executionMock.AddStart(nil, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	awaitStopFirstSM := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	awaitStopSecondSM := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	{
		// send first call request
		server.SendPayload(ctx, pl)
	}

	// await first SMExecute go completed work (after complete SMExecute publish result to table in SMObject)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, awaitStopFirstSM)

	{
		// send second call request
		server.SendPayload(ctx, pl)
	}

	// second SMExecute should send result again because she started after first machine complete
	commontestutils.WaitSignalsTimed(t, 10*time.Second, awaitStopSecondSM)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

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
	defer commontestutils.LeakTester(t)

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
		ctx    = test.Context
		server = test.Server

		isolation     = contract.ConstructorIsolation()
		class         = server.RandomGlobalWithPulse()
		previousPulse = test.Server.GetPulse().PulseNumber
		outgoing      = test.Server.BuildRandomOutgoingWithPulse()
		object        = reference.NewSelf(outgoing.GetLocal())
		executeDone   = test.Server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		foundError    = synckit.ClosedChannel()
	)
	if test.ExecuteShouldHaveError != "" {
		foundError = test.Server.Journal.Wait(func(event debuglogger.UpdateEvent) bool {
			if event.Data.Error != nil {
				stack := throw.DeepestStackTraceOf(event.Data.Error)
				if stack == nil {
					return false
				}
				return strings.Contains(stack.StackTraceAsText(), test.ExecuteShouldHaveError)
			}
			return false
		})
	}

	// populate needed VStateReport fields
	test.VState.Object = object
	if test.VState.OrderedPendingCount > 0 {
		test.VState.OrderedPendingEarliestPulse = previousPulse
	}
	test.VState.AsOf = previousPulse

	// populate needed VFindCallResponse fields
	if test.VFindCall != nil {
		test.VFindCall.Callee = object
		if test.VFindCall.CallResult != nil {
			test.VFindCall.CallResult = utils.MakeMinimumValidVStateResult(server, ExecutionResultFromPreviousNode)
			test.VFindCall.CallResult.Callee = object
		}
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
		test.VDelegatedCall.CallIncoming = reference.NewRecordOf(class, test.VDelegatedCall.CallOutgoing.GetLocal())
	}

	if test.VDelegatedRequestFinished != nil {
		test.VDelegatedRequestFinished = &payload.VDelegatedRequestFinished{
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(isolation.Interference, isolation.State),
			Callee:       object,
			CallOutgoing: outgoing,
			CallIncoming: reference.NewRecordOf(class, outgoing.GetLocal()),
			LatestState: &payload.ObjectState{
				Class: class,
				State: ExecutionResultFromPreviousNode,
			},
		}
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
			commontestutils.WaitSignalsTimed(t, 10*time.Second, VStateReportDone)

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
		commontestutils.WaitSignalsTimed(t, 10*time.Second, VStateReportDone)
	}

	if test.VDelegatedCall != nil {
		server.SendPayload(ctx, test.VDelegatedCall)
	}

	if test.VDelegatedRequestFinished != nil {
		server.SendPayload(ctx, test.VDelegatedRequestFinished)
	}

	{
		pl := utils.GenerateVCallRequestConstructor(server)
		pl.Callee = class
		pl.CallOutgoing = outgoing

		server.SendPayload(ctx, pl)
	}

	{
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone(), foundError)
	}

	{
		assert.Equal(t, test.vStateRequestCount(), test.TypedChecker.VStateRequest.Count())
		assert.Equal(t, test.vCallResultCount(), test.TypedChecker.VCallResult.Count())
		assert.Equal(t, test.vFindCallCount(), test.TypedChecker.VFindCallRequest.Count())
		assert.Equal(t, test.vDelegatedCallResponseCount(), test.TypedChecker.VDelegatedCallResponse.Count())
	}
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
					CallResult: &payload.VCallResult{},
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
					CallResult: &payload.VCallResult{},
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
					CallResult: &payload.VCallResult{},
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
					CallResult: &payload.VCallResult{},
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
	errorFragmentDeduplicate := "(*SMExecute).stepProcessFindCallResponse"
	filterErrorDeduplicate := func(s string) bool {
		return !strings.Contains(s, errorFragmentDeduplicate)
	}

	{
		vStateReportInactive := payload.VStateReport{
			Status:              payload.Inactive,
			OrderedPendingCount: 0,
			ProvidedContent:     nil,
		}

		tests = append(tests,
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("missing call").WithErrorFilter(filterErrorDeduplicate),
				VState:   vStateReportInactive,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.MissingCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragmentDeduplicate,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("unknown call").WithErrorFilter(filterErrorDeduplicate),
				VState:   vStateReportInactive,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.UnknownCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragmentDeduplicate,
			},
			// expected panic of SM
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("known call wo result").WithErrorFilter(filterErrorDeduplicate),
				VState:   vStateReportInactive,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: nil,
				},
				VCallResultExpected:    false,
				ExecutionExpected:      false,
				ExecuteShouldHaveError: errorFragmentDeduplicate,
			},
			&DeduplicationDifferentPulsesCase{
				TestCase: utils.NewTestCase("known call w result"),
				VState:   vStateReportInactive,
				VFindCall: &payload.VFindCallResponse{
					Status:     payload.FoundCall,
					CallResult: &payload.VCallResult{},
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
	}.Run(t)
}
