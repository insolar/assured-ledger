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

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

// 1. Send CallRequest
// 2. Change pulse in mocked executor
// 4. Since we changed pulse during execution, we expect that VStateReport will be sent
// 5. Check that in VStateReport new object state is stored
func TestVirtual_SendVStateReport_IfPulseChanged(t *testing.T) {
	t.Skip("skipped until PLAT-314")
	t.Log("C4934")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	testBalance := uint32(555)
	additionalBalance := uint(133)
	objectRef := gen.UniqueReference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		rawWalletState := makeRawWalletState(t, testBalance)
		msg := makeVStateReportEvent(t, objectRef, stateID, rawWalletState)
		require.NoError(t, server.AddInput(ctx, msg))
	}

	// generate new state since it will be changed by CallAPIAddAmount
	newRawWalletState := makeRawWalletState(t, testBalance+uint32(additionalBalance))

	callMethod := func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error) {
		// we want to change pulse during execution
		server.IncrementPulse(ctx)

		emptyResult := makeEmptyResult(t)
		return newRawWalletState, emptyResult, nil
	}

	mockExecutor(t, server, callMethod, nil)

	var (
		countVStateReport int
	)
	gotVStateReport := make(chan *payload.VStateReport, 0)
	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		if err != nil {
			return nil
		}

		switch payLoadData := pl.(type) {
		case *payload.VStateReport:
			countVStateReport++
			gotVStateReport <- payLoadData
		case *payload.VCallResult:
		default:
			fmt.Printf("Going message: %T", payLoadData)
		}

		server.SendMessage(ctx, messages[0])
		return nil
	})

	code, _ := server.CallAPIAddAmount(ctx, objectRef, additionalBalance)
	require.Equal(t, 200, code)

	select {
	case _ = <-gotVStateReport:
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}

	require.Equal(t, 1, countVStateReport)
}
