// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func TestDeduplication_SecondCallOfMethodDuringExecution(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C5095")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	oneExecutionEnded := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	helper := utils.NewHelper(server)

	p1 := server.GetPulse().PulseNumber
	server.IncrementPulseAndWaitIdle(ctx)

	outgoing := helper.BuildObjectOutgoing()
	class := gen.UniqueGlobalRef()
	object := gen.UniqueGlobalRef()

	report := &payload.VStateReport{
		Status: payload.Ready,
		AsOf:   p1,
		Object: object,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				State:       []byte("memory"),
				Deactivated: false,
			},
		},
	}
	server.SendPayload(ctx, report)

	releaseBlockedExecution := make(chan struct{}, 0)
	numberOfExecutions := 0
	{
		isolation := contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		runnerMock.AddExecutionClassify("SomeMethod", isolation, nil)

		newObjDescriptor := descriptor.NewObject(
			reference.Global{}, reference.Local{}, class, []byte(""),
		)

		requestResult := requestresult.New([]byte("call result"), gen.UniqueGlobalRef())
		requestResult.SetAmend(newObjDescriptor, []byte("new memory"))

		executionMock := runnerMock.AddExecutionMock("SomeMethod")
		executionMock.AddStart(func(ctx execution.Context) {
			numberOfExecutions++
			<-releaseBlockedExecution
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.SetResend(false).ExpectedCount(1)

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Callee:         object,
		CallSiteMethod: "SomeMethod",
		CallOutgoing:   outgoing,
	}
	server.SendPayload(ctx, &pl)
	server.SendPayload(ctx, &pl)

	testutils.WaitSignalsTimed(t, 10*time.Second, oneExecutionEnded)

	close(releaseBlockedExecution)

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, 1, numberOfExecutions)
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

func TestDeduplication_SecondCallOfMethodAfterExecution(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C5096")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	helper := utils.NewHelper(server)

	p1 := server.GetPulse().PulseNumber

	server.IncrementPulseAndWaitIdle(ctx)

	outgoing := helper.BuildObjectOutgoing()
	class := gen.UniqueGlobalRef()
	object := gen.UniqueGlobalRef()

	report := &payload.VStateReport{
		Status: payload.Ready,
		AsOf:   p1,
		Object: object,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				State:       []byte("memory"),
				Deactivated: false,
			},
		},
	}
	server.SendPayload(ctx, report)

	numberOfExecutions := 0
	{
		isolation := contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		runnerMock.AddExecutionClassify("SomeMethod", isolation, nil)

		newObjDescriptor := descriptor.NewObject(
			reference.Global{}, reference.Local{}, class, []byte(""),
		)

		requestResult := requestresult.New([]byte("call result"), gen.UniqueGlobalRef())
		requestResult.SetAmend(newObjDescriptor, []byte("new memory"))

		executionMock := runnerMock.AddExecutionMock("SomeMethod")
		executionMock.AddStart(func(ctx execution.Context) {
			numberOfExecutions++
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	var firstResult *payload.VCallResult

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		if firstResult != nil {
			require.Equal(t, firstResult, result)
		} else {
			firstResult = result
		}

		return false
	}).ExpectedCount(2)

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Callee:         object,
		CallSiteMethod: "SomeMethod",
		CallOutgoing:   outgoing,
	}

	server.SendPayload(ctx, &pl)
	server.PublisherMock.WaitCount(1, 10*time.Second)
	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	server.SendPayload(ctx, &pl)
	server.PublisherMock.WaitCount(2, 10*time.Second)
	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	assert.Equal(t, 1, numberOfExecutions)

	mc.Finish()
}

// test deduplication of method calls using VFindCallRequest
// and previous virtual executor

// we send single VCallRequest with method call and pulse of outgoing
// less than the current. Object may have pending execution that is either
// the request or some other one. Pending can be confirmed or not, finished
// or not.

// Depending on options we expect execution to happen or not and call result
// message to be sent or not.

