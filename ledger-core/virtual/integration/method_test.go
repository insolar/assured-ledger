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
	"github.com/insolar/assured-ledger/ledger-core/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
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
	t.Skip("https://insolar.atlassian.net/browse/PLAT-397")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		class        = testwallet.GetClass()
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
			CallSiteDeclaration: class,
			CallSiteMethod:      "random",
			CallOutgoing:        server.RandomLocalWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}

		server.SendPayload(ctx, &pl)

		// TODO fix it after implementation https://insolar.atlassian.net/browse/PLAT-397
	}
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
				AddStart(checkExecution, &executionupdate.ContractExecutionStateUpdate{
					Type:   executionupdate.Done,
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

	{
		select {
		case <-executeDone:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "timeout")
		}
	}

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
		class = gen.UniqueReference()

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		flags = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	outgoingCall := executionevent.NewRPCBuilder(gen.UniqueReference(), objectAGlobal).CallMethod(objectBGlobal, class, "Bar", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			require.Equal(t, objectAGlobal, ctx.Request.Callee)
			t.Log("ExecutionStart [A.Foo]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:     executionupdate.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.Foo]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock("Bar").AddStart(
		func(ctx execution.Context) {
			require.Equal(t, objectBGlobal, ctx.Request.Callee)
			t.Log("ExecutionStart [B.Bar]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
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
		class = gen.UniqueReference()

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	outgoingCall := executionevent.NewRPCBuilder(gen.UniqueReference(), objectAGlobal).CallMethod(objectBGlobal, class, "Bar", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			require.Equal(t, objectAGlobal, ctx.Request.Callee)
			t.Log("ExecutionStart [A.Foo]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:     executionupdate.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.Foo]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock("Bar").AddStart(
		func(ctx execution.Context) {
			require.Equal(t, objectBGlobal, ctx.Request.Callee)
			t.Log("ExecutionStart [B.Bar]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
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
		class         = gen.UniqueReference()
		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

	outgoingCall := executionevent.NewRPCBuilder(gen.UniqueReference(), objectAGlobal).CallMethod(objectAGlobal, class, "Bar", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.Foo]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:     executionupdate.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.Foo]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock("Bar").AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.Bar]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
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
		class         = gen.UniqueReference()
		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

	outgoingCall := executionevent.NewRPCBuilder(gen.UniqueReference(), objectAGlobal).CallMethod(objectAGlobal, class, "Bar", []byte{})
	objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
	objectAExecutionMock.AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.Foo]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:     executionupdate.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		},
	)
	objectAExecutionMock.AddContinue(
		func(result []byte) {
			t.Log("ExecutionContinue [A.Foo]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		},
	)

	runnerMock.AddExecutionMock("Bar").AddStart(
		func(ctx execution.Context) {
			t.Log("ExecutionStart [A.Bar]")
		},
		&executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
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

		classA        = gen.UniqueReference()
		outgoingA     = server.RandomLocalWithPulse()
		objectAGlobal = reference.NewSelf(outgoingA)

		classB        = gen.UniqueReference()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingCallRef = gen.UniqueReference()
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := executionevent.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectBGlobal, classB, "Foo", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), objectAGlobal)
		objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))
		objectAExecutionMock := runnerMock.AddExecutionMock("New")
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.New]")
				require.Equal(t, classA, ctx.Request.Callee)
				require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
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
				require.Equal(t, []byte("finish B.Foo"), result)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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
	{
		select {
		case <-executeDone:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "timeout")
		}
		select {
		case <-server.Journal.WaitAllAsyncCallsDone():
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
		}
	}

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

		classA        = gen.UniqueReference()
		outgoingA     = server.RandomLocalWithPulse()
		objectAGlobal = reference.NewSelf(outgoingA)

		classB        = gen.UniqueReference()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingCallRef = gen.UniqueReference()
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := executionevent.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectBGlobal, classB, "Foo", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), objectAGlobal)
		objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))
		objectAExecutionMock := runnerMock.AddExecutionMock("New")
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.New]")
				require.Equal(t, classA, ctx.Request.Callee)
				require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
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
				require.Equal(t, []byte("finish B.Foo"), result)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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
	{
		select {
		case <-executeDone:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "timeout")
		}
		select {
		case <-server.Journal.WaitAllAsyncCallsDone():
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
		}
	}

	require.Equal(t, 1, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

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

		classB         = gen.UniqueReference()
		objectB1Global = reference.NewSelf(server.RandomLocalWithPulse())
		objectB2Global = reference.NewSelf(server.RandomLocalWithPulse())
		objectB3Global = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingCallRef = gen.UniqueReference()
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
		builder := executionevent.NewRPCBuilder(outgoingCallRef, objectAGlobal)
		objectAExecutionMock := runnerMock.AddExecutionMock(objectAGlobal.String())
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.Foo]")
				require.Equal(t, server.GlobalCaller(), ctx.Request.Caller)
				require.Equal(t, objectAGlobal, ctx.Request.Callee)
				require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:     executionupdate.OutgoingCall,
				Error:    nil,
				Outgoing: builder.CallMethod(objectB1Global, classB, "Bar", []byte("B1")),
			},
		)

		objectAExecutionMock.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B1.Bar"), result)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:     executionupdate.OutgoingCall,
				Error:    nil,
				Outgoing: builder.CallMethod(objectB2Global, classB, "Bar", []byte("B2")),
			},
		)
		objectAExecutionMock.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B2.Bar"), result)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:     executionupdate.OutgoingCall,
				Error:    nil,
				Outgoing: builder.CallMethod(objectB3Global, classB, "Bar", []byte("B3")),
			},
		)
		objectAExecutionMock.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B3.Bar"), result)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Error:  nil,
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
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
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
	{
		select {
		case <-executeDone:
		case <-time.After(20 * time.Second):
			require.FailNow(t, "timeout")
		}
		select {
		case <-server.Journal.WaitAllAsyncCallsDone():
		case <-time.After(20 * time.Second):
			require.FailNow(t, "timeout")
		}
	}

	require.Equal(t, 3, typedChecker.VCallRequest.Count())
	require.Equal(t, 4, typedChecker.VCallResult.Count())

	mc.Finish()
}

