// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_VDelegatedCallRequest(t *testing.T) {
	defer commontestutils.LeakTester(t)

//	t.Log("C4983")
	instestlogger.LogCase(t, "C4983")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	server.IncrementPulse(ctx)

	var (
		mc          = minimock.NewController(t)
		testBalance = uint32(500)
		objectRef   = gen.UniqueGlobalRef()
		sender      = server.JetCoordinatorMock.Me()
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VDelegatedCallResponse.Set(func(pl *payload.VDelegatedCallResponse) bool {
		require.NotEmpty(t, pl.ResponseDelegationSpec)
		require.Equal(t, objectRef, pl.ResponseDelegationSpec.Callee)
		require.Equal(t, sender, pl.ResponseDelegationSpec.DelegateTo)

		return false // no resend msg
	})

	server.WaitIdleConveyor()

	{
		// send VStateReport: save wallet
		stateID := gen.UniqueLocalRefWithPulse(server.GetPulse().PulseNumber)
		rawWalletState := makeRawWalletState(testBalance)
		payloadMeta := &payload.VStateReport{
			Status:                        payload.Ready,
			Object:                        objectRef,
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
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, payloadMeta).SetSender(sender).Finalize()
		server.SendMessage(ctx, msg)
	}

	server.WaitActiveThenIdleConveyor()

	{
		// send VDelegatedCall
		pl := payload.VDelegatedCallRequest{
			CallOutgoing: reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() + 10)),
			Callee:       objectRef,
			CallFlags:    payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
		}

		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(sender).SetSender(server.JetCoordinatorMock.Me()).Finalize()
		server.SendMessage(ctx, msg)
	}

	server.PublisherMock.WaitCount(1, 10*time.Second)
	mc.Finish()

	{
		assert.Equal(t, 1, typedChecker.VDelegatedCallResponse.Count())
	}
}