type deduplicateMethodUsingPrevVETestInfo struct {
	name string

	// pendings
	pending             bool
	confirmPending      bool
	pendingIsTheRequest bool
	finishPending       bool

	// lookup
	expectFindRequestMessage bool // is VFindCallRequest expected
	findRequestStatus        payload.VFindCallResponse_CallState
	findRequestHasResult     bool

	// actions
	expectExecution     bool // should request be executed
	expectResultMessage bool // is VCallResult expected
}

func TestDeduplication_MethodUsingPrevVE(t *testing.T) {
	t.Log("C5097")

	table := []deduplicateMethodUsingPrevVETestInfo{
		{
			name: "no pendings, findCall message, missing call",

			expectFindRequestMessage: true,
			findRequestStatus:        payload.MissingCall,

			expectExecution:     true,
			expectResultMessage: true,
		},
		{
			name: "no pendings, findCall message, unknown call",

			expectFindRequestMessage: true,
			findRequestStatus:        payload.UnknownCall,

			expectExecution:     true,
			expectResultMessage: true,
		},
		{
			name: "no pendings, findCall message, found call, has result",

			expectFindRequestMessage: true,
			findRequestStatus:        payload.FoundCall,
			findRequestHasResult:     true,

			expectResultMessage: true,
		},

		{
			name: "pending is the request, no confirmation, findCall message, found call, no result",

			pending:             true,
			pendingIsTheRequest: true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.FoundCall,
		},
		{
			name: "pending is the request, no confirmation, findCall message, found call, has result",

			pending:             true,
			pendingIsTheRequest: true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.FoundCall,
			findRequestHasResult:     true,

			expectResultMessage: true,
		},
		{
			name: "pending is the request, confirmed, not finished, no findCall message",

			pending:             true,
			pendingIsTheRequest: true,
			confirmPending:      true,
		},
		{
			name: "pending is the request, confirmed, finished, findCall message, found call, has result",

			pending:             true,
			pendingIsTheRequest: true,
			confirmPending:      true,
			finishPending:       true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.FoundCall,
			findRequestHasResult:     true,

			expectResultMessage: true,
		},

		{
			name: "other pending, not confirmed, findCall message, missing",

			pending: true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.MissingCall,

			expectExecution:     true,
			expectResultMessage: true,
		},
		{
			name: "other pending, not confirmed, findCall message, unknown",

			pending: true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.UnknownCall,

			expectExecution:     true,
			expectResultMessage: true,
		},
		{
			name: "other pending, not confirmed, findCall message, found, has result",

			pending: true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.FoundCall,
			findRequestHasResult:     true,

			expectResultMessage: true,
		},

		{
			name: "other pending, confirmed, findCall message, missing",

			pending:        true,
			confirmPending: true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.MissingCall,

			expectExecution:     true,
			expectResultMessage: true,
		},
		{
			name: "other pending, confirmed, findCall message, unknown",

			pending:        true,
			confirmPending: true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.UnknownCall,

			expectExecution:     true,
			expectResultMessage: true,
		},
		{
			name: "other pending, confirmed, findCall message, found, has result",

			pending:        true,
			confirmPending: true,

			expectFindRequestMessage: true,
			findRequestStatus:        payload.FoundCall,
			findRequestHasResult:     true,

			expectResultMessage: true,
		},

		// not testing "other pending, confirmed, finished", should be equivalent to prev
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			suite := &deduplicateMethodUsingPrevVETest{}

			ctx := suite.initServer(t)

			executeDone := suite.server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			suite.switchPulse(ctx)
			suite.generateClass()
			suite.generateCaller()
			suite.generateObjectRef()
			suite.generateOutgoing()

			suite.setMessageCheckers(ctx, t, test)
			suite.setRunnerMock()

			if test.confirmPending {
				suite.confirmPending(ctx, test)
			}
			if test.finishPending {
				suite.finishPending(ctx)
			}

			request := payload.VCallRequest{
				CallType:       payload.CTMethod,
				CallFlags:      payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
				Caller:         suite.getCaller(),
				Callee:         suite.getObject(),
				CallSiteMethod: "SomeMethod",
				CallSequence:   1,
				CallOutgoing:   suite.getOutgoingLocal(),
			}
			suite.addPayloadAndWaitIdle(ctx, &request)

			testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, suite.server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, suite.typedChecker.VStateRequest.Count())
			if test.expectResultMessage {
				require.Equal(t, 1, suite.typedChecker.VCallResult.Count())
			}
			if test.expectFindRequestMessage {
				require.Equal(t, 1, suite.typedChecker.VFindCallRequest.Count())
			}

			if test.expectExecution {
				require.Equal(t, 1, suite.getNumberOfExecutions())
			} else {
				require.Equal(t, 0, suite.getNumberOfExecutions())
			}

			suite.finish()
		})
	}

}

