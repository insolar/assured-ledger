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
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
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

			syncChan := make(chan rms.Reference, 1)
			defer close(syncChan)

			suite.typedChecker.VStateReport.Set(func(rep *rms.VStateReport) bool {
				require.NotEmpty(t, rep.ProvidedContent.LatestDirtyState.Reference)
				syncChan <- rep.ProvidedContent.LatestDirtyState.Reference
				return false // no resend msg
			})

			suite.server.IncrementPulse(ctx)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, suite.typedChecker.VStateReport.Wait(ctx, 1))

			suite.typedChecker.VCachedMemoryResponse.Set(func(resp *rms.VCachedMemoryResponse) bool {
				assert.Equal(t, suite.object, resp.Object.GetValue())
				assert.Equal(t, []byte(newState), resp.Memory.GetBytes())
				return false
			})

			var stateRef rms.Reference

			select {
			case stateRef = <-syncChan:
			case <-time.After(10 * time.Second):
				require.FailNow(t, "timeout")
			}

			executeDone := suite.server.Journal.WaitStopOf(&handlers.SMVCachedMemoryRequest{}, 1)
			{
				cachReq := &rms.VCachedMemoryRequest{
					Object:  rms.NewReference(suite.object),
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

	Method_PrepareObject(ctx, s.server, rms.StateStatusReady, s.object, prevPulse)

	pl := utils.GenerateVCallRequestMethod(s.server)
	pl.Callee.Set(s.object)
	pl.CallSiteMethod = "ordered"

	newObjDescriptor := descriptor.NewObject(reference.Global{}, reference.Local{}, s.server.RandomGlobalWithPulse(), []byte("blabla"), false)
	result := requestresult.New([]byte("result"), s.object)
	result.SetAmend(newObjDescriptor, []byte(newState))

	key := pl.CallOutgoing.GetValue()
	s.runnerMock.AddExecutionMock(key).AddStart(nil, &execution.Update{
		Type:   execution.Done,
		Result: result,
	})
	s.runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
		Interference: pl.CallFlags.GetInterference(),
		State:        pl.CallFlags.GetState(),
	}, nil)

	s.typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
		assert.Equal(t, s.object, result.Callee.GetValue())
		assert.Equal(t, []byte("result"), result.ReturnArguments.GetBytes())
		return false
	})

	executeDone := s.server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	s.server.SendPayload(ctx, pl)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.server.Journal.WaitAllAsyncCallsDone())
}

func constructorPrecondition(s *memoryCacheTest, ctx context.Context, t *testing.T) {
	plWrapper := utils.GenerateVCallRequestConstructor(s.server)
	plWrapper.SetClass(s.class)
	pl := plWrapper.Get()

	s.object = plWrapper.GetObject()

	result := requestresult.New([]byte("result"), s.object)
	result.SetActivate(s.class, []byte(newState))

	key := plWrapper.GetOutgoing()
	s.runnerMock.AddExecutionMock(key).AddStart(nil, &execution.Update{
		Type:   execution.Done,
		Result: result,
	})
	s.runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
		Interference: pl.CallFlags.GetInterference(),
		State:        pl.CallFlags.GetState(),
	}, nil)

	s.typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
		assert.Equal(t, s.object, result.Callee.GetValue())
		assert.Equal(t, []byte("result"), result.ReturnArguments.GetBytes())
		return false
	})

	executeDone := s.server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	s.server.SendPayload(ctx, &pl)
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

	s.typedChecker = s.server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, s.mc, server)

	return ctx
}

func pendingPrecondition(s *memoryCacheTest, ctx context.Context, t *testing.T) {
	var (
		prevPulse = s.server.GetPulse().PulseNumber
		outgoing  = s.server.BuildRandomOutgoingWithPulse()
		incoming  = reference.NewRecordOf(s.object, outgoing.GetLocal())
	)

	s.server.IncrementPulse(ctx)

	report := utils.GenerateVStateReport(s.server, s.object, prevPulse)
	report.OrderedPendingCount = 1
	report.OrderedPendingEarliestPulse = prevPulse

	wait := s.server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	s.server.SendPayload(ctx, report)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)

	flags := rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)

	s.typedChecker.VDelegatedCallResponse.SetResend(false)

	{ // delegation request
		delegationReq := &rms.VDelegatedCallRequest{
			Callee:       rms.NewReference(s.object),
			CallFlags:    flags,
			CallOutgoing: rms.NewReference(outgoing),
			CallIncoming: rms.NewReference(incoming),
		}
		await := s.server.Journal.WaitStopOf(&handlers.SMVDelegatedCallRequest{}, 1)
		s.server.SendPayload(ctx, delegationReq)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.server.Journal.WaitAllAsyncCallsDone())
	}
	{ // send delegation request finished with new state
		pl := rms.VDelegatedRequestFinished{
			CallType:     rms.CallTypeMethod,
			Callee:       rms.NewReference(s.object),
			CallOutgoing: rms.NewReference(outgoing),
			CallIncoming: rms.NewReference(incoming),
			CallFlags:    flags,
			LatestState: &rms.ObjectState{
				Reference: rms.NewReferenceLocal(s.server.RandomLocalWithPulse()),
				Class:     rms.NewReference(s.class),
				Memory:    rms.NewBytes([]byte(newState)),
			},
		}
		await := s.server.Journal.WaitStopOf(&handlers.SMVDelegatedRequestFinished{}, 1)
		s.server.SendPayload(ctx, &pl)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.server.Journal.WaitAllAsyncCallsDone())

		assert.Equal(t, 1, s.typedChecker.VDelegatedCallResponse.Count())
	}
}
