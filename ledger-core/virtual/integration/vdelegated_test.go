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
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
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
	typedChecker.VDelegatedCallResponse.Set(func(pl *payload.VDelegatedCallResponse) bool {
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
		payloadMeta := &payload.VStateReport{
			Status:                        payload.Ready,
			Object:                        objectRef,
			AsOf:                          prevPulse,
			UnorderedPendingCount:         1,
			UnorderedPendingEarliestPulse: prevPulse,
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
			CallOutgoing: outgoing,
			CallIncoming: incoming,
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

func TestVirtual_VDelegatedRequestFinished_WithoutToken(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5146")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	server.IncrementPulse(ctx)

	var (
		mc        = minimock.NewController(t)
		prevPulse = server.GetPulse().PulseNumber
		objectRef = server.RandomGlobalWithPulse()
		sender    = server.RandomGlobalWithPulse()
		class     = server.RandomGlobalWithPulse()

		outgoing = server.BuildRandomOutgoingWithPulse()
		incoming = reference.NewRecordOf(objectRef, outgoing.GetLocal())
	)

	server.IncrementPulseAndWaitIdle(ctx)

	// send VStateReport with ordered pending
	{
		stateID := gen.UniqueLocalRefWithPulse(prevPulse)
		payloadMeta := &payload.VStateReport{
			Status:                      payload.Ready,
			Object:                      objectRef,
			AsOf:                        prevPulse,
			OrderedPendingCount:         1,
			OrderedPendingEarliestPulse: prevPulse,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: stateID,
					Class:     class,
					State:     []byte("init state"),
				},
			},
		}
		wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, payloadMeta)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, wait)
	}

	node := server.RandomGlobalWithPulse()
	server.JetCoordinatorMock.MeMock.Return(node)
	server.JetCoordinatorMock.QueryRoleMock.Return([]reference.Global{sender}, nil)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		// bug , VDelegatedRequestFinished without token updated state
		assert.Equal(t, []byte("new state"), report.ProvidedContent.LatestDirtyState.State)
		return false
	})
	typedChecker.VDelegatedCallResponse.Set(func(pl *payload.VDelegatedCallResponse) bool {
		require.NotEmpty(t, pl.ResponseDelegationSpec)
		require.Equal(t, objectRef, pl.ResponseDelegationSpec.Callee)
		require.Equal(t, sender, pl.ResponseDelegationSpec.DelegateTo)
		return false // no resend msg
	})

	// send VDelegatedCallRequest
	{
		pl := payload.VDelegatedCallRequest{
			CallOutgoing: outgoing,
			CallIncoming: incoming,
			Callee:       objectRef,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		}
		wait := server.Journal.WaitStopOf(&handlers.SMVDelegatedCallRequest{}, 1)
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(sender).Finalize()
		server.SendMessage(ctx, msg)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, wait)
	}

	// send VDelegatedRequestFinished
	{
		pl := payload.VDelegatedRequestFinished{
			CallType:     payload.CTMethod,
			Callee:       objectRef,
			CallOutgoing: outgoing,
			CallIncoming: incoming,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			LatestState: &payload.ObjectState{
				State: []byte("new state"),
			},
		}
		wait := server.Journal.WaitStopOf(&handlers.SMVDelegatedRequestFinished{}, 1)
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(sender).Finalize()
		server.SendMessage(ctx, msg)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, wait)
	}

	server.IncrementPulse(ctx)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))

	mc.Finish()
}
