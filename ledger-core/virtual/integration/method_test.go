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

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

const initialBalance uint32 = 500

func Method_PrepareObject(ctx context.Context, server *utils.Server, state payload.VStateReport_StateStatus, object reference.Global) {
	var (
		walletState = makeRawWalletState(initialBalance)

		content *payload.VStateReport_ProvidedContentBody
	)

	switch state {
	case payload.Missing:
		content = nil
	case payload.Ready:
		content = &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: reference.Local{},
				Class:     testwalletProxy.GetClass(),
				State:     walletState,
			},
		}
	default:
		panic("unexpected state")
	}

	payload := &payload.VStateReport{
		Status:                        state,
		Object:                        object,
		UnorderedPendingEarliestPulse: pulse.OfNow(),
		ProvidedContent:               content,
	}

	server.WaitIdleConveyor()
	server.SendPayload(ctx, payload)
	server.WaitActiveThenIdleConveyor()
}

func TestVirtual_BadMethod_WithExecutor(t *testing.T) {
	t.Log("C4976")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		outgoing     = server.RandomLocalWithPulse()
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	expectedError, err := foundation.MarshalMethodErrorResult(
		throw.W(throw.E("failed to find contracts method"), "failed to classify method"))

	require.NoError(t, err)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		assert.Equal(t, res.Callee, objectGlobal)
		assert.Equal(t, res.CallOutgoing, outgoing)
		assert.Equal(t, expectedError, res.ReturnArguments)

		return false // no resend msg
	})

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: class,
			CallSiteMethod:      "random",
			CallOutgoing:        outgoing,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}

		server.SendPayload(ctx, &pl)
	}

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Method_WithExecutor(t *testing.T) {
	t.Log("C4923")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool { return false })

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
		CallAsOf:            0,
		Caller:              server.GlobalCaller(),
		Callee:              objectGlobal,
		CallSiteDeclaration: class,
		CallSiteMethod:      "GetBalance",
		CallRequestFlags:    0,
		CallOutgoing:        server.RandomLocalWithPulse(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	countBefore := server.PublisherMock.GetCount()
	server.SendPayload(ctx, &pl)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Error("failed to wait for result")
	}

	mc.Finish()
}

func TestVirtual_Method_WithExecutor_ObjectIsNotExist(t *testing.T) {
	t.Log("C4974")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: testwallet.GetClass(),
			CallSiteMethod:      "GetBalance",
			CallOutgoing:        server.RandomLocalWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		server.SendPayload(ctx, &pl)

		// TODO fix it after implementation https://insolar.atlassian.net/browse/PLAT-395
	}
}

