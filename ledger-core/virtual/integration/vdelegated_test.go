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
		objectRef   = server.RandomGlobalWithPulse()
		sender      = server.JetCoordinatorMock.Me()

		outgoing = server.BuildRandomOutgoingWithPulse()
		incoming = reference.NewRecordOf(objectRef, outgoing.GetLocal())
	)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	typedChecker.VDelegatedCallResponse.Set(func(pl *rms.VDelegatedCallResponse) bool {
		require.NotEmpty(t, pl.ResponseDelegationSpec)
		assert.Equal(t, objectRef, pl.ResponseDelegationSpec.Callee.GetValue())
		assert.Equal(t, sender, pl.ResponseDelegationSpec.DelegateTo.GetValue())

		return false // no resend msg
	})

	{
		report := server.StateReportBuilder().Object(objectRef).Ready().
			Memory(makeRawWalletState(testBalance)).Class(testwalletProxy.GetClass()).
			UnorderedPendings(1).Report()

		server.IncrementPulse(ctx)

		waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, &report)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, waitReport)
	}

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
