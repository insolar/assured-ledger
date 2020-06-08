// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_VDelegatedCallRequest(t *testing.T) {
	t.Log("C4983")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		mc          = minimock.NewController(t)
		testBalance = uint32(500)
		objectRef   = gen.UniqueReference()
		sender      = gen.UniqueReference()
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VDelegatedCallResponse.Set(func(pl *payload.VDelegatedCallResponse) bool {
		require.NotEmpty(t, pl.DelegationSpec)
		require.Equal(t, objectRef, pl.DelegationSpec.Callee)
		require.Equal(t, sender, pl.DelegationSpec.DelegateTo)

		return false // no resend msg
	}).ExpectedCount(1)

	server.WaitIdleConveyor()

	{
		// send VStateReport: save wallet
		stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
		rawWalletState := makeRawWalletState(t, testBalance)
		payloadMeta := &payload.VStateReport{
			Status:                        payload.Ready,
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
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, payloadMeta).Finalize()
		server.SendMessage(ctx, msg)
	}

	server.WaitActiveThenIdleConveyor()

	{
		// send VDelegatedCall
		pl := payload.VDelegatedCallRequest{
			RequestReference: reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow() + 10)),
			Callee:           objectRef,
			CallFlags:        payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
		}

		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(sender).Finalize()
		server.SendMessage(ctx, msg)
	}

	server.PublisherMock.WaitCount(1, 10*time.Second)
	mc.Finish()
}