type deduplicateMethodUsingPrevVETest struct {
	mu sync.RWMutex

	mc           *minimock.Controller
	server       *utils.Server
	runnerMock   *logicless.ServiceMock
	typedChecker *checker.Typed

	p1       pulse.Number
	class    reference.Global
	caller   reference.Global
	object   reference.Global
	outgoing reference.Global
	pending  reference.Global

	numberOfExecutions int
}

func (s *deduplicateMethodUsingPrevVETest) initServer(t *testing.T) context.Context {

	s.mc = minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	s.server = server

	s.runnerMock = logicless.NewServiceMock(ctx, t, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(s.runnerMock)

	server.Init(ctx)

	s.typedChecker = s.server.PublisherMock.SetTypedChecker(ctx, s.mc, server)

	return ctx
}

func (s *deduplicateMethodUsingPrevVETest) switchPulse(ctx context.Context) {
	s.p1 = s.server.GetPulse().PulseNumber
	s.server.IncrementPulseAndWaitIdle(ctx)
}

func (s *deduplicateMethodUsingPrevVETest) getP1() pulse.Number {
	return s.p1
}

func (s *deduplicateMethodUsingPrevVETest) generateCaller() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.caller = reference.NewSelf(gen.UniqueLocalRefWithPulse(s.p1))
}

func (s *deduplicateMethodUsingPrevVETest) generateObjectRef() {
	p := s.getP1()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.object = reference.NewSelf(gen.UniqueLocalRefWithPulse(p))
}

func (s *deduplicateMethodUsingPrevVETest) generateOutgoing() {
	p := s.getP1()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.outgoing = reference.NewRecordOf(s.caller, gen.UniqueLocalRefWithPulse(p))
}

func (s *deduplicateMethodUsingPrevVETest) generateClass() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.class = gen.UniqueGlobalRef()
}

func (s *deduplicateMethodUsingPrevVETest) getObject() reference.Global {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.object
}

func (s *deduplicateMethodUsingPrevVETest) getCaller() reference.Global {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.caller
}

func (s *deduplicateMethodUsingPrevVETest) getClass() reference.Global {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.class
}

func (s *deduplicateMethodUsingPrevVETest) getOutgoingRef() reference.Global {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return reference.NewRecordOf(s.getCaller(), s.outgoing.GetLocal())
}

func (s *deduplicateMethodUsingPrevVETest) getOutgoingLocal() reference.Global {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.outgoing
}

func (s *deduplicateMethodUsingPrevVETest) incNumberOfExecutions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.numberOfExecutions++
}

func (s *deduplicateMethodUsingPrevVETest) getNumberOfExecutions() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.numberOfExecutions
}

func (s *deduplicateMethodUsingPrevVETest) confirmPending(
	ctx context.Context, testInfo deduplicateMethodUsingPrevVETestInfo,
) {
	if testInfo.pendingIsTheRequest {
		s.pending = s.getOutgoingRef()
	} else {
		s.pending = reference.NewRecordOf(s.getObject(), gen.UniqueLocalRefWithPulse(s.getP1()))
	}

	pl := payload.VDelegatedCallRequest{
		Callee:       s.getObject(),
		CallOutgoing: s.pending,
		CallFlags:    payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
	}

	s.addPayloadAndWaitIdle(ctx, &pl)
}

