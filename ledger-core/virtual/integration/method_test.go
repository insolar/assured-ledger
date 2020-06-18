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
	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, payload).SetSender(server.JetCoordinatorMock.Me()).Finalize()

	server.WaitIdleConveyor()
	server.SendMessage(ctx, msg)
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
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

		server.SendMessage(ctx, msg)

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
	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

	countBefore := server.PublisherMock.GetCount()
	server.SendMessage(ctx, msg)

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
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

		server.SendMessage(ctx, msg)

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
			msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

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

			server.SendMessage(ctx, msg)
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
		it := server.Journal.GetJournalIterator()
		select {
		case <-it.WaitStop(&execute.SMExecute{}, 2):
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
		}
		it.Stop()
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

	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()
	beforeCount := server.PublisherMock.GetCount()
	server.SendMessage(ctx, msg)
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

	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()
	beforeCount := server.PublisherMock.GetCount()
	server.SendMessage(ctx, msg)
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

	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()
	beforeCount := server.PublisherMock.GetCount()
	server.SendMessage(ctx, msg)
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

	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()
	beforeCount := server.PublisherMock.GetCount()
	server.SendMessage(ctx, msg)
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

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		isolation = contract.ConstructorIsolation()

		classA        = gen.UniqueReference()
		outgoingA     = server.RandomLocalWithPulse()
		objectAGlobal = reference.NewSelf(outgoingA)

		classB        = gen.UniqueReference()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := executionevent.NewRPCBuilder(gen.UniqueReference(), classA).CallMethod(objectBGlobal, classB, "Foo", []byte("123"))
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
				require.Equal(t, classA, ctx.Request.Caller)
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
			require.Equal(t, objectBGlobal, request.Callee)
			require.Equal(t, classA, request.Caller)
			require.Equal(t, []byte("123"), request.Arguments)
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			switch res.Callee {
			case classA:
				require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
			case objectBGlobal:
				require.Equal(t, []byte("finish B.Foo"), res.ReturnArguments)
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == classA
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         classA,
		CallSiteMethod: "New",
		CallOutgoing:   outgoingA,
	}
	msg := server.WrapPayload(&pl).Finalize()
	server.SendMessage(ctx, msg)

	// wait for all calls and SMs
	{
		it := server.Journal.GetJournalIterator()
		select {
		case <-it.WaitStop(&execute.SMExecute{}, 2):
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
		}
		select {
		case <-it.WaitAllAsyncCallsFinished():
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
		}
		it.Stop()
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

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		isolation = contract.ConstructorIsolation()

		classA        = gen.UniqueReference()
		outgoingA     = server.RandomLocalWithPulse()
		objectAGlobal = reference.NewSelf(outgoingA)

		classB        = gen.UniqueReference()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := executionevent.NewRPCBuilder(gen.UniqueReference(), classA).CallMethod(objectBGlobal, classB, "Foo", []byte("123"))
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
				require.Equal(t, classA, ctx.Request.Caller)
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
			require.Equal(t, objectBGlobal, request.Callee)
			require.Equal(t, classA, request.Caller)
			require.Equal(t, []byte("123"), request.Arguments)
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			switch res.Callee {
			case classA:
				require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
			case objectBGlobal:
				require.Equal(t, []byte("finish B.Foo"), res.ReturnArguments)
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == classA
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         classA,
		CallSiteMethod: "New",
		CallOutgoing:   outgoingA,
	}
	msg := server.WrapPayload(&pl).Finalize()
	server.SendMessage(ctx, msg)

	// wait for all calls and SMs
	{
		it := server.Journal.GetJournalIterator()
		select {
		case <-it.WaitStop(&execute.SMExecute{}, 2):
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
		}
		select {
		case <-it.WaitAllAsyncCallsFinished():
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
		}
		it.Stop()
	}

	require.Equal(t, 1, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}
