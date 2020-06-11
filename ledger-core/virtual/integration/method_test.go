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
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
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
		Callee:                        object,
		UnorderedPendingEarliestPulse: pulse.OfNow(),
		ProvidedContent:               content,
	}
	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, payload).Finalize()

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
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()

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
	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()

	countBefore := server.PublisherMock.GetCount()
	server.SendMessage(ctx, msg)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Error("failed to wait for result")
	}

	mc.Finish()
}

func TestVirtual_Method_WithExecutor_ObjectIsNotExist(t *testing.T) {
	t.Log("C4974")
	// t.Skip("https://insolar.atlassian.net/browse/PLAT-395")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

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
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()

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

	{
		typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, res.ReturnArguments, []byte("345"))
			require.Equal(t, res.Callee, objectGlobal)

			return false // no resend msg
		}).ExpectedCount(2)

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
			msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()

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
	typedChecker.VCallResult.ExpectedCount(1).SetResend()
	typedChecker.VCallRequest.ExpectedCount(1).SetResend()
	typedChecker.VStateReport.ExpectedCount(1).SetResend()

	server.IncrementPulseAndWaitIdle(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	// Change pulse to force send VStateRequest
	server.IncrementPulseAndWaitIdle(ctx)

	checkBalance(ctx, t, server, objectGlobal, initialBalance)

	mc.Finish()
}
