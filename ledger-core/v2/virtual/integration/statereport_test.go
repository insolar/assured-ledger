// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/contract/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

func makeDispatcherMessage(t *testing.T, payLoadMeta payload.Payload) *statemachine.DispatcherMessage {
	rawPayLoad, err := payload.Marshal(payLoadMeta)
	require.NoError(t, err)

	return &statemachine.DispatcherMessage{
		MessageMeta: message.Metadata{},
		PayloadMeta: &payload.Meta{Payload: rawPayLoad},
	}
}

func makeVStateReportEvent(t *testing.T, ref insolar.Reference, rawState []byte) *statemachine.DispatcherMessage {
	payLoadMeta := &payload.VStateReport{
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: ref,
				Prototype: testwalletProxy.GetPrototype(),
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

func checkBalance(t *testing.T, server *utils.Server, objectRef reference.Global, testBalance uint32) {
	code, byteBuffer := server.CallAPIGetBalance(objectRef)
	require.Equal(t, 200, code, string(byteBuffer))

	response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
	require.NoError(t, err)
	require.Empty(t, response.Err)
	require.NotEmpty(t, response.TraceID)
	require.Equal(t, uint(testBalance), response.Amount)
}

func TestVirtual_VStateReport_HappyPath(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])
		return nil
	}

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.Reference()
	{
		// send VStateRequest: save wallet
		msg := makeVStateReportEvent(t, objectRef, rawWalletState)
		require.NoError(t, server.AddInput(msg))
	}

	checkBalance(t, server, objectRef, testBalance)
}

func TestVirtual_VStateReport_TwoStateReports(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])
		return nil
	}

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.Reference()
	{
		// send VStateRequest: save wallet
		msg := makeVStateReportEvent(t, objectRef, rawWalletState)
		require.NoError(t, server.AddInput(msg))
	}

	checkBalance(t, server, objectRef, testBalance)

	{
		// send VStateRequest: one more time to simulate rewrite
		msg := makeVStateReportEvent(t, objectRef, makeRawWalletState(t, 444))
		require.NoError(t, server.AddInput(msg))
	}

	checkBalance(t, server, objectRef, testBalance)
}