func TestVirtual_Method_WithoutExecutor_Unordered(t *testing.T) {
	t.Log("C4930")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		waitInputChannel  = make(chan struct{}, 2)
		waitOutputChannel = make(chan struct{}, 0)

		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	checkExecution := func(_ execution.Context) {
		// tell the test that we know about next request
		waitInputChannel <- struct{}{}

		// wait the test result
		<-waitOutputChannel
	}

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	{
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, res.ReturnArguments, []byte("345"))
			require.Equal(t, res.Callee, objectGlobal)

			return false // no resend msg
		})

		countBefore := server.PublisherMock.GetCount()

		for i := 0; i < 2; i++ {
			callOutgoing := server.RandomLocalWithPulse()

			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
				CallAsOf:            0,
				Caller:              server.GlobalCaller(),
				Callee:              objectGlobal,
				CallSiteDeclaration: class,
				CallSiteMethod:      "GetBalance",
				CallOutgoing:        callOutgoing,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			}

			result := requestresult.New([]byte("345"), objectGlobal)

			key := calculateOutgoing(pl).String()
			runnerMock.AddExecutionMock(key).
				AddStart(checkExecution, &execution.Update{
					Type:   execution.Done,
					Result: result,
				})
			runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallValidated,
			}, nil)

			server.SendPayload(ctx, &pl)
		}

		for i := 0; i < 2; i++ {
			select {
			case <-waitInputChannel:
			case <-time.After(10 * time.Second):
				require.Failf(t, "", "timeout")
			}
		}

		for i := 0; i < 2; i++ {
			waitOutputChannel <- struct{}{}
		}

		if !server.PublisherMock.WaitCount(countBefore+2, 10*time.Second) {
			t.Error("failed to wait for result")
		}
	}

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

	{
		assert.Equal(t, 2, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

func TestVirtual_CallMethodAfterPulseChange(t *testing.T) {
	t.Log("C4870")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true)
	typedChecker.VCallResult.SetResend(true)
	typedChecker.VStateReport.SetResend(true)

	server.IncrementPulseAndWaitIdle(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	// Change pulse to force send VStateRequest
	server.IncrementPulseAndWaitIdle(ctx)

	checkBalance(ctx, t, server, objectGlobal, initialBalance)

	{
		assert.Equal(t, 1, typedChecker.VCallRequest.Count())
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
		assert.Equal(t, 1, typedChecker.VStateReport.Count())
	}

	mc.Finish()
}

// ordered A.Foo calls ordered B.Bar
func TestVirtual_CallContractFromContract_Ordered(t *testing.T) {
	t.Log("C5086")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class = gen.UniqueGlobalRef()

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		flags = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	outgoingCall := execution.NewRPCBuilder(gen.UniqueGlobalRef(), objectAGlobal).CallMethod(objectBGlobal, class, "Bar", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			require.Equal(t, objectAGlobal, ctx.Request.Callee)
			t.Log("ExecutionStart [A.Foo]")
		},
		&execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.Foo]")
		},
		&execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock("Bar").AddStart(
		func(ctx execution.Context) {
			require.Equal(t, objectBGlobal, ctx.Request.Callee)
			t.Log("ExecutionStart [B.Bar]")
		},
		&execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish B.Bar"), objectBGlobal),
		},
	)

	runnerMock.AddExecutionClassify("Foo", flags, nil)
	runnerMock.AddExecutionClassify("Bar", flags, nil)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true).ExpectedCount(1)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		switch res.Callee {
		case objectAGlobal:
			require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
		case objectBGlobal:
			require.Equal(t, []byte("finish B.Bar"), res.ReturnArguments)
		}
		// we should resend that message only if it's CallResult from B to A
		return res.Caller == objectAGlobal
	}).ExpectedCount(2)

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Caller:              server.GlobalCaller(),
		Callee:              objectAGlobal,
		CallSiteDeclaration: class,
		CallSiteMethod:      "Foo",
		CallOutgoing:        server.RandomLocalWithPulse(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	beforeCount := server.PublisherMock.GetCount()
	server.SendPayload(ctx, &pl)
	if !server.PublisherMock.WaitCount(beforeCount+3, 10*time.Second) {
		t.Fatal("failed to wait until all messages returned")
	}

	server.WaitActiveThenIdleConveyor()
	mc.Finish()
}

// ordered A.Foo calls unordered B.Bar
func TestVirtual_CallContractFromContract_Unordered(t *testing.T) {
	t.Log("C5087")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class = gen.UniqueGlobalRef()

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	outgoingCall := execution.NewRPCBuilder(gen.UniqueGlobalRef(), objectAGlobal).CallMethod(objectBGlobal, class, "Bar", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			require.Equal(t, objectAGlobal, ctx.Request.Callee)
			t.Log("ExecutionStart [A.Foo]")
		},
		&execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.Foo]")
		},
		&execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock("Bar").AddStart(
		func(ctx execution.Context) {
			require.Equal(t, objectBGlobal, ctx.Request.Callee)
			t.Log("ExecutionStart [B.Bar]")
		},
		&execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish B.Bar"), objectBGlobal),
		},
	)

	runnerMock.AddExecutionClassify("Foo", contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}, nil)
	runnerMock.AddExecutionClassify("Bar", contract.MethodIsolation{Interference: contract.CallIntolerable, State: contract.CallDirty}, nil)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true).ExpectedCount(1)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		switch res.Callee {
		case objectAGlobal:
			require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
		case objectBGlobal:
			require.Equal(t, []byte("finish B.Bar"), res.ReturnArguments)
		}
		// we should resend that message only if it's CallResult from B to A
		return res.Caller == objectAGlobal
	}).ExpectedCount(2)

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Caller:              server.GlobalCaller(),
		Callee:              objectAGlobal,
		CallSiteDeclaration: class,
		CallSiteMethod:      "Foo",
		CallOutgoing:        server.RandomLocalWithPulse(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	beforeCount := server.PublisherMock.GetCount()
	server.SendPayload(ctx, &pl)
	if !server.PublisherMock.WaitCount(beforeCount+3, 10*time.Second) {
		t.Fatal("failed to wait until all messages returned")
	}

	server.WaitActiveThenIdleConveyor()
	mc.Finish()
}

