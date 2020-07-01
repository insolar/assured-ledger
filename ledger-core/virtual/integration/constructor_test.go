// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func TestVirtual_Constructor_WithoutExecutor(t *testing.T) {
	t.Log("C4835")

	var (
		mc        = minimock.NewController(t)
		isolation = contract.ConstructorIsolation()
		class     = gen.UniqueGlobalRef()
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		outgoingRef = server.BuildRandomOutgoingWithPulse()
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "test",
		CallOutgoing:   outgoingRef,
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, []byte("123"), res.ReturnArguments)
		require.Equal(t, outgoingRef, res.Callee)
		require.Equal(t, outgoingRef, res.CallOutgoing)

		return false // no resend msg
	})

	{
		requestResult := requestresult.New([]byte("123"), outgoingRef)
		requestResult.SetActivate(reference.Global{}, class, []byte("234"))

		runnerMock.AddExecutionMock(outgoingRef.String()).
			AddStart(nil, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	server.SendPayload(ctx, &pl)

	assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))

	mc.Finish()
}

func TestVirtual_Constructor_WithExecutor(t *testing.T) {
	t.Log("C5180")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	var (
		isolation = contract.ConstructorIsolation()
		outgoing  = server.BuildRandomOutgoingWithPulse()
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         testwallet.GetClass(),
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, outgoing, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)

		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
		require.NoError(t, sysErr)
		require.Nil(t, contractErr)

		return false // no resend msg
	})

	server.SendPayload(ctx, &pl)

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Constructor_BadClassRef(t *testing.T) {
	t.Log("C5030")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	isolation := contract.ConstructorIsolation()
	outgoingOne := server.BuildRandomOutgoingWithPulse()
	outgoingTwo := server.BuildRandomOutgoingWithPulse()
	expectedError, err := foundation.MarshalMethodErrorResult(errors.New("bad class reference"))
	require.NoError(t, err)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	called := 0
	randomCallee := server.RandomGlobalWithPulse()
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		if called == 0 {
			assert.Equal(t, outgoingOne, res.Callee)
			assert.Equal(t, outgoingOne, res.CallOutgoing)
			called++
		} else {
			assert.Equal(t, outgoingTwo, res.Callee)
			assert.Equal(t, outgoingTwo, res.CallOutgoing)
		}

		assert.Equal(t, expectedError, res.ReturnArguments)

		return false // no resend msg
	})

	// Call constructor on an empty class ref
	server.SendPayload(ctx, &payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		CallSiteMethod: "New",
		CallOutgoing:   outgoingOne,
		Arguments:      insolar.MustSerialize([]interface{}{}),
	})

	server.WaitActiveThenIdleConveyor()

	// Call constructor on a bad class ref
	server.SendPayload(ctx, &payload.VCallRequest{
		CallType:       payload.CTConstructor,
		Callee:         randomCallee,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		CallSiteMethod: "New",
		CallOutgoing:   outgoingTwo,
		Arguments:      insolar.MustSerialize([]interface{}{}),
	})

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Constructor_CurrentPulseWithoutObject(t *testing.T) {
	t.Log("C4995")

	var (
		mc        = minimock.NewController(t)
		isolation = contract.ConstructorIsolation()
		class     = gen.UniqueGlobalRef()
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		outgoing     = server.BuildRandomOutgoingWithPulse()
		runnerResult = []byte("123")
	)

	flags := payload.BuildCallFlags(isolation.Interference, isolation.State)
	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      flags,
		CallAsOf:       server.GetPulse().PulseNumber,
		Callee:         class,
		CallSiteMethod: "test",
		CallOutgoing:   outgoing,
	}
	msg := server.WrapPayload(&pl).Finalize()

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, runnerResult, res.ReturnArguments)
		require.Equal(t, outgoing, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)
		require.Equal(t, payload.CTConstructor, res.CallType)
		require.Equal(t, flags, res.CallFlags)

		return false // no resend msg
	})

	{
		requestResult := requestresult.New(runnerResult, outgoing)
		requestResult.SetActivate(reference.Global{}, class, []byte("234"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(nil, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	server.SendMessage(ctx, msg)

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Constructor_HasStateWithMissingStatus(t *testing.T) {
	t.Log("C4996")

	// VE has object's state record with Status==Missing
	// Constructor call should work on top of such entry
	var (
		mc        = minimock.NewController(t)
		isolation = contract.ConstructorIsolation()
		class     = gen.UniqueGlobalRef()
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		outgoing = server.BuildRandomOutgoingWithPulse()
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
		Arguments:      []byte("arguments"),
	}

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte("234"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(func(execution execution.Context) {
				require.Equal(t, "New", execution.Request.CallSiteMethod)
				require.Equal(t, []byte("arguments"), execution.Request.Arguments)
			}, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, []byte("123"), res.ReturnArguments)
		require.Equal(t, outgoing, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)

		return false // no resend msg
	})

	{
		pl := makeVStateReportWithState(outgoing, payload.Missing, nil)
		server.SendPayload(ctx, pl)
	}

	server.SendPayload(ctx, &pl)

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Constructor_NoVFindCallRequestWhenMissing(t *testing.T) {
	// Constructor call with outgoing.Pulse < currentPulse
	// state request, state report
	t.Log("C4997")

	var (
		mc        = minimock.NewController(t)
		isolation = contract.ConstructorIsolation()
		class     = gen.UniqueGlobalRef()
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		p1       = server.GetPulse().PulseNumber
		outgoing = server.BuildRandomOutgoingWithPulse()
	)

	server.IncrementPulseAndWaitIdle(ctx)
	p2 := server.GetPulse().PulseNumber

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateRequest.Set(func(req *payload.VStateRequest) bool {
		require.Equal(t, p1, req.AsOf)
		require.Equal(t, outgoing, req.Object)

		flags := payload.StateRequestContentFlags(0)
		flags.Set(
			payload.RequestLatestDirtyState,
			payload.RequestLatestValidatedState,
			payload.RequestOrderedQueue,
			payload.RequestUnorderedQueue,
		)
		require.Equal(t, flags, req.RequestedContent)

		report := payload.VStateReport{
			Status: payload.Missing,
			AsOf:   p1,
			Object: outgoing,
		}

		server.SendMessage(ctx, utils.NewRequestWrapper(p2, &report).SetSender(server.JetCoordinatorMock.Me()).Finalize())

		return false // no resend msg
	})
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, []byte("123"), res.ReturnArguments)
		require.Equal(t, outgoing, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)

		return false // no resend msg
	})

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
		Arguments:      []byte("arguments"),
	}
	msg := utils.NewRequestWrapper(p2, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte("234"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(func(execution execution.Context) {
				require.Equal(t, "New", execution.Request.CallSiteMethod)
				require.Equal(t, []byte("arguments"), execution.Request.Arguments)
			}, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	beforeCount := server.PublisherMock.GetCount()
	server.SendMessage(ctx, msg)
	if !server.PublisherMock.WaitCount(beforeCount+2, 10*time.Second) {
		t.Fatal("timeout waiting for messages on publisher")
	}

	server.WaitActiveThenIdleConveyor()
	require.Equal(t, 2, server.PublisherMock.GetCount())

	mc.Finish()
}

// A.New calls B.New
func TestVirtual_CallConstructorFromConstructor(t *testing.T) {
	t.Log("C5090")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.Callee.String()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		isolation = contract.ConstructorIsolation()
		callFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)

		classA    = gen.UniqueGlobalRef()
		outgoingA = server.BuildRandomOutgoingWithPulse()

		classB        = gen.UniqueGlobalRef()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingCallRef = gen.UniqueGlobalRef()
	)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoingCallRef, outgoingA).CallConstructor(classB, "New", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), outgoingA)
		objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))
		objectAExecutionMock := runnerMock.AddExecutionMock(classA.String())
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.New]")
				require.Equal(t, classA, ctx.Request.Callee)
				require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Error:    nil,
				Outgoing: outgoingCall,
			},
		)
		objectAExecutionMock.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.New]")
				require.Equal(t, []byte("finish B.New"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: objectAResult,
			},
		)

		objectBResult := requestresult.New([]byte("finish B.New"), objectBGlobal)
		objectBResult.SetActivate(reference.Global{}, classB, []byte("state B"))
		runnerMock.AddExecutionMock(classB.String()).AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B.New]")
				require.Equal(t, classB, ctx.Request.Callee)
				require.Equal(t, outgoingA, ctx.Request.Caller)
				require.Equal(t, []byte("123"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: objectBResult,
			},
		)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, classB, request.Callee)
			assert.Equal(t, outgoingA, request.Caller)
			assert.Equal(t, []byte("123"), request.Arguments)
			assert.Equal(t, payload.CTConstructor, request.CallType)
			assert.Equal(t, uint32(1), request.CallSequence)
			assert.Equal(t, outgoingCallRef, request.CallReason)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetLocal().Pulse())
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, payload.CTConstructor, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee {
			case outgoingA:
				require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
				require.Equal(t, server.GlobalCaller(), res.Caller)
				require.Equal(t, outgoingA, res.CallOutgoing)
			default:
				require.Equal(t, []byte("finish B.New"), res.ReturnArguments)
				require.Equal(t, outgoingA, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.GetLocal().Pulse())
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == outgoingA
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      callFlags,
		Caller:         server.GlobalCaller(),
		Callee:         classA,
		CallSiteMethod: "New",
		CallOutgoing:   outgoingA,
	}
	msg := server.WrapPayload(&pl).Finalize()
	server.SendMessage(ctx, msg)

	// wait for all calls and SMs
	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Constructor_WrongConstructorName(t *testing.T) {
	t.Log("C4977")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	var (
		isolation = contract.ConstructorIsolation()
		outgoing  = server.BuildRandomOutgoingWithPulse()
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         testwallet.GetClass(),
		CallSiteMethod: "NotExistingConstructorName",
		CallOutgoing:   outgoing,
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, outgoing, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)

		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
		require.Equal(t, &foundation.Error{"failed to execute request;\texecution error;\tfailed to find contracts constructor"}, contractErr)
		require.NoError(t, sysErr)

		return false // no resend msg
	})

	server.SendPayload(ctx, &pl)

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Constructor_PulseChangedWhileOutgoing(t *testing.T) {
	t.Log("C5085")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		isolation = contract.ConstructorIsolation()
		callFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)

		class    = gen.UniqueGlobalRef()
		outgoing = server.BuildRandomOutgoingWithPulse()

		constructorPulse = server.GetPulse().PulseNumber

		delegationToken payload.CallDelegationToken
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      callFlags,
		CallAsOf:       constructorPulse,
		Callee:         class,
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, outgoing, report.Object)
		assert.Equal(t, payload.Empty, report.Status)
		assert.Equal(t, int32(1), report.OrderedPendingCount)
		assert.Equal(t, int32(0), report.UnorderedPendingCount)
		return false
	})
	typedChecker.VDelegatedCallRequest.Set(func(msg *payload.VDelegatedCallRequest) bool {
		delegationToken = server.DelegationToken(msg.CallOutgoing, server.GlobalCaller(), msg.Callee)
		server.SendPayload(ctx, &payload.VDelegatedCallResponse{
			Callee:                 msg.Callee,
			CallIncoming:           msg.CallIncoming,
			ResponseDelegationSpec: delegationToken,
		})
		return false
	})
	typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
		assert.Equal(t, payload.CTConstructor, finished.CallType)
		assert.Equal(t, callFlags, finished.CallFlags)
		assert.NotNil(t, finished.LatestState)
		assert.Equal(t, outgoing, finished.CallOutgoing)
		assert.Equal(t, []byte("234"), finished.LatestState.State)
		return false
	})
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		assert.Equal(t, []byte("123"), res.ReturnArguments)
		assert.Equal(t, outgoing, res.Callee)
		assert.Equal(t, outgoing, res.CallOutgoing)
		assert.Equal(t, payload.CTConstructor, res.CallType)
		assert.Equal(t, callFlags, res.CallFlags)
		return false
	})

	synchronizeExecution := synchronization.NewPoint(1)

	{
		requestResult := requestresult.New([]byte("123"), outgoing)
		requestResult.SetActivate(reference.Global{}, class, []byte("234"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(func(ctx execution.Context) {
				synchronizeExecution.Synchronize()
			}, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	server.SendPayload(ctx, &pl)

	testutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
	server.IncrementPulseAndWaitIdle(ctx)

	synchronizeExecution.WakeUp()

	msgVStateRequest := payload.VStateRequest{
		AsOf:   constructorPulse,
		Object: outgoing,
	}

	server.SendPayload(ctx, &msgVStateRequest)
	server.WaitActiveThenIdleConveyor()

	synchronizeExecution.Done()
	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedCallRequest.Count())
		assert.Equal(t, 0, typedChecker.VDelegatedCallResponse.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())
		assert.Equal(t, 2, typedChecker.VStateReport.Count())
	}

	mc.Finish()
}
