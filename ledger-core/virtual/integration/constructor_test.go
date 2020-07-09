// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
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
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
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
		require.Equal(t, reference.NewSelf(outgoingRef.GetLocal()), res.Callee)
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
		require.Equal(t, reference.NewSelf(outgoing.GetLocal()), res.Callee)
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
	randomCallee := server.RandomGlobalWithPulse()

	leftMessages := map[reference.Global]struct{}{
		reference.NewSelf(outgoingOne.GetLocal()): struct{}{},
		reference.NewSelf(outgoingTwo.GetLocal()): struct{}{},
	}
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		if _, ok := leftMessages[res.Callee]; !ok {
			require.FailNow(t, "unexpected Callee")
		}
		ref := reference.NewSelf(res.CallOutgoing.GetLocal())

		require.Equal(t, res.Callee, ref)
		delete(leftMessages, ref)

		require.Equal(t, expectedError, res.ReturnArguments)

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
	require.Len(t, leftMessages, 0)

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
		require.Equal(t, reference.NewSelf(outgoing.GetLocal()), res.Callee)
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
		require.Equal(t, reference.NewSelf(outgoing.GetLocal()), res.Callee)
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
		require.Equal(t, reference.NewSelf(outgoing.GetLocal()), req.Object)

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
			Object: reference.NewSelf(outgoing.GetLocal()),
		}

		server.SendMessage(ctx, utils.NewRequestWrapper(p2, &report).SetSender(server.JetCoordinatorMock.Me()).Finalize())

		return false // no resend msg
	})
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, []byte("123"), res.ReturnArguments)
		require.Equal(t, reference.NewSelf(outgoing.GetLocal()), res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)

		return false // no resend msg
	})
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		expected := &payload.VStateReport{
			Status:           payload.Ready,
			AsOf:             p2,
			Object:           reference.NewSelf(outgoing.GetLocal()),
			LatestDirtyState: reference.NewSelf(outgoing.GetLocal()),
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					State: []byte("some memory"),
					Class: class,
				},
			},
		}
		expected.ProvidedContent.LatestDirtyState.Reference = report.ProvidedContent.LatestDirtyState.Reference

		assert.Equal(t, expected, report)

		return false
	})

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
		requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte("some memory"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(func(execution execution.Context) {
				require.Equal(t, "New", execution.Request.CallSiteMethod)
				require.Equal(t, []byte("arguments"), execution.Request.Arguments)
			}, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	server.SendPayload(ctx, &pl)
	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

	server.IncrementPulseAndWaitIdle(ctx)
	testutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))

	require.Equal(t, 1, typedChecker.VStateRequest.Count())
	require.Equal(t, 1, typedChecker.VCallResult.Count())
	require.Equal(t, 1, typedChecker.VStateReport.Count())

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
			case reference.NewSelf(outgoingA.GetLocal()):
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
		require.Equal(t, reference.NewSelf(outgoing.GetLocal()), res.Callee)
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

	authService := authentication.NewServiceMock(t)
	server.ReplaceAuthenticationService(authService)

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

	// add type checks
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			assert.Equal(t, reference.NewSelf(outgoing.GetLocal()), report.Object)
			assert.Equal(t, payload.Empty, report.Status)
			assert.Equal(t, int32(1), report.OrderedPendingCount)
			assert.Equal(t, constructorPulse, report.OrderedPendingEarliestPulse)
			assert.Equal(t, int32(0), report.UnorderedPendingCount)
			assert.Empty(t, report.UnorderedPendingEarliestPulse)
			assert.Empty(t, report.LatestDirtyState)
			assert.Empty(t, report.LatestValidatedState)

			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(msg *payload.VDelegatedCallRequest) bool {
			assert.Zero(t, msg.DelegationSpec)
			assert.Equal(t, reference.NewSelf(outgoing.GetLocal()), msg.Callee)
			assert.Equal(t, outgoing, msg.CallOutgoing)

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
			assert.Equal(t, outgoing, finished.CallOutgoing)
			assert.Equal(t, reference.NewSelf(outgoing.GetLocal()), finished.Callee)
			require.NotNil(t, finished.LatestState)
			assert.Equal(t, []byte("234"), finished.LatestState.State)
			assert.Equal(t, delegationToken, finished.DelegationSpec)
			return false
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, []byte("123"), res.ReturnArguments)
			assert.Equal(t, outgoing, res.CallOutgoing)
			assert.Equal(t, reference.NewSelf(outgoing.GetLocal()), res.Callee)
			assert.Equal(t, payload.CTConstructor, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)
			assert.Equal(t, delegationToken, res.DelegationSpec)
			return false
		})
	}

	synchronizeExecution := synchronization.NewPoint(1)

	// add executionMock
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

	// add authService mock functions
	{
		authService.HasToSendTokenMock.Set(func(token payload.CallDelegationToken) (b1 bool) {
			assert.Equal(t, reference.NewSelf(outgoing.GetLocal()), token.Callee)
			return true
		})
		authService.IsMessageFromVirtualLegitimateMock.Set(func(ctx context.Context, payloadObj interface{}, sender reference.Global, pr pulse.Range) (mustReject bool, err error) {
			assert.Equal(t, server.GlobalCaller(), sender)
			return false, nil
		})
		authService.GetCallDelegationTokenMock.Set(func(outgoingRef reference.Global, to reference.Global, pn pulse.Number, object reference.Global) (c1 payload.CallDelegationToken) {
			assert.Equal(t, reference.NewSelf(outgoing.GetLocal()), object)
			return payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				PulseNumber:       server.GetPulse().PulseNumber,
				Callee:            object,
				Outgoing:          outgoingRef,
				DelegateTo:        to,
				Approver:          server.GlobalCaller(),
			}
		})
	}

	server.SendPayload(ctx, &pl)

	testutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
	server.IncrementPulseAndWaitIdle(ctx)

	synchronizeExecution.WakeUp()

	synchronizeExecution.Done()
	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	msgVStateRequest := payload.VStateRequest{
		AsOf:   constructorPulse,
		Object: reference.NewSelf(outgoing.GetLocal()),
	}

	server.SendPayload(ctx, &msgVStateRequest)
	server.WaitActiveThenIdleConveyor()

	testutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 2))

	{
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedCallRequest.Count())
		assert.Equal(t, 0, typedChecker.VDelegatedCallResponse.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())
		assert.Equal(t, 2, typedChecker.VStateReport.Count())
	}

	mc.Finish()
}