// ordered A.Foo calls unordered A.Bar
func TestVirtual_Call_UnorderedMethod_From_OrderedMethod(t *testing.T) {
	t.Log("C5116")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class         = gen.UniqueGlobalRef()
		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

	outgoingCall := execution.NewRPCBuilder(gen.UniqueGlobalRef(), objectAGlobal).CallMethod(objectAGlobal, class, "Bar", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.Foo]")
		},
		&execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.Foo]")
		},
		&execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock("Bar").AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.Bar]")
		},
		&execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Bar"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionClassify("Foo", contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}, nil)
	runnerMock.AddExecutionClassify("Bar", contract.MethodIsolation{Interference: contract.CallIntolerable, State: contract.CallDirty}, nil)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true).ExpectedCount(1)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		switch res.Caller {
		case objectAGlobal:
			require.Equal(t, []byte("finish A.Bar"), res.ReturnArguments)
		default:
			require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
		}
		// we should resend that message only if it's CallResult from A to A
		return res.Caller == objectAGlobal
	}).ExpectedCount(2)

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Caller:              server.GlobalCaller(),
		Callee:              objectAGlobal,
		CallSiteDeclaration: class,
		CallSiteMethod:      "Foo",
		CallOutgoing:        server.RandomLocalWithPulse(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	beforeCount := server.PublisherMock.GetCount()
	server.SendPayload(ctx, &pl)
	if !server.PublisherMock.WaitCount(beforeCount+3, 10*time.Second) {
		t.Fatal("failed to wait until all messages returned")
	}

	server.WaitActiveThenIdleConveyor()
	mc.Finish()
}

// unordered A.Foo calls unordered A.Bar
func TestVirtual_Call_UnorderedMethod_From_UnorderedMethod(t *testing.T) {
	t.Log("C5122")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		flags         = contract.MethodIsolation{Interference: contract.CallIntolerable, State: contract.CallDirty}
		class         = gen.UniqueGlobalRef()
		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

	outgoingCall := execution.NewRPCBuilder(gen.UniqueGlobalRef(), objectAGlobal).CallMethod(objectAGlobal, class, "Bar", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.Foo]")
		},
		&execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.Foo]")
		},
		&execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock("Bar").AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.Bar]")
		},
		&execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Bar"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionClassify("Foo", flags, nil)
	runnerMock.AddExecutionClassify("Bar", flags, nil)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true).ExpectedCount(1)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		switch res.Caller {
		case objectAGlobal:
			require.Equal(t, []byte("finish A.Bar"), res.ReturnArguments)
		default:
			require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
		}
		// we should resend that message only if it's CallResult from A to A
		return res.Caller == objectAGlobal
	}).ExpectedCount(2)

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
		Caller:              server.GlobalCaller(),
		Callee:              objectAGlobal,
		CallSiteDeclaration: class,
		CallSiteMethod:      "Foo",
		CallOutgoing:        server.RandomLocalWithPulse(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	beforeCount := server.PublisherMock.GetCount()
	server.SendPayload(ctx, &pl)
	if !server.PublisherMock.WaitCount(beforeCount+3, 10*time.Second) {
		t.Fatal("failed to wait until all messages returned")
	}

	server.WaitActiveThenIdleConveyor()
	mc.Finish()
}

