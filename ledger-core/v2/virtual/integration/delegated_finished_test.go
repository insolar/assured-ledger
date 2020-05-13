// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/builtin/foundation"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func makeEmptyResult(t *testing.T) []byte {
	var ret0 error
	ser := common.NewCBORSerializer()
	var emptyResult []byte
	err := ser.Serialize(foundation.Result{Returns: []interface{}{ret0}}, &emptyResult)
	require.NoError(t, err)

	return emptyResult
}

type callMethodFunc = func(ctx context.Context, callContext *insolar.LogicCallContext, code reference.Global, data []byte, method string, args insolar.Arguments) (newObjectState []byte, methodResults insolar.Arguments, err error)

func mockExecutor(t *testing.T, server *utils.Server, callMethod callMethodFunc) {
	executorMock := testutils.NewMachineLogicExecutorMock(t)
	executorMock.CallMethodMock.Set(callMethod)
	manager := executor.NewManager()
	err := manager.RegisterExecutor(insolar.MachineTypeBuiltin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(manager)

	cacheMock := descriptor.NewCacheMock(t)
	server.ReplaceCache(cacheMock)
	cacheMock.ByPrototypeRefMock.Return(
		descriptor.NewPrototype(gen.Reference(), gen.ID(), gen.Reference()),
		descriptor.NewCode(nil, insolar.MachineTypeBuiltin, gen.Reference()),
		nil,
	)
}

// 1. Send CallRequest
// 2. Change pulse in mocked executor
// 4. Since we changed pulse during execution, we expect that VDelegatedRequestFinished will be sent
// 5. Check that in VDelegatedRequestFinished new object state is stored
func TestVirtual_SendDelegatedFinished_IfPulseChanged(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	testBalance := uint32(555)
	additionalBalance := uint(133)
	objectRef := gen.Reference()

	{
		// send VStateReport: save wallet
		rawWalletState := makeRawWalletState(t, testBalance)
		stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
		msg := makeVStateReportEvent(t, objectRef, stateID, rawWalletState)
		require.NoError(t, server.AddInput(ctx, msg))

		server.IncrementPulse(ctx)
	}

	// generate new state since it will be changed by CallAPIAddAmount
	newRawWalletState := makeRawWalletState(t, testBalance+uint32(additionalBalance))

	callMethod := func(ctx context.Context, callContext *insolar.LogicCallContext, code reference.Global, data []byte, method string, args insolar.Arguments) (newObjectState []byte, methodResults insolar.Arguments, err error) {
		// we want to change pulse during execution
		server.IncrementPulse(ctx)

		emptyResult := makeEmptyResult(t)
		return newRawWalletState, emptyResult, nil
	}

	mockExecutor(t, server, callMethod)

	var countVCallResult int
	gotDelegatedRequestFinished := make(chan *payload.VDelegatedRequestFinished, 0)
	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
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
	}

	code, _ := server.CallAPIAddAmount(ctx, objectRef, additionalBalance)
	require.Equal(t, 200, code)

	select {
	case delegateFinishedMsg := <-gotDelegatedRequestFinished:
		callFlags := payload.CallRequestFlags(0)
		callFlags.SetTolerance(payload.CallTolerable)
		callFlags.SetState(payload.CallDirty)

		require.Equal(t, objectRef, delegateFinishedMsg.Callee)
		require.Equal(t, payload.CTMethod, delegateFinishedMsg.CallType)
		require.Equal(t, callFlags, delegateFinishedMsg.CallFlags)

		require.Equal(t, newRawWalletState, delegateFinishedMsg.ObjectBody)
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}

	require.Equal(t, 1, countVCallResult)
}