func (s *deduplicateMethodUsingPrevVETest) finishPending(
	ctx context.Context,
) {
	pl := payload.VDelegatedRequestFinished{
		Callee:       s.getObject(),
		CallOutgoing: s.pending,
		CallFlags:    payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
	}
	s.addPayloadAndWaitIdle(ctx, &pl)
}

func (s *deduplicateMethodUsingPrevVETest) setMessageCheckers(
	ctx context.Context,
	t *testing.T,
	testInfo deduplicateMethodUsingPrevVETestInfo,
) {

	s.typedChecker.VStateRequest.Set(func(req *payload.VStateRequest) bool {
		require.Equal(t, s.getP1(), req.AsOf)
		require.Equal(t, s.getObject(), req.Object)

		report := payload.VStateReport{
			AsOf:   s.getP1(),
			Status: payload.Ready,
			Object: s.getObject(),

			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: gen.UniqueLocalRefWithPulse(s.getP1()),
					Class:     s.getClass(),
					State:     []byte("object memory"),
				},
			},
		}

		if testInfo.pending {
			report.UnorderedPendingCount = 1
			report.UnorderedPendingEarliestPulse = s.getP1()
		}

		s.server.SendPayload(ctx, &report)

		return false // no resend msg
	}).ExpectedCount(1)

	if testInfo.confirmPending {
		s.typedChecker.VDelegatedCallResponse.SetResend(false)
	}

	if testInfo.expectFindRequestMessage {
		s.typedChecker.VFindCallRequest.Set(func(req *payload.VFindCallRequest) bool {
			require.Equal(t, s.getP1(), req.LookAt)
			require.Equal(t, s.getObject(), req.Callee)
			require.Equal(t, s.getOutgoingRef(), req.Outgoing)

			response := payload.VFindCallResponse{
				LookedAt: s.getP1(),
				Callee:   s.getObject(),
				Outgoing: s.getOutgoingRef(),
				Status:   testInfo.findRequestStatus,
			}

			if testInfo.findRequestHasResult {
				response.CallResult = &payload.VCallResult{
					CallType:        payload.CTMethod,
					CallFlags:       payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
					Caller:          s.getCaller(),
					Callee:          s.getObject(),
					CallOutgoing:    s.getOutgoingLocal(),
					ReturnArguments: []byte("found request"),
				}
			}

			s.server.SendPayload(ctx, &response)
			return false
		}).ExpectedCount(1)
	}

	if testInfo.expectResultMessage {
		s.typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, payload.CTMethod, res.CallType)
			flags := payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
			require.Equal(t, flags, res.CallFlags)
			require.Equal(t, s.getCaller(), res.Caller)
			require.Equal(t, s.getObject(), res.Callee)
			require.Equal(t, s.getOutgoingLocal(), res.CallOutgoing)

			if testInfo.expectExecution {
				require.Equal(t, []byte("execution"), res.ReturnArguments)
			} else {
				require.Equal(t, []byte("found request"), res.ReturnArguments)
			}

			return false
		}).ExpectedCount(1)
	}
}

func (s *deduplicateMethodUsingPrevVETest) setRunnerMock() {
	isolation := contract.MethodIsolation{Interference: contract.CallIntolerable, State: contract.CallDirty}
	s.runnerMock.AddExecutionClassify("SomeMethod", isolation, nil)

	requestResult := requestresult.New([]byte("execution"), gen.UniqueGlobalRef())

	executionMock := s.runnerMock.AddExecutionMock("SomeMethod")
	executionMock.AddStart(func(ctx execution.Context) {
		s.incNumberOfExecutions()
	}, &execution.Update{
		Type:   execution.Done,
		Result: requestResult,
	})
}

func (s *deduplicateMethodUsingPrevVETest) addPayloadAndWaitIdle(
	ctx context.Context, pl payload.Marshaler,
) {
	s.server.SuspendConveyorAndWaitThenResetActive()
	s.server.SendPayload(ctx, pl)
	s.server.WaitActiveThenIdleConveyor()
}

func (s *deduplicateMethodUsingPrevVETest) finish() {
	s.server.Stop()
	s.mc.Finish()
}
