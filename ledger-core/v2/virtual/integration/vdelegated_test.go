// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func wrapVDelegatedCallRequest(pl payload.VDelegatedCallRequest, number pulse.Number, sender reference.Global) (*message.Message, error) {
	plBytes, err := pl.Marshal()
	if err != nil {
		return nil, err
	}

	msg, err := payload.NewMessage(&payload.Meta{
		Payload:    plBytes,
		Sender:     sender,
		Receiver:   reference.Global{},
		Pulse:      number,
		ID:         nil,
		OriginHash: payload.MessageHash{},
	})
	return msg, err
}

func TestVirtual_VDelegatedCallRequest(t *testing.T) {
	t.Log("C4983")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	testBalance := uint32(500)
	objectRef := gen.UniqueReference()

	{
		// send VStateReport: save wallet
		stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
		rawWalletState := makeRawWalletState(t, testBalance)
		payloadMeta := &payload.VStateReport{
			Callee:                        objectRef,
			UnorderedPendingCount:         1,
			UnorderedPendingEarliestPulse: pulse.OfNow(),
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: stateID,
					Class:     testwalletProxy.GetClass(),
					State:     rawWalletState,
				},
			},
		}
		msg := makeDispatcherMessage(t, payloadMeta)
		require.NoError(t, server.AddInput(ctx, msg))
	}

	pl := payload.VDelegatedCallRequest{
		RequestReference: reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow() + 10)),
		Callee:           objectRef,
		CallFlags:        payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
	}
	sender := gen.UniqueReference()
	msg, err := wrapVDelegatedCallRequest(pl, server.GetPulse().PulseNumber, sender)
	require.NoError(t, err)

	requestIsDone := make(chan struct{}, 0)

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		defer func() { requestIsDone <- struct{}{} }()

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		require.NoError(t, err)

		switch pl.(type) {
		case *payload.VDelegatedCallResponse:
			callResultPl := pl.(*payload.VDelegatedCallResponse)
			require.NotEmpty(t, callResultPl.DelegationSpec)
			require.Equal(t, objectRef, callResultPl.DelegationSpec.Callee)
			require.Equal(t, sender, callResultPl.DelegationSpec.DelegateTo)
		default:
			require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VDelegatedCallResponse", pl)
		}

		return nil
	})

	server.SendMessage(ctx, msg)

	select {
	case <-requestIsDone:
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}
}