func TestVirtual_Constructor_IsolationNegotiation(t *testing.T) {
	t.Log("C5031")
	table := []struct {
		name      string
		isolation contract.MethodIsolation
	}{
		{
			name:      "call constructor with intolerable and dirty flags",
			isolation: contract.MethodIsolation{Interference: contract.CallIntolerable, State: contract.CallDirty},
		},
		{
			name:      "call constructor with tolerable and validated flags",
			isolation: contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallValidated},
		},
		{
			name:      "call constructor with intolerable and validated flags",
			isolation: contract.MethodIsolation{Interference: contract.CallIntolerable, State: contract.CallValidated},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			outgoing := server.BuildRandomOutgoingWithPulse()

			expectedError := throw.W(throw.IllegalValue(), "failed to negotiate call isolation params", struct {
				methodIsolation contract.MethodIsolation
				callIsolation   contract.MethodIsolation
			}{
				methodIsolation: contract.ConstructorIsolation(),
				callIsolation:   test.isolation,
			})

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
				require.Equal(t, reference.NewSelf(outgoing.GetLocal()), result.Callee)
				require.Equal(t, outgoing, result.CallOutgoing)

				contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments)
				require.NoError(t, sysErr)
				require.NotNil(t, contractErr)
				require.Equal(t, expectedError.Error(), contractErr.Error())

				return false // no resend msg
			})

			pl := payload.VCallRequest{
				CallType:       payload.CTConstructor,
				CallFlags:      payload.BuildCallFlags(test.isolation.Interference, test.isolation.State),
				Callee:         testwallet.GetClass(),
				CallSiteMethod: "New",
				CallOutgoing:   outgoing,
			}
			server.SendPayload(ctx, &pl)

			testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}
