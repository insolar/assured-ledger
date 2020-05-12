// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

// 1. Send CallRequest
// 2. Since there is no state SM send VSateRequest
// 3. We catch VSateRequest. Change pulse. Send VStateReport
// 4. Since we changed pulse during execution, we expect that VDelegatedRequestFinished will be sent
// 5. Check that in VDelegatedRequestFinished new object state is stored
func TestVirtual_SendDelegatedFinished_IfPulseChanged(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	vStateRequestMsg := make(chan *message.Message, 1)

	currentPN := server.GetPulse().PulseNumber

	objectRef := gen.Reference()
	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	go func() {
		// Here we simulate pulse change during execution
		_ = <-vStateRequestMsg

		server.IncrementPulse(ctx)

		stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
		{
			// send VStateReport: save wallet
			msg := makeVStateReportEvent(t, objectRef, stateID, rawWalletState)
			require.NoError(t, server.AddInputWithPulse(ctx, msg, currentPN))
		}
	}()

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
		case *payload.VStateRequest:
			// notify about coming of VDelegatedRequestFinished
			vStateRequestMsg <- messages[0]
			return nil
		default:
		}

		server.SendMessage(ctx, messages[0])
		return nil
	}

	additionalBalance := uint(133)
	server.CallAPIAddAmount(objectRef, additionalBalance)

	select {
	case delegateFinishedMsg := <-gotDelegatedRequestFinished:
		callFlags := payload.CallRequestFlags(0)
		callFlags.SetTolerance(payload.CallTolerable)
		callFlags.SetState(payload.CallDirty)

		require.Equal(t, objectRef, delegateFinishedMsg.Callee)
		require.Equal(t, payload.CTMethod, delegateFinishedMsg.CallType)
		require.Equal(t, callFlags, delegateFinishedMsg.CallFlags)

		// generate new state since it was changed by CallAPIAddAmount
		newRawWalletState := makeRawWalletState(t, testBalance+uint32(additionalBalance))
		require.Equal(t, newRawWalletState, delegateFinishedMsg.ObjectBody)

	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}
}
