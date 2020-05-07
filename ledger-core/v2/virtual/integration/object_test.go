// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func TestInitViaCTMethod(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	// Call method on non-existent object, expect calling of VStateRequest
	pl := payload.VCallRequest{
		Polymorph: uint32(payload.TypeVCallRequest),
		CallType:  payload.CTMethod,
		Callee:    reference.NewSelf(server.RandomLocalWithPulse()),
	}
	msg, err := wrapVCallRequest(server.GetPulse().PulseNumber, pl)
	require.NoError(t, err)

	requestIsDone := make(chan struct{}, 0)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		defer func() { requestIsDone <- struct{}{} }()

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		require.NoError(t, err)

		switch request := pl.(type) {
		case *payload.VStateRequest:
			for _, flag := range []payload.StateRequestContentFlags{payload.RequestLatestValidatedState, payload.RequestLatestDirtyState,
				payload.RequestMutableQueue, payload.RequestImmutableQueue} {
				assert.True(t, request.RequestedContent.Contains(flag))
			}
		default:
			require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
		}

		return nil
	}

	server.SendMessage(ctx, msg)

	select {
	case <-requestIsDone:
	case <-time.After(3 * time.Second):
		require.Failf(t, "", "timeout")
	}
}
