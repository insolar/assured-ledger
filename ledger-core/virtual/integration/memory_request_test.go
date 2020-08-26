package integration

import (
	"context"
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
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

type TestStep func(s *memoryCachTest, ctx context.Context, t *testing.T)

const newState = "new state"

func TestVirtual_VCachedMemoryRequest(t *testing.T) {
	insrail.LogSkipCase(t, "", "https://insolar.atlassian.net/browse/PLAT-747")
	defer commonTestUtils.LeakTester(t)
	testCases := []struct {
		name         string
		precondition TestStep
	}{
		{name: "Object state created from constructor", precondition: constructorPrecondition},
		{name: "Object state created from method", precondition: methodPrecondition},
		// {name: "Object state created from pending", pending: true},
	}

	for _, cases := range testCases {
		t.Run(cases.name, func(t *testing.T) {

			suite := &memoryCachTest{}

			ctx := suite.initServer(t)

			cases.precondition(suite, ctx, t)

			var stateRef payload.Reference

			suite.server.IncrementPulse(ctx)
			suite.typedChecker.VStateReport.Set(func(rep *payload.VStateReport) bool {
				stateRef = rep.LatestValidatedState
				return false // no resend msg
			})

			suite.typedChecker.VCachedMemoryResponse.Set(func(resp *payload.VCachedMemoryResponse) bool {
				require.Equal(t, suite.object, resp.Object)
				require.Equal(t, newState, resp.Memory)
				return false
			})

			executeDone := suite.server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			{
				cachReq := &payload.VCachedMemoryRequest{
					Object:  suite.object,
					StateID: stateRef,
				}
				suite.server.SendPayload(ctx, cachReq)
			}
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, suite.server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, suite.typedChecker.VCachedMemoryResponse.Count())
			suite.mc.Finish()
		})
	}

}

func methodPrecondition(s *memoryCachTest, ctx context.Context, t *testing.T) {
	prevPulse := s.server.GetPulse().PulseNumber

	s.server.IncrementPulse(ctx)

	Method_PrepareObject(ctx, s.server, payload.StateStatusReady, s.object, prevPulse)

	pl := utils.GenerateVCallRequestMethod(s.server)
	pl.Callee = s.object
	pl.CallSiteMethod = "ordered"
	callOutgoing := pl.CallOutgoing

	result := requestresult.New([]byte(newState), s.object)

	key := callOutgoing.String()
	s.runnerMock.AddExecutionMock(key).
		AddStart(nil, &execution.Update{
			Type:   execution.Done,
			Result: result,
		})
	s.runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
		Interference: pl.CallFlags.GetInterference(),
		State:        pl.CallFlags.GetState(),
	}, nil)

	s.server.SendPayload(ctx, pl)
}

func constructorPrecondition(s *memoryCachTest, ctx context.Context, t *testing.T) {
	pl := utils.GenerateVCallRequestConstructor(s.server)
	pl.Caller = s.class
	callOutgoing := pl.CallOutgoing

	result := requestresult.New([]byte(newState), s.object)

	key := callOutgoing.String()
	s.runnerMock.AddExecutionMock(key).
		AddStart(nil, &execution.Update{
			Type:   execution.Done,
			Result: result,
		})
	s.runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
		Interference: pl.CallFlags.GetInterference(),
		State:        pl.CallFlags.GetState(),
	}, nil)

	s.server.SendPayload(ctx, pl)
}

type memoryCachTest struct {
	mc           *minimock.Controller
	server       *utils.Server
	runnerMock   *logicless.ServiceMock
	typedChecker *checker.Typed

	class  reference.Global
	caller reference.Global
	object reference.Global
}

func (s *memoryCachTest) initServer(t *testing.T) context.Context {

	s.mc = minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	s.server = server

	s.runnerMock = logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(s.runnerMock)

	server.Init(ctx)

	s.typedChecker = s.server.PublisherMock.SetTypedChecker(ctx, s.mc, server)

	return ctx
}

/*

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
	}*/
