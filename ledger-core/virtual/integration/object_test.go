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

	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
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

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateRequest.Set(func(request *payload.VStateRequest) bool {
		for _, flag := range []payload.StateRequestContentFlags{
			payload.RequestLatestValidatedState,
			payload.RequestLatestDirtyState,
			payload.RequestOrderedQueue,
			payload.RequestUnorderedQueue,
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

	// potentially failing test, if execution would sleep for some time before that check
	if server.PublisherMock.WaitCount(1, 500*time.Millisecond) {
		require.Failf(t, "", "SM Object needs to wait until sm.waitGetStateUntil (potentialy failing)")
	}

	if !server.PublisherMock.WaitCount(1, 1*time.Second) {
		require.Failf(t, "", "timeout")
	}
}
