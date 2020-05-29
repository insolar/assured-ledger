// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/contract/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

func makeDispatcherMessage(t *testing.T, payLoadMeta payload.Payload) *statemachine.DispatcherMessage {
	rawPayLoad, err := payLoadMeta.Marshal()
	require.NoError(t, err)

	return &statemachine.DispatcherMessage{
		MessageMeta: message.Metadata{},
		PayloadMeta: &payload.Meta{Payload: rawPayLoad},
	}
}

func makeVStateReportEvent(t *testing.T, objectRef reference.Global, stateRef reference.Local, rawState []byte) *statemachine.DispatcherMessage {
	class := testwalletProxy.GetClass()
	payLoadMeta := &payload.VStateReport{
		Callee: objectRef,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: stateRef,
				Class:     class,
				State:     rawState,
			},
		},
	}
	return makeDispatcherMessage(t, payLoadMeta)
}

func makeRawWalletState(t *testing.T, balance uint32) []byte {
	wallet := testwallet.Wallet{Balance: balance}
	ser := common.NewCBORSerializer()
	var buf []byte
	err := ser.Serialize(wallet, &buf)
	require.NoError(t, err)

	return buf
}

func checkBalance(ctx context.Context, t *testing.T, server *utils.Server, objectRef reference.Global, testBalance uint32) {
	code, byteBuffer := server.CallAPIGetBalance(ctx, objectRef)
	require.Equal(t, 200, code, string(byteBuffer))

	response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
	require.NoError(t, err)
	require.Empty(t, response.Err)
	require.NotEmpty(t, response.TraceID)
	require.Equal(t, uint(testBalance), response.Amount)
}

func TestVirtual_VStateReport_HappyPath(t *testing.T) {
	t.Log("C4866")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])
		return nil
	})

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.UniqueReference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(t, objectRef, stateID, rawWalletState)
		require.NoError(t, server.AddInput(ctx, msg))
	}

	checkBalance(ctx, t, server, objectRef, testBalance)
}

func TestVirtual_VStateReport_TwoStateReports(t *testing.T) {
	t.Log("C4919")

	server, ctx := utils.NewServerIgnoreLogErrors(nil, t) // TODO PLAT-367 fix test to be stable and have no errors in logs
	defer server.Stop()

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])
		return nil
	})

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.UniqueReference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(t, objectRef, stateID, rawWalletState)
		require.NoError(t, server.AddInput(ctx, msg))
	}

	checkBalance(ctx, t, server, objectRef, testBalance)
	newStateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: one more time to simulate rewrite
		msg := makeVStateReportEvent(t, objectRef, newStateID, makeRawWalletState(t, 444))
		require.NoError(t, server.AddInput(ctx, msg))
	}

	checkBalance(ctx, t, server, objectRef, testBalance)
}
