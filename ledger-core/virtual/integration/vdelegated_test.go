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
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_VDelegatedCallRequest(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4983")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	server.IncrementPulse(ctx)

	var (
		mc          = minimock.NewController(t)
		testBalance = uint32(500)
		prevPulse   = server.GetPulse().PulseNumber
		objectRef   = server.RandomGlobalWithPulse()
		sender      = server.JetCoordinatorMock.Me()

		outgoing = server.BuildRandomOutgoingWithPulse()
		incoming = reference.NewRecordOf(objectRef, outgoing.GetLocal())
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VDelegatedCallResponse.Set(func(pl *rms.VDelegatedCallResponse) bool {
		require.NotEmpty(t, pl.ResponseDelegationSpec)
		require.Equal(t, objectRef, pl.ResponseDelegationSpec.Callee)
		require.Equal(t, sender, pl.ResponseDelegationSpec.DelegateTo)

		return false // no resend msg
	})

	server.IncrementPulseAndWaitIdle(ctx)

	{
		// send VStateReport: save wallet
		stateID := gen.UniqueLocalRefWithPulse(prevPulse)
		rawWalletState := makeRawWalletState(testBalance)
		payloadMeta := &rms.VStateReport{
			Status:                        rms.StateStatusReady,
			Object:                        rms.NewReference(objectRef),
			AsOf:                          prevPulse,
			UnorderedPendingCount:         1,
			UnorderedPendingEarliestPulse: prevPulse,
			ProvidedContent: &rms.VStateReport_ProvidedContentBody{
				LatestDirtyState: &rms.ObjectState{
					Reference: rms.NewReferenceLocal(stateID),
					Class:     rms.NewReference(testwalletProxy.GetClass()),
					State:     rms.NewBytes(rawWalletState),
				},
			},
		}
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, payloadMeta).SetSender(sender).Finalize()
		server.SendMessage(ctx, msg)
	}

	server.WaitActiveThenIdleConveyor()

	{
		// send VDelegatedCall
		pl := rms.VDelegatedCallRequest{
			CallOutgoing: rms.NewReference(outgoing),
			CallIncoming: rms.NewReference(incoming),
			Callee:       rms.NewReference(objectRef),
			CallFlags:    rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
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
