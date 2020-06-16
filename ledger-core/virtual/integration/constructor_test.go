// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
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
	"github.com/insolar/assured-ledger/ledger-core/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func calculateOutgoing(request payload.VCallRequest) reference.Global {
	return reference.NewRecordOf(request.Callee, request.CallOutgoing)
}

func TestVirtual_Constructor_WithoutExecutor(t *testing.T) {
	t.Log("C4835")

	var (
		mc        = minimock.NewController(t)
		isolation = contract.ConstructorIsolation()
		class     = gen.UniqueReference()
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
	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

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
			AddStart(nil, &executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Result: requestResult,
			})
	}

	server.SendMessage(ctx, msg)

	assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))

	mc.Finish()
}

func TestVirtual_Constructor_WithExecutor(t *testing.T) {
	t.Log("C4835")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

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
	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, res.Callee, objectRef)
		require.Equal(t, res.CallOutgoing, outgoing)

		return false // no resend msg
	})

	server.SendMessage(ctx, msg)

	assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))

	mc.Finish()
}

func TestVirtual_Constructor_HasStateWithMissingStatus(t *testing.T) {
	t.Log("C4996")

	// VE has object's state record with Status==Missing
	// Constructor call should work on top of such entry
	var (
		mc        = minimock.NewController(t)
		isolation = contract.ConstructorIsolation()
		class     = gen.UniqueReference()
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		pulseNumber = server.GetPulse().PulseNumber
		outgoing    = server.RandomLocalWithPulse()
		objectRef   = reference.NewSelf(outgoing)
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
		requestResult := requestresult.New([]byte("123"), gen.UniqueReference())
		requestResult.SetActivate(gen.UniqueReference(), class, []byte("234"))

		runnerMock.AddExecutionMock(calculateOutgoing(pl).String()).
			AddStart(func(execution execution.Context) {
				require.Equal(t, "New", execution.Request.CallSiteMethod)
				require.Equal(t, []byte("arguments"), execution.Request.Arguments)
			}, &executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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
		msg := makeVStateReportWithState(pulseNumber, objectRef, payload.Missing, nil, server.JetCoordinatorMock.Me())
		server.SendMessage(ctx, msg)
	}

	msg := utils.NewRequestWrapper(pulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()
	server.SendMessage(ctx, msg)

	assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))

	mc.Finish()
}

func TestVirtual_Constructor_NoVFindCallRequestWhenMissing(t *testing.T) {
	// Constructor call with outgoing.Pulse < currentPulse
	// state request, state report
	t.Log("C4997")

	var (
		mc        = minimock.NewController(t)
		isolation = contract.ConstructorIsolation()
		class     = gen.UniqueReference()
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
		requestResult := requestresult.New([]byte("123"), gen.UniqueReference())
		requestResult.SetActivate(gen.UniqueReference(), class, []byte("234"))

		runnerMock.AddExecutionMock(calculateOutgoing(pl).String()).
			AddStart(func(execution execution.Context) {
				require.Equal(t, "New", execution.Request.CallSiteMethod)
				require.Equal(t, []byte("arguments"), execution.Request.Arguments)
			}, &executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.Callee.String()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		isolation = contract.ConstructorIsolation()

		classA        = gen.UniqueReference()
		outgoingA     = server.RandomLocalWithPulse()
		objectAGlobal = reference.NewSelf(outgoingA)

		classB        = gen.UniqueReference()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	outgoingCall := executionevent.NewRPCBuilder(gen.UniqueReference(), classA).CallConstructor(classB, "New", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock(classA.String())
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.New]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:     executionupdate.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.New]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
			Result: requestresult.New([]byte("finish A.New"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock(classB.String()).AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [B.New]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
			Result: requestresult.New([]byte("finish B.New"), objectBGlobal),
		},
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true).ExpectedCount(1)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		switch res.Callee {
		case classA:
			require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
		case classB:
			require.Equal(t, []byte("finish B.New"), res.ReturnArguments)
		}
		// we should resend that message only if it's CallResult from B to A
		return res.Caller == classA
	}).ExpectedCount(2)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         classA,
		CallSiteMethod: "New",
		CallOutgoing:   outgoingA,
	}
	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()
	beforeCount := server.PublisherMock.GetCount()
	server.SendMessage(ctx, msg)
	if !server.PublisherMock.WaitCount(beforeCount+3, 10*time.Second) {
		t.Fatal("failed to wait until all messages returned")
	}

	server.WaitActiveThenIdleConveyor()
	mc.Finish()
}