// A.New calls ordered B.Foo
func TestVirtual_CallMethodFromConstructor_Ordered(t *testing.T) {
	t.Log("C5091")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

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

	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectBGlobal, classB, "Foo", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), objectAGlobal)
		objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))
		objectAExecutionMock := runnerMock.AddExecutionMock("New")
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
				require.Equal(t, []byte("finish B.Foo"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: objectAResult,
			},
		)

		runnerMock.AddExecutionMock("Foo").AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B.Foo]")
				require.Equal(t, objectBGlobal, ctx.Request.Callee)
				require.Equal(t, objectAGlobal, ctx.Request.Caller)
				require.Equal(t, []byte("123"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish B.Foo"), objectBGlobal),
			},
		)

		runnerMock.AddExecutionClassify("Foo", contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, objectBGlobal, request.Callee)
			assert.Equal(t, objectAGlobal, request.Caller)
			assert.Equal(t, []byte("123"), request.Arguments)
			assert.Equal(t, payload.CTMethod, request.CallType)
			assert.Equal(t, uint32(1), request.CallSequence)
			assert.Equal(t, outgoingCallRef, request.CallReason)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.Pulse())
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee {
			case objectAGlobal:
				require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
				require.Equal(t, payload.CTConstructor, res.CallType)
				require.Equal(t, server.GlobalCaller(), res.Caller)
				require.Equal(t, outgoingA, res.CallOutgoing)
			case objectBGlobal:
				require.Equal(t, []byte("finish B.Foo"), res.ReturnArguments)
				require.Equal(t, payload.CTMethod, res.CallType)
				require.Equal(t, objectAGlobal, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.Pulse())

			default:
				t.Fatalf("wrong Callee")
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

// A.New calls unordered B.Foo
func TestVirtual_CallMethodFromConstructor_Unordered(t *testing.T) {
	t.Log("C5092")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

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

	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectBGlobal, classB, "Foo", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), objectAGlobal)
		objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))
		objectAExecutionMock := runnerMock.AddExecutionMock("New")
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
				require.Equal(t, []byte("finish B.Foo"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: objectAResult,
			},
		)

		runnerMock.AddExecutionMock("Foo").AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B.Foo]")
				require.Equal(t, objectBGlobal, ctx.Request.Callee)
				require.Equal(t, objectAGlobal, ctx.Request.Caller)
				require.Equal(t, []byte("123"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish B.Foo"), objectBGlobal),
			},
		)

		runnerMock.AddExecutionClassify("Foo", contract.MethodIsolation{Interference: contract.CallIntolerable, State: contract.CallDirty}, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, objectBGlobal, request.Callee)
			assert.Equal(t, objectAGlobal, request.Caller)
			assert.Equal(t, []byte("123"), request.Arguments)
			assert.Equal(t, payload.CTMethod, request.CallType)
			assert.Equal(t, uint32(1), request.CallSequence)
			assert.Equal(t, outgoingCallRef, request.CallReason)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.Pulse())
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee {
			case objectAGlobal:
				require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
				require.Equal(t, payload.CTConstructor, res.CallType)
				require.Equal(t, server.GlobalCaller(), res.Caller)
				require.Equal(t, outgoingA, res.CallOutgoing)
			case objectBGlobal:
				require.Equal(t, []byte("finish B.Foo"), res.ReturnArguments)
				require.Equal(t, payload.CTMethod, res.CallType)
				require.Equal(t, objectAGlobal, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.Pulse())

			default:
				t.Fatalf("wrong Callee")
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
}

