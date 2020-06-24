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
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func calculateOutgoing(request payload.VCallRequest) reference.Global {
	return reference.NewRecordOf(request.Callee, request.CallOutgoing)
}

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
		outgoing  = server.RandomLocalWithPulse()
		objectRef = reference.NewSelf(outgoing)
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "test",
		CallOutgoing:   outgoing,
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, res.ReturnArguments, []byte("123"))
		require.Equal(t, res.Callee, objectRef)
		require.Equal(t, res.CallOutgoing, outgoing)

		return false // no resend msg
	})

	{
		requestResult := requestresult.New([]byte("123"), objectRef)
		requestResult.SetActivate(reference.Global{}, class, []byte("234"))

		runnerMock.AddExecutionMock(calculateOutgoing(pl).String()).
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
		outgoing  = server.RandomLocalWithPulse()
		objectRef = reference.NewSelf(outgoing)
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
		require.Equal(t, res.Callee, objectRef)
		require.Equal(t, res.CallOutgoing, outgoing)

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
	outgoing := server.RandomLocalWithPulse()
	objectRef := reference.NewSelf(outgoing)
	expectedError, err := foundation.MarshalMethodErrorResult(errors.New("bad class reference"))
	require.NoError(t, err)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		assert.Equal(t, res.Callee, objectRef)
		assert.Equal(t, res.CallOutgoing, outgoing)
		assert.Equal(t, expectedError, res.ReturnArguments)

		return false // no resend msg
	})

	// Call constructor on an empty class ref
	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}
	server.SendPayload(ctx, &pl)

	// Call constructor on a bad class ref
	pl.Callee = server.RandomGlobalWithPulse()
	server.SendPayload(ctx, &pl)

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
		outgoing     = server.RandomLocalWithPulse()
		objectRef    = reference.NewSelf(outgoing)
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
		require.Equal(t, res.ReturnArguments, runnerResult)
		require.Equal(t, res.Callee, objectRef)
		require.Equal(t, res.CallOutgoing, outgoing)
		require.Equal(t, payload.CTConstructor, res.CallType)
		require.Equal(t, flags, res.CallFlags)

		return false // no resend msg
	})

	{
		requestResult := requestresult.New(runnerResult, objectRef)
		requestResult.SetActivate(reference.Global{}, class, []byte("234"))

		runnerMock.AddExecutionMock(calculateOutgoing(pl).String()).
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
		outgoing  = server.RandomLocalWithPulse()
		objectRef = reference.NewSelf(outgoing)
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

		runnerMock.AddExecutionMock(calculateOutgoing(pl).String()).
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
		require.Equal(t, res.ReturnArguments, []byte("123"))
		require.Equal(t, res.Callee, objectRef)
		require.Equal(t, res.CallOutgoing, outgoing)

		return false // no resend msg
	})

	{
		pl := makeVStateReportWithState(objectRef, payload.Missing, nil)
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
		p1        = server.GetPulse().PulseNumber
		outgoing  = server.RandomLocalWithPulse()
		objectRef = reference.NewSelf(outgoing)
	)

	server.IncrementPulseAndWaitIdle(ctx)
	p2 := server.GetPulse().PulseNumber

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateRequest.Set(func(req *payload.VStateRequest) bool {
		require.Equal(t, p1, req.AsOf)
		require.Equal(t, objectRef, req.Object)

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
			Object: objectRef,
		}

		server.SendMessage(ctx, utils.NewRequestWrapper(p2, &report).SetSender(server.JetCoordinatorMock.Me()).Finalize())

		return false // no resend msg
	})
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, []byte("123"), res.ReturnArguments)
		require.Equal(t, objectRef, res.Callee)
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

		runnerMock.AddExecutionMock(calculateOutgoing(pl).String()).
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

		classA        = gen.UniqueGlobalRef()
		outgoingA     = server.RandomLocalWithPulse()
		objectAGlobal = reference.NewSelf(outgoingA)

		classB        = gen.UniqueGlobalRef()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingCallRef = gen.UniqueGlobalRef()
	)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallConstructor(classB, "New", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), objectAGlobal)
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
				require.Equal(t, objectAGlobal, ctx.Request.Caller)
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
			assert.Equal(t, objectAGlobal, request.Caller)
			assert.Equal(t, []byte("123"), request.Arguments)
			assert.Equal(t, payload.CTConstructor, request.CallType)
			assert.Equal(t, uint32(1), request.CallSequence)
			assert.Equal(t, outgoingCallRef, request.CallReason)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.Pulse())
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, payload.CTConstructor, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee {
			case objectAGlobal:
				require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
				require.Equal(t, server.GlobalCaller(), res.Caller)
				require.Equal(t, outgoingA, res.CallOutgoing)
			default:
				require.Equal(t, []byte("finish B.New"), res.ReturnArguments)
				require.Equal(t, objectAGlobal, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.Pulse())
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == objectAGlobal
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
