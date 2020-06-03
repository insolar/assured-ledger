// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func makeVStateUnavailableEvent(pulseNumber pulse.Number, ref reference.Global, reason payload.VStateUnavailable_ReasonType) *message.Message {
	payload := &payload.VStateUnavailable{
		Lifeline: ref,
		Reason:   reason,
	}

	return utils.NewRequestWrapper(pulseNumber, payload).Finalize()
}

func TestVirtual_VStateUnavailable_NoSuchObject(t *testing.T) {
	t.Log("C4864")

	server, ctx := utils.NewServerIgnoreLogErrors(nil, t)
	defer server.Stop()

	objectRef := reference.NewSelf(server.RandomLocalWithPulse())

	reasons := []payload.VStateUnavailable_ReasonType{payload.Inactive, payload.Missing, payload.Unknown}
	for _, reason := range reasons {
		msg := makeVStateUnavailableEvent(server.GetPulse().PulseNumber, objectRef, reason)
		server.SendMessage(ctx, msg)
	}
}

func TestVirtual_VStateUnavailable_StateAlreadyExists(t *testing.T) {
	t.Log("C4865")

	server, ctx := utils.NewServerIgnoreLogErrors(nil, t)
	defer server.Stop()

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := reference.NewSelf(server.RandomLocalWithPulse())
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(server.GetPulse().PulseNumber, objectRef, stateID, rawWalletState)
		server.SendMessage(ctx, msg)
	}

	reasons := []payload.VStateUnavailable_ReasonType{payload.Inactive, payload.Missing, payload.Unknown}
	for _, reason := range reasons {
		msg := makeVStateUnavailableEvent(server.GetPulse().PulseNumber, objectRef, reason)
		server.SendMessage(ctx, msg)
	}

	checkBalance(ctx, t, server, objectRef, testBalance)
}
