// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

type TestStep func(s *memoryCacheTest, ctx context.Context, t *testing.T)

const newState = "new state"

type memoryCacheTest struct {
	mc           *minimock.Controller
	server       *utils.Server
	runnerMock   *logicless.ServiceMock
	typedChecker *checker.Typed

	class  reference.Global
	object reference.Global
}

func TestVirtual_VCachedMemoryRequestHandler(t *testing.T) {
	insrail.LogCase(t, "C5681")
	defer commonTestUtils.LeakTester(t)
	var testCases = []struct {
		name         string
		precondition TestStep
	}{
		{name: "Object state created from constructor", precondition: constructorPrecondition},
		{name: "Object state created from method", precondition: methodPrecondition},
		{name: "Object state created from pending", precondition: pendingPrecondition},
	}
	for _, cases := range testCases {
		t.Run(cases.name, func(t *testing.T) {
			suite := &memoryCacheTest{}

			ctx := suite.initServer(t)
			defer suite.server.Stop()

			suite.object = suite.server.RandomGlobalWithPulse()
			suite.class = suite.server.RandomGlobalWithPulse()

			cases.precondition(suite, ctx, t)

			syncChan := make(chan payload.Reference, 1)
			defer close(syncChan)

			suite.typedChecker.VStateReport.Set(func(rep *payload.VStateReport) bool {
				syncChan <- rep.LatestValidatedState
				return false // no resend msg
			})

			suite.server.IncrementPulse(ctx)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, suite.typedChecker.VStateReport.Wait(ctx, 1))

			suite.typedChecker.VCachedMemoryResponse.Set(func(resp *payload.VCachedMemoryResponse) bool {
				require.Equal(t, suite.object, resp.Object)
				require.Equal(t, []byte(newState), resp.Memory)
				return false
			})

			var stateRef payload.Reference

			select {
			case stateRef = <-syncChan:
			case <-time.After(10 * time.Second):
				require.FailNow(t, "timeout")
			}

			executeDone := suite.server.Journal.WaitStopOf(&handlers.SMVCachedMemoryRequest{}, 1)
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

func methodPrecondition(s *memoryCacheTest, ctx context.Context, t *testing.T) {
	prevPulse := s.server.GetPulse().PulseNumber

	s.server.IncrementPulse(ctx)

	Method_PrepareObject(ctx, s.server, payload.StateStatusReady, s.object, prevPulse)

	pl := utils.GenerateVCallRequestMethod(s.server)
	pl.Callee = s.object
	pl.CallSiteMethod = "ordered"
	callOutgoing := pl.CallOutgoing

	newObjDescriptor := descriptor.NewObject(reference.Global{}, reference.Local{}, gen.UniqueGlobalRef(), []byte("blabla"), false)
	result := requestresult.New([]byte("result"), s.object)
	result.SetAmend(newObjDescriptor, []byte(newState))

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

	s.typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		assert.Equal(t, s.object, result.Callee)
		assert.Equal(t, []byte("result"), result.ReturnArguments)
		return false
	})

	executeDone := s.server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	s.server.SendPayload(ctx, pl)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.server.Journal.WaitAllAsyncCallsDone())
}

func constructorPrecondition(s *memoryCacheTest, ctx context.Context, t *testing.T) {
	pl := utils.GenerateVCallRequestConstructor(s.server)
	pl.Caller = s.class
	callOutgoing := pl.CallOutgoing
	s.object = reference.NewSelf(callOutgoing.GetLocal())

	result := requestresult.New([]byte("result"), s.object)
	result.SetActivate(reference.Global{}, s.class, []byte(newState))

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

	s.typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		assert.Equal(t, s.object, result.Callee)
		assert.Equal(t, []byte("result"), result.ReturnArguments)
		return false
	})

	executeDone := s.server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	s.server.SendPayload(ctx, pl)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.server.Journal.WaitAllAsyncCallsDone())
}

func (s *memoryCacheTest) initServer(t *testing.T) context.Context {
	s.mc = minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	s.server = server

	s.runnerMock = logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(s.runnerMock)

	server.Init(ctx)

	s.typedChecker = s.server.PublisherMock.SetTypedChecker(ctx, s.mc, server)

	return ctx
}

func pendingPrecondition(s *memoryCacheTest, ctx context.Context, t *testing.T) {
	prevPulse := s.server.GetPulse().PulseNumber
	outgoing := s.server.BuildRandomOutgoingWithPulse()
	incoming := reference.NewRecordOf(s.object, outgoing.GetLocal())

	s.server.IncrementPulse(ctx)

	report := utils.GenerateVStateReport(s.server, s.object, prevPulse)
	report.OrderedPendingCount = 1
	report.OrderedPendingEarliestPulse = prevPulse

	wait := s.server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	s.server.SendPayload(ctx, report)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)

	flags := payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)

	s.typedChecker.VDelegatedCallResponse.SetResend(false)

	{ // delegation request
		delegationReq := &payload.VDelegatedCallRequest{
			Callee:       s.object,
			CallFlags:    flags,
			CallOutgoing: outgoing,
			CallIncoming: incoming,
		}
		await := s.server.Journal.WaitStopOf(&handlers.SMVDelegatedCallRequest{}, 1)
		s.server.SendPayload(ctx, delegationReq)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.server.Journal.WaitAllAsyncCallsDone())
	}
	{ // send delegation request finished with new state
		pl := payload.VDelegatedRequestFinished{
			CallType:     payload.CallTypeMethod,
			Callee:       s.object,
			CallOutgoing: outgoing,
			CallIncoming: incoming,
			CallFlags:    flags,
			LatestState: &payload.ObjectState{
				State: []byte(newState),
			},
		}
		await := s.server.Journal.WaitStopOf(&handlers.SMVDelegatedRequestFinished{}, 1)
		s.server.SendPayload(ctx, &pl)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.server.Journal.WaitAllAsyncCallsDone())

		require.Equal(t, 1, s.typedChecker.VDelegatedCallResponse.Count())
	}
}