// call A.Foo -> B.Bar two times
func TestVirtual_CallContractTwoTimes(t *testing.T) {
	t.Log("C5183")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 4)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		switch execution.Request.CallSiteMethod {
		case "Foo":
			return execution.Request.CallOutgoing.String()
		case "Bar":
			return execution.Request.CallReason.String()
		default:
			return "default"
		}
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		flags     = contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		callFlags = payload.BuildCallFlags(flags.Interference, flags.State)

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		classB        = gen.UniqueReference()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingFirstCall  = server.RandomLocalWithPulse()
		outgoingSecondCall = server.RandomLocalWithPulse()

		outgoingReasonFirst  = gen.UniqueReference()
		outgoingReasonSecond = gen.UniqueReference()
	)

	// create objects
	{
		Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
		Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)
	}

	// add ExecutionMocks to runnerMock
	{
		objectAExecutionMockFirst := runnerMock.AddExecutionMock(outgoingFirstCall.String())
		objectAExecutionMockFirst.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.Foo] first call")
				require.Equal(t, server.GlobalCaller(), ctx.Request.Caller)
				require.Equal(t, objectAGlobal, ctx.Request.Callee)
				require.Equal(t, outgoingFirstCall, ctx.Request.CallOutgoing)
				require.Equal(t, []byte("first"), ctx.Request.Arguments)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:     executionupdate.OutgoingCall,
				Error:    nil,
				Outgoing: executionevent.NewRPCBuilder(outgoingReasonFirst, objectAGlobal).CallMethod(objectBGlobal, classB, "Bar", []byte("first outgoing")),
			},
		)
		objectAExecutionMockFirst.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.Foo] first call")
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Error:  nil,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		objectAExecutionMockSecond := runnerMock.AddExecutionMock(outgoingSecondCall.String())
		objectAExecutionMockSecond.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.Foo] second call")
				require.Equal(t, server.GlobalCaller(), ctx.Request.Caller)
				require.Equal(t, objectAGlobal, ctx.Request.Callee)
				require.Equal(t, outgoingSecondCall, ctx.Request.CallOutgoing)
				require.Equal(t, []byte("second"), ctx.Request.Arguments)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:     executionupdate.OutgoingCall,
				Error:    nil,
				Outgoing: executionevent.NewRPCBuilder(outgoingReasonSecond, objectAGlobal).CallMethod(objectBGlobal, classB, "Bar", []byte("second outgoing")),
			},
		)
		objectAExecutionMockSecond.AddContinue(
			func(result []byte) {
				t.Log("ExecutionContinue [A.Foo] second call")
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Error:  nil,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		runnerMock.AddExecutionMock(outgoingReasonFirst.String()).AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B.Bar] first call")
				require.Equal(t, objectBGlobal, ctx.Request.Callee)
				require.Equal(t, objectAGlobal, ctx.Request.Caller)
				require.Equal(t, []byte("first outgoing"), ctx.Request.Arguments)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Result: requestresult.New([]byte("finish B.Bar"), objectBGlobal),
			},
		)

		runnerMock.AddExecutionMock(outgoingReasonSecond.String()).AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B.Bar] second call")
				require.Equal(t, objectBGlobal, ctx.Request.Callee)
				require.Equal(t, objectAGlobal, ctx.Request.Caller)
				require.Equal(t, []byte("second outgoing"), ctx.Request.Arguments)
			},
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Result: requestresult.New([]byte("finish B.Bar"), objectBGlobal),
			},
		)

		runnerMock.AddExecutionClassify(outgoingFirstCall.String(), flags, nil)
		runnerMock.AddExecutionClassify(outgoingSecondCall.String(), flags, nil)
		runnerMock.AddExecutionClassify(outgoingReasonFirst.String(), flags, nil)
		runnerMock.AddExecutionClassify(outgoingReasonSecond.String(), flags, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, objectAGlobal, request.Caller)
			assert.Equal(t, objectBGlobal, request.Callee)
			assert.Equal(t, payload.CTMethod, request.CallType)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.Pulse())

			switch request.CallReason {
			case outgoingReasonFirst:
				require.Equal(t, []byte("first outgoing"), request.Arguments)
				require.Equal(t, uint32(1), request.CallSequence)
			case outgoingReasonSecond:
				require.Equal(t, []byte("second outgoing"), request.Arguments)
				require.Equal(t, uint32(1), request.CallSequence)
			default:
				t.Fatal("wrong CallReason")
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
				switch res.CallOutgoing {
				case outgoingFirstCall:
				case outgoingSecondCall:
				default:
					t.Fatal("wrong CallOutgoing")
				}
			case objectBGlobal:
				require.Equal(t, []byte("finish B.Bar"), res.ReturnArguments)
				require.Equal(t, objectAGlobal, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.Pulse())
			default:
				t.Fatal("wrong Callee")
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == objectAGlobal
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
			Arguments:      []byte("first"),
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
			Arguments:      []byte("second"),
		}
		msg := server.WrapPayload(&pl).Finalize()
		server.SendMessage(ctx, msg)
	}

	// wait for all calls and SMs
	{
		select {
		case <-executeDone:
		case <-time.After(20 * time.Second):
			require.FailNow(t, "timeout")
		}
		select {
		case <-server.Journal.WaitAllAsyncCallsDone():
		case <-time.After(20 * time.Second):
			require.FailNow(t, "timeout")
		}
	}

	require.Equal(t, 2, typedChecker.VCallRequest.Count())
	require.Equal(t, 4, typedChecker.VCallResult.Count())

	mc.Finish()
}