// unordered A.Foo sends ordered outgoing and receives error
func TestVirtual_CallContractFromContract_InterferenceViolation(t *testing.T) {
	t.Log("C4980")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)
	var (
		class = gen.UniqueGlobalRef()

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		flags = contract.MethodIsolation{
			Interference: contract.CallIntolerable,
			State:        contract.CallDirty,
		}
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

	outgoingCallRef := gen.UniqueGlobalRef()

	expectedError, err := foundation.MarshalMethodErrorResult(throw.E("interference violation: ordered call from unordered call"))
	require.NoError(t, err)

	expectedResult := []byte("finish A.Foo")

	outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectAGlobal, class, "Bar", []byte("123")).SetInterference(contract.CallTolerable)
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			assert.Equal(t, objectAGlobal, ctx.Object)
			assert.Equal(t, flags, ctx.Isolation)
			t.Log("ExecutionStart [A.Foo]")
		},
		&execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	).AddContinue(func(result []byte) {
		assert.Equal(t, expectedError, result)

	}, &execution.Update{
		Type:   execution.Done,
		Result: requestresult.New(expectedResult, objectAGlobal),
	})

	runnerMock.AddExecutionClassify("Foo", flags, nil)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		switch res.Callee {
		case objectAGlobal:
			assert.Equal(t, expectedResult, res.ReturnArguments)
		default:
			assert.Fail(t, "unexpected VCallResult")
		}
		return false
	})

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(flags.Interference, flags.State),
		Caller:              server.GlobalCaller(),
		Callee:              objectAGlobal,
		CallSiteDeclaration: class,
		CallSiteMethod:      "Foo",
		CallOutgoing:        server.RandomLocalWithPulse(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	server.SendPayload(ctx, &pl)
	{
		testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
		testutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	require.Equal(t, 0, typedChecker.VCallRequest.Count())
	require.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

// unordered A.Foo sends ordered outgoing and receives error
func TestVirtual_CallConstructorFromContract_InterferenceViolation(t *testing.T) {
	t.Log("C5203")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)
	var (
		class = gen.UniqueGlobalRef()

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		flags = contract.MethodIsolation{
			Interference: contract.CallIntolerable,
			State:        contract.CallDirty,
		}
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

	outgoingCallRef := gen.UniqueGlobalRef()

	expectedError, err := foundation.MarshalMethodErrorResult(throw.E("interference violation: constructor call from unordered call"))
	require.NoError(t, err)

	expectedResult := []byte("finish A.Foo")

	outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallConstructor(class, "Bar", []byte("123"))
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.
		AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, objectAGlobal, ctx.Object)
				assert.Equal(t, flags, ctx.Isolation)
				t.Log("ExecutionStart [A.Foo]")
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Error:    nil,
				Outgoing: outgoingCall,
			},
		).
		AddContinue(func(result []byte) {
			assert.Equal(t, expectedError, result)

		}, &execution.Update{
			Type:   execution.Done,
			Result: requestresult.New(expectedResult, objectAGlobal),
		})

	runnerMock.AddExecutionClassify("Foo", flags, nil)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		switch res.Callee {
		case objectAGlobal:
			assert.Equal(t, expectedResult, res.ReturnArguments)
		default:
			assert.Fail(t, "unexpected VCallResult")
		}
		return false
	})

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(flags.Interference, flags.State),
		Caller:              server.GlobalCaller(),
		Callee:              objectAGlobal,
		CallSiteDeclaration: class,
		CallSiteMethod:      "Foo",
		CallOutgoing:        server.RandomLocalWithPulse(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	server.SendPayload(ctx, &pl)
	{
		testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
		testutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	require.Equal(t, 0, typedChecker.VCallRequest.Count())
	require.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

// A.Foo calls ordered B1.Bar, B2.Bar, B3.Bar
func TestVirtual_CallMultipleContractsFromContract_Ordered(t *testing.T) {
	t.Log("C5114")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 4)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.Callee.String()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		flags     = contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		callFlags = payload.BuildCallFlags(flags.Interference, flags.State)

		outgoingA     = server.RandomLocalWithPulse()
		objectAGlobal = reference.NewSelf(outgoingA)

		classB         = gen.UniqueGlobalRef()
		objectB1Global = reference.NewSelf(server.RandomLocalWithPulse())
		objectB2Global = reference.NewSelf(server.RandomLocalWithPulse())
		objectB3Global = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingCallRef = gen.UniqueGlobalRef()
	)

	// create objects
	{
		Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
		Method_PrepareObject(ctx, server, payload.Ready, objectB1Global)
		Method_PrepareObject(ctx, server, payload.Ready, objectB2Global)
		Method_PrepareObject(ctx, server, payload.Ready, objectB3Global)
	}

	// add ExecutionMocks to runnerMock
	{
		builder := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal)
		objectAExecutionMock := runnerMock.AddExecutionMock(objectAGlobal.String())
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.Foo]")
				require.Equal(t, server.GlobalCaller(), ctx.Request.Caller)
				require.Equal(t, objectAGlobal, ctx.Request.Callee)
				require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: builder.CallMethod(objectB1Global, classB, "Bar", []byte("B1")),
			},
		)

		objectAExecutionMock.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B1.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: builder.CallMethod(objectB2Global, classB, "Bar", []byte("B2")),
			},
		)
		objectAExecutionMock.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B2.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: builder.CallMethod(objectB3Global, classB, "Bar", []byte("B3")),
			},
		)
		objectAExecutionMock.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B3.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		runnerMock.AddExecutionMock(objectB1Global.String()).AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B1.Bar]")
				require.Equal(t, objectB1Global, ctx.Request.Callee)
				require.Equal(t, objectAGlobal, ctx.Request.Caller)
				require.Equal(t, []byte("B1"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish B1.Bar"), objectB1Global),
			},
		)

		runnerMock.AddExecutionMock(objectB2Global.String()).AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B2.Bar]")
				require.Equal(t, objectB2Global, ctx.Request.Callee)
				require.Equal(t, objectAGlobal, ctx.Request.Caller)
				require.Equal(t, []byte("B2"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish B2.Bar"), objectB2Global),
			},
		)

		runnerMock.AddExecutionMock(objectB3Global.String()).AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B3.Bar]")
				require.Equal(t, objectB3Global, ctx.Request.Callee)
				require.Equal(t, objectAGlobal, ctx.Request.Caller)
				require.Equal(t, []byte("B3"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish B3.Bar"), objectB3Global),
			},
		)

		runnerMock.AddExecutionClassify(objectAGlobal.String(), flags, nil)
		runnerMock.AddExecutionClassify(objectB1Global.String(), flags, nil)
		runnerMock.AddExecutionClassify(objectB2Global.String(), flags, nil)
		runnerMock.AddExecutionClassify(objectB3Global.String(), flags, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, objectAGlobal, request.Caller)
			assert.Equal(t, payload.CTMethod, request.CallType)
			assert.Equal(t, outgoingCallRef, request.CallReason)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.Pulse())

			switch request.Callee {
			case objectB1Global:
				require.Equal(t, []byte("B1"), request.Arguments)
				require.Equal(t, uint32(1), request.CallSequence)
			case objectB2Global:
				require.Equal(t, []byte("B2"), request.Arguments)
				require.Equal(t, uint32(2), request.CallSequence)
			case objectB3Global:
				require.Equal(t, []byte("B3"), request.Arguments)
				require.Equal(t, uint32(3), request.CallSequence)
			default:
				t.Fatal("wrong Callee")
			}
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, payload.CTMethod, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee {
			case objectAGlobal:
				require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
				require.Equal(t, server.GlobalCaller(), res.Caller)
				require.Equal(t, outgoingA, res.CallOutgoing)
			case objectB1Global:
				require.Equal(t, []byte("finish B1.Bar"), res.ReturnArguments)
				require.Equal(t, objectAGlobal, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.Pulse())
			case objectB2Global:
				require.Equal(t, []byte("finish B2.Bar"), res.ReturnArguments)
				require.Equal(t, objectAGlobal, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.Pulse())
			case objectB3Global:
				require.Equal(t, []byte("finish B3.Bar"), res.ReturnArguments)
				require.Equal(t, objectAGlobal, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.Pulse())
			default:
				t.Fatal("wrong Callee")
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == objectAGlobal
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      callFlags,
		Caller:         server.GlobalCaller(),
		Callee:         objectAGlobal,
		CallSiteMethod: "Foo",
		CallOutgoing:   outgoingA,
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}
	msg := server.WrapPayload(&pl).Finalize()
	server.SendMessage(ctx, msg)

	// wait for all calls and SMs
	testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 3, typedChecker.VCallRequest.Count())
	require.Equal(t, 4, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Method_Have_ObjectState(t *testing.T) {
	type runnerObjectChecker func(objectState *payload.VStateReport_ProvidedContentBody, runnerObjectState descriptor.Object) bool
	table := []struct {
		name string
		code string
		skip string

		state  contract.StateFlag
		checks []runnerObjectChecker
	}{
		{
			name:  "Method with CallFlags.Dirty must be called with dirty object state",
			code:  "C5184",
			skip:  "",
			state: contract.CallDirty,
		},
		{
			name:  "Method with CallFlags.Validated must be called with validated object state",
			code:  "C5123",
			skip:  "https://insolar.atlassian.net/browse/PLAT-404",
			state: contract.CallValidated,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			t.Log(test.code)
			if len(test.skip) > 0 {
				t.Skip(test.skip)
			}

			var (
				mc = minimock.NewController(t)
			)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			runnerMock := logicless.NewServiceMock(ctx, t, nil)
			server.ReplaceRunner(runnerMock)

			server.Init(ctx)
			server.IncrementPulse(ctx)

			var (
				class             = gen.UniqueGlobalRef()
				objectLocal       = server.RandomLocalWithPulse()
				objectGlobal      = reference.NewSelf(objectLocal)
				dirtyStateRef     = server.RandomLocalWithPulse()
				dirtyState        = reference.NewSelf(dirtyStateRef)
				validatedStateRef = server.RandomLocalWithPulse()
				validatedState    = reference.NewSelf(validatedStateRef)
			)
			const (
				validatedMem = "12345"
				dirtyMem     = "54321"
			)

			{ // send object state to server
				pl := payload.VStateReport{
					Status:               payload.Ready,
					Object:               objectGlobal,
					LatestValidatedState: validatedState,
					LatestDirtyState:     dirtyState,
					ProvidedContent: &payload.VStateReport_ProvidedContentBody{
						LatestValidatedState: &payload.ObjectState{
							Reference: validatedStateRef,
							Class:     class,
							State:     []byte(validatedMem),
						},
						LatestDirtyState: &payload.ObjectState{
							Reference: dirtyStateRef,
							Class:     class,
							State:     []byte(dirtyMem),
						},
					},
					UnorderedPendingEarliestPulse: pulse.OfNow(),
				}

				server.WaitIdleConveyor()
				server.SendPayload(ctx, &pl)
				server.WaitActiveThenIdleConveyor()
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			{
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					require.Equal(t, res.ReturnArguments, []byte("345"))
					require.Equal(t, res.Callee, objectGlobal)

					return false // no resend msg
				})

				pl := payload.VCallRequest{
					CallType:            payload.CTMethod,
					CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, test.state),
					CallAsOf:            0,
					Caller:              server.GlobalCaller(),
					Callee:              objectGlobal,
					CallSiteDeclaration: class,
					CallSiteMethod:      "Test",
					CallOutgoing:        objectLocal,
					Arguments:           insolar.MustSerialize([]interface{}{}),
				}

				key := calculateOutgoing(pl).String()
				runnerMock.AddExecutionMock(key).
					AddStart(func(ctx execution.Context) {
						require.Equal(t, objectGlobal, ctx.Object)
						require.Equal(t, test.state, ctx.Request.CallFlags.GetState())
						require.Equal(t, test.state, ctx.Isolation.State)
						require.Equal(t, objectGlobal, ctx.ObjectDescriptor.HeadRef())
						stateClass, err := ctx.ObjectDescriptor.Class()
						require.NoError(t, err)
						require.Equal(t, class, stateClass)

						if test.state == contract.CallValidated {
							require.Equal(t, validatedStateRef, ctx.ObjectDescriptor.StateID())
							require.Equal(t, []byte(validatedMem), ctx.ObjectDescriptor.Memory())
						} else {
							require.Equal(t, dirtyStateRef, ctx.ObjectDescriptor.StateID())
							require.Equal(t, []byte(dirtyMem), ctx.ObjectDescriptor.Memory())
						}
					}, &execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("345"), objectGlobal),
					})
				runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
					Interference: contract.CallIntolerable,
					State:        test.state,
				}, nil)

				server.SendPayload(ctx, &pl)
			}

			testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

