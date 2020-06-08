// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func makeEmptyResult(t *testing.T) []byte {
	var ret0 error
	ser := common.NewCBORSerializer()
	var emptyResult []byte
	err := ser.Serialize(foundation.Result{Returns: []interface{}{ret0}}, &emptyResult)
	require.NoError(t, err)

	return emptyResult
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
func TestVirtual_SendDelegatedFinished_IfPulseChanged_WithSideAffect(t *testing.T) {
	t.Log("C4935")

	server, ctx := utils.NewServerIgnoreLogErrors(nil, t) // TODO PLAT-367 fix test to be stable and have no errors in logs
	defer server.Stop()

	var countVCallResult int
	gotDelegatedRequestFinished := make(chan *payload.VDelegatedRequestFinished, 0)
	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		if err != nil {
			return nil
		}

		switch payLoadData := pl.(type) {
		case *payload.VDelegatedRequestFinished:
			gotDelegatedRequestFinished <- payLoadData
		case *payload.VCallResult:
			countVCallResult++
		default:
			fmt.Printf("Going message: %T", payLoadData)
		}

		server.SendMessage(ctx, messages[0])
		return nil
	})

	testBalance := uint32(555)
	additionalBalance := uint(133)
	objectRef := gen.UniqueReference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)

	// send VStateReport: save wallet
	rawWalletState := makeRawWalletState(t, testBalance)
	msg := makeVStateReportEvent(server.GetPulse().PulseNumber, objectRef, stateID, rawWalletState)

	server.WaitIdleConveyor()

	server.SendMessage(ctx, msg)

	server.WaitActiveThenIdleConveyor()

	server.IncrementPulse(ctx)
	server.WaitActiveThenIdleConveyor()

	// generate new state since it will be changed by CallAPIAddAmount
	newRawWalletState := makeRawWalletState(t, testBalance+uint32(additionalBalance))

	callMethod := func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error) {
		// we want to change pulse during execution
		server.IncrementPulse(ctx)

		emptyResult := makeEmptyResult(t)
		return newRawWalletState, emptyResult, nil
	}

	mockExecutor(t, server, callMethod, nil)

	code, _ := server.CallAPIAddAmount(ctx, objectRef, additionalBalance)
	require.Equal(t, 200, code)

	select {
	case delegateFinishedMsg := <-gotDelegatedRequestFinished:
		callFlags := payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)

		require.Equal(t, objectRef, delegateFinishedMsg.Callee)
		require.Equal(t, payload.CTMethod, delegateFinishedMsg.CallType)
		require.Equal(t, callFlags, delegateFinishedMsg.CallFlags)

		latestState := delegateFinishedMsg.LatestState
		require.Equal(t, newRawWalletState, latestState.State)
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}

	server.WaitIdleConveyor()

	require.Equal(t, 1, countVCallResult)
}

// In this case VDelegatedRequestFinished must not contain new state since it was not changed
func TestVirtual_SendDelegatedFinished_IfPulseChanged_Without_SideEffect(t *testing.T) {
	t.Log("C4990")

	server, ctx := utils.NewServerIgnoreLogErrors(nil, t) // TODO PLAT-367 fix test to be stable and have no errors in logs
	defer server.Stop()

	var countVCallResult int
	gotDelegatedRequestFinished := make(chan *payload.VDelegatedRequestFinished, 0)
	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		if err != nil {
			return nil
		}

		switch payLoadData := pl.(type) {
		case *payload.VDelegatedRequestFinished:
			gotDelegatedRequestFinished <- payLoadData
		case *payload.VCallResult:
			countVCallResult++
		default:
			inslogger.FromContext(ctx).Infof("Going message: %T", payLoadData)
		}

		server.SendMessage(ctx, messages[0])
		return nil
	})

	testBalance := uint32(555)
	objectRef := gen.UniqueReference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)

	// send VStateReport: save wallet
	rawWalletState := makeRawWalletState(t, testBalance)
	msg := makeVStateReportEvent(server.GetPulse().PulseNumber, objectRef, stateID, rawWalletState)

	server.WaitIdleConveyor()
	server.ResetActiveConveyorFlag()

	server.SendMessage(ctx, msg)

	server.WaitActiveThenIdleConveyor()
	server.ResetActiveConveyorFlag()

	server.IncrementPulse(ctx)
	server.WaitActiveThenIdleConveyor()

	callMethod := func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error) {
		// we want to change pulse during execution
		server.IncrementPulse(ctx)

		emptyResult := makeEmptyResult(t)
		return rawWalletState, emptyResult, nil
	}

	mockExecutor(t, server, callMethod, nil)

	code, _ := server.CallAPIGetBalance(ctx, objectRef)
	require.Equal(t, 200, code)

	select {
	case delegateFinishedMsg := <-gotDelegatedRequestFinished:
		callFlags := payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated)

		require.Equal(t, objectRef, delegateFinishedMsg.Callee)
		require.Equal(t, payload.CTMethod, delegateFinishedMsg.CallType)
		require.Equal(t, callFlags, delegateFinishedMsg.CallFlags)

		require.Nil(t, delegateFinishedMsg.LatestState)
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}

	server.WaitIdleConveyor()

	require.Equal(t, 1, countVCallResult)
}

func TestVirtual_SendDelegatedFinished_IfPulseChanged_Constructor(t *testing.T) {
	t.Log("C4988")
	t.Skip("skipped until PLAT-304")
	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	testBalance := uint32(333)
	rawWalletState := makeRawWalletState(t, testBalance)
	callConstructor := func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) (objectState []byte, result []byte, err error) {
		// we want to change pulse during construction
		server.IncrementPulse(ctx)

		emptyResult := makeEmptyResult(t)
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
			fmt.Printf("Going message: %T", payLoadData)
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
