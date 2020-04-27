// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

func makeVStateUnavailableEvent(t *testing.T, ref insolar.Reference) *statemachine.DispatcherMessage {
	payLoadMeta := &payload.VStateUnavailable{
		Reference: ref,
		Reason:    payload.Inactive,
	}
	return makeDispatcherMessage(t, payLoadMeta)
}

func TestVirtual_VStateUnavailable_NoSuchObject(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])
		return nil
	}

	objectRef := reference.NewGlobalSelf(server.RandomLocalWithPulse())
	// send VStateRequest: save wallet
	msg := makeVStateUnavailableEvent(t, objectRef)
	require.NoError(t, server.AddInput(msg))
}
