// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestInitViaCTMethod(t *testing.T) {
	t.Log("C4867")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	// Call method on non-existent object, expect calling of VStateRequest
	pl := payload.VCallRequest{
		CallType:  payload.CTMethod,
		Callee:    reference.NewSelf(server.RandomLocalWithPulse()),
		CallFlags: payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
	}

	requestIsDone := make(chan struct{}, 0)

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		defer func() { requestIsDone <- struct{}{} }()

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		require.NoError(t, err)

		switch request := pl.(type) {
		case *payload.VStateRequest:
			for _, flag := range []payload.StateRequestContentFlags{payload.RequestLatestValidatedState, payload.RequestLatestDirtyState,
				payload.RequestOrderedQueue, payload.RequestUnorderedQueue} {
				assert.True(t, request.RequestedContent.Contains(flag))
			}
		default:
			require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
		}

		return nil
	})

	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()
	server.SendMessage(ctx, msg)

	select {
	case <-requestIsDone:
		require.Failf(t, "", "SM Object needs to wait until sm.waitGetStateUntil")
	case <-time.After(500 * time.Millisecond):
	}

	select {
	case <-requestIsDone:
	case <-time.After(1 * time.Second):
		require.Failf(t, "", "timeout")
	}
}
