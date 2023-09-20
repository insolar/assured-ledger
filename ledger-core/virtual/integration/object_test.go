package integration

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestInitViaCTMethod(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4867")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	typedChecker.VStateRequest.Set(func(request *rms.VStateRequest) bool {
		for _, flag := range []rms.StateRequestContentFlags{
			rms.RequestLatestValidatedState,
			rms.RequestLatestDirtyState,
			rms.RequestOrderedQueue,
			rms.RequestUnorderedQueue,
		} {
			assert.True(t, request.RequestedContent.Contains(flag))
		}
		return false
	})

	{
		// Call method on non-existent object, expect calling of VStateRequest
		pl := utils.GenerateVCallRequestMethod(server)
		server.SendPayload(ctx, pl)
	}

	waitTime := server.GetPulseTime()

	// potentially failing test, if execution would sleep for some time before that check
	if server.PublisherMock.WaitCount(1, waitTime/20) {
		require.Failf(t, "", "SM Object needs to wait until sm.waitGetStateUntil (potentialy failing)")
	}

	if !server.PublisherMock.WaitCount(1, waitTime/5) {
		require.Failf(t, "", "timeout")
	}
}
