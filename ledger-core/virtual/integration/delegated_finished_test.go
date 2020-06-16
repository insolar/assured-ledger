// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

func makeEmptyResult() []byte {
	var ret0 error

	return insolar.MustSerialize(foundation.Result{
		Error:   nil,
		Returns: []interface{}{ret0},
	})
}

type callMethodFunc = func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error)
type callConstructorFunc = func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) (objectState []byte, result []byte, err error)

func mockExecutor(t *testing.T, server *utils.Server, callMethod callMethodFunc, constructorFunc callConstructorFunc) {
	builtinExecutor, err := server.Runner.Manager.GetExecutor(machine.Builtin)
	require.NoError(t, err)

	cache := server.Runner.Cache
	_, walletCodeRef, err := cache.ByClassRef(context.Background(), testwallet.GetClass())
	require.NoError(t, err)

	executorMock := machine.NewExecutorMock(t)
	if callMethod != nil {
		executorMock.CallMethodMock.Set(callMethod)
	}
	if constructorFunc != nil {
		executorMock.CallConstructorMock.Set(constructorFunc)
	}
	executorMock.ClassifyMethodMock.Set(builtinExecutor.ClassifyMethod)
	manager := machine.NewManager()
	err = manager.RegisterExecutor(machine.Builtin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(manager)

	cacheMock := descriptor.NewCacheMock(t)
	server.ReplaceCache(cacheMock)
	cacheMock.ByClassRefMock.Return(
		descriptor.NewClass(gen.UniqueReference(), gen.UniqueID(), testwallet.GetClass()),
		descriptor.NewCode(nil, machine.Builtin, walletCodeRef.Ref()),
		nil,
	)
}

// 1. Send CallRequest
// 2. Change pulse in mocked executor
// 4. Since we changed pulse during execution, we expect that VDelegatedRequestFinished will be sent
// 5. Check that in VDelegatedRequestFinished new object state is stored
func TestVirtual_SendDelegatedFinished_IfPulseChanged_WithSideEffect(t *testing.T) {
	t.Log("C4935")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewUninitializedServer(nil, t) // TODO PLAT-367 fix test to be stable and have no errors in logs
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	gotDelegatedRequestFinished := make(chan *payload.VDelegatedRequestFinished, 1)

	{
		typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
		typedChecker.SetDefaultResend(true)
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			gotDelegatedRequestFinished <- finished
			return false
		})
		typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool { return false })
	}

	var (
		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		newState     = makeRawWalletState(initialBalance + 100)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: class,
			CallSiteMethod:      "AddAmount",
			CallOutgoing:        server.RandomLocalWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

		newObjectDescriptor := descriptor.NewObject(reference.Global{}, objectLocal, class, []byte(""), reference.Global{})

		result := requestresult.New(makeEmptyResult(), objectGlobal)
		result.SetAmend(newObjectDescriptor, newState)

		key := calculateOutgoing(pl).String()
		runnerMock.AddExecutionMock(key).
			AddStart(func(execution execution.Context) {
				server.IncrementPulse(ctx)
			}, &executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Result: result,
			})
		runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}, nil)

		beforeCount := server.PublisherMock.GetCount()
		server.SendMessage(ctx, msg)
		if !server.PublisherMock.WaitCount(beforeCount+5, 10*time.Second) {
			require.Fail(t, "timeout waiting for message")
		}
	}

	server.WaitActiveThenIdleConveyor()

	select {
	case delegateFinishedMsg := <-gotDelegatedRequestFinished:
		callFlags := payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)

		require.Equal(t, objectGlobal, delegateFinishedMsg.Callee)
		require.Equal(t, payload.CTMethod, delegateFinishedMsg.CallType)
		require.Equal(t, callFlags, delegateFinishedMsg.CallFlags)

		require.Equal(t, newState, delegateFinishedMsg.LatestState.State)
	default:
		require.Fail(t, "unexpected situation")
	}
}

// In this case VDelegatedRequestFinished must not contain new state since it was not changed
func TestVirtual_SendDelegatedFinished_IfPulseChanged_Without_SideEffect(t *testing.T) {
	t.Log("C4990")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)

		typedChecker = server.PublisherMock.SetTypedChecker(ctx, mc, server)
	)

	{
		typedChecker.VCallResult.SetResend(false)
		typedChecker.VDelegatedCallResponse.SetResend(true)
		typedChecker.VDelegatedCallRequest.SetResend(true)
		typedChecker.VStateReport.SetResend(true)
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			require.Equal(t, objectGlobal, finished.Callee)
			require.Equal(t, payload.CTMethod, finished.CallType)
			require.Equal(t, payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty), finished.CallFlags)

			require.Nil(t, finished.LatestState)

			return true
		})
	}

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: class,
			CallSiteMethod:      "AddAmount",
			CallOutgoing:        server.RandomLocalWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

		key := calculateOutgoing(pl).String()
		runnerMock.AddExecutionMock(key).
			AddStart(func(execution execution.Context) {
				server.IncrementPulse(ctx)
			}, &executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Result: requestresult.New(makeEmptyResult(), objectGlobal),
			})
		runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}, nil)

		server.SendMessage(ctx, msg)
	}

	{
		it := server.Journal.GetJournalIterator()
		select {
		case <-it.WaitStop(&object.SMAwaitDelegate{}, 1):
		case <-time.After(10 * time.Second):
			require.FailNow(t, "timeout")
		}
		select {
		case <-it.WaitAllAsyncCallsFinished():
		case <-time.After(10 * time.Second):
			require.FailNow(t, "timeout")
		}
		it.Stop()
	}

	{
		require.Equal(t, typedChecker.Handlers.VCallResult.Count.Load(), 1)
		require.Equal(t, typedChecker.Handlers.VDelegatedCallResponse.Count.Load(), 1)
		require.Equal(t, typedChecker.Handlers.VDelegatedCallRequest.Count.Load(), 1)
		require.Equal(t, typedChecker.Handlers.VStateReport.Count.Load(), 1)
		require.Equal(t, typedChecker.Handlers.VDelegatedRequestFinished.Count.Load(), 1)
	}
}

func TestVirtual_SendDelegatedFinished_IfPulseChanged_Constructor(t *testing.T) {
	t.Log("C4988")
	t.Skip("skipped until PLAT-304")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	testBalance := uint32(333)
	rawWalletState := makeRawWalletState(testBalance)
	callConstructor := func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) (objectState []byte, result []byte, err error) {
		// we want to change pulse during construction
		server.IncrementPulseAndWaitIdle(ctx)

		emptyResult := makeEmptyResult()
		return rawWalletState, emptyResult, nil
	}

	gotDelegatedRequestFinished := make(chan *payload.VDelegatedRequestFinished, 0)
	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		require.NoError(t, err)

		switch payLoadData := pl.(type) {
		case *payload.VDelegatedRequestFinished:
			gotDelegatedRequestFinished <- payLoadData
		default:
			t.Logf("Going message: %T", payLoadData)
		}

		server.SendMessage(ctx, messages[0])
		return nil
	})

	mockExecutor(t, server, nil, callConstructor)
	code, _ := server.CallAPICreateWallet(ctx)
	require.Equal(t, 200, code)

	select {
	case delegateFinishedMsg := <-gotDelegatedRequestFinished:
		callFlags := payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)

		require.NotEmpty(t, delegateFinishedMsg.Callee)
		require.Equal(t, payload.CTConstructor, delegateFinishedMsg.CallType)
		require.Equal(t, callFlags, delegateFinishedMsg.CallFlags)

		latestState := delegateFinishedMsg.LatestState
		require.Equal(t, rawWalletState, latestState.State)
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}
}