// twice ( A.Foo -> B.Bar, B.Bar )
func TestVirtual_CallContractTwoTimes(t *testing.T) {
	t.Log("C5183")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallOutgoing.String()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		flags     = contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		callFlags = payload.BuildCallFlags(flags.Interference, flags.State)

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		classB        = gen.UniqueGlobalRef()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingFirstCall  = server.RandomLocalWithPulse()
		outgoingSecondCall = server.RandomLocalWithPulse()

		outgoingReasonFirst  = reference.NewSelf(outgoingFirstCall)
		outgoingReasonSecond = reference.NewSelf(outgoingSecondCall)
	)

	// create objects
	{
		Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
		Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)
	}

	// add ExecutionMocks to runnerMock
	{
		firstBuilder := execution.NewRPCBuilder(outgoingReasonFirst, objectAGlobal)
		objectAExecutionFirstMock := runnerMock.AddExecutionMock(outgoingFirstCall.String())
		objectAExecutionFirstMock.AddStart(nil,
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: firstBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("first")),
			},
		)
		objectAExecutionFirstMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: firstBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("second")),
			},
		)
		objectAExecutionFirstMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		secondBuilder := execution.NewRPCBuilder(outgoingReasonSecond, objectAGlobal)
		objectAExecutionSecondMock := runnerMock.AddExecutionMock(outgoingSecondCall.String())
		objectAExecutionSecondMock.AddStart(nil,
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: secondBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("first")),
			},
		)
		objectAExecutionSecondMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: secondBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("second")),
			},
		)
		objectAExecutionSecondMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		runnerMock.AddExecutionClassify(outgoingFirstCall.String(), flags, nil)
		runnerMock.AddExecutionClassify(outgoingSecondCall.String(), flags, nil)
	}

	// add publish checker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			switch string(request.Arguments[0]) {
			case "f":
				require.Equal(t, []byte("first"), request.Arguments)
				require.Equal(t, uint32(1), request.CallSequence)
			case "s":
				require.Equal(t, []byte("second"), request.Arguments)
				require.Equal(t, uint32(2), request.CallSequence)
			default:
				t.Fatal("wrong call args")
			}

			result := payload.VCallResult{
				CallType:        request.CallType,
				CallFlags:       request.CallFlags,
				Caller:          request.Caller,
				Callee:          request.Callee,
				CallOutgoing:    request.CallOutgoing,
				ReturnArguments: []byte("finish B.Bar"),
			}
			msg := server.WrapPayload(&result).Finalize()
			server.SendMessage(ctx, msg)

			return false
		})
		typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
			require.Equal(t, []byte("finish A.Foo"), result.ReturnArguments)
			return false
		})
	}

	// send first VCallRequest A.Foo
	{
		pl := payload.VCallRequest{
			CallType:       payload.CTMethod,
			CallFlags:      callFlags,
			Caller:         server.GlobalCaller(),
			Callee:         objectAGlobal,
			CallSiteMethod: "Foo",
			CallOutgoing:   outgoingFirstCall,
			Arguments:      []byte("call foo"),
		}
		msg := server.WrapPayload(&pl).Finalize()
		server.SendMessage(ctx, msg)
		server.WaitActiveThenIdleConveyor()
	}
	// send second VCallRequest A.Foo
	{
		pl := payload.VCallRequest{
			CallType:       payload.CTMethod,
			CallFlags:      callFlags,
			Caller:         server.GlobalCaller(),
			Callee:         objectAGlobal,
			CallSiteMethod: "Foo",
			CallOutgoing:   outgoingSecondCall,
			Arguments:      []byte("call foo"),
		}
		msg := server.WrapPayload(&pl).Finalize()
		server.SendMessage(ctx, msg)
	}

	// wait for all calls and SMs
	{
		testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
		testutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	require.Equal(t, 4, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}
