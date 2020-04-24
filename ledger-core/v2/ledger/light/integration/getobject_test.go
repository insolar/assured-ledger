// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

func Test_GetObject_PassingRequestID(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)
	cfg := DefaultLightConfig()
	s, err := NewServer(ctx, cfg, nil)
	require.NoError(t, err)
	defer s.Stop()

	// First pulse goes in storage then interrupts.
	s.SetPulse(ctx)

	t.Run("Incoming request can't have several different results", func(t *testing.T) {
		var firstReqID insolar.ID
		// Creating root reason request.
		{
			msg, _ := MakeSetIncomingRequest(
				gen.ID(),
				gen.IDWithPulse(s.Pulse()),
				insolar.ID{},
				true,
				true,
				"first",
			)
			rep := SendMessage(ctx, s, &msg)
			RequireNotError(rep)
			firstReqID = rep.(*payload.RequestInfo).RequestID

			p, _ := CallActivateObject(ctx, s, firstReqID)
			RequireNotError(p)
		}

		s.SetPulse(ctx)

		var secondReqID insolar.ID
		var thirdreqID insolar.ID
		// Register second request
		{
			msg, _ := MakeSetIncomingRequest(
				firstReqID,
				firstReqID,
				insolar.ID{},
				false,
				true,
				"second",
			)
			rep := SendMessage(ctx, s, &msg)
			RequireNotError(rep)
			secondReqID = rep.(*payload.RequestInfo).RequestID

			msg, _ = MakeSetIncomingRequest(
				firstReqID,
				firstReqID,
				insolar.ID{},
				false,
				true,
				"third",
			)
			rep = SendMessage(ctx, s, &msg)
			RequireNotError(rep)
			thirdreqID = rep.(*payload.RequestInfo).RequestID
		}

		s.SetPulse(ctx)
		// Call get object with second ID (while first isn't closed)
		{
			lifelinePL, statePL := CallGetObject(ctx, s, firstReqID, &thirdreqID)
			RequireNotError(lifelinePL)
			RequireNotError(statePL)

			lifeline := lifelinePL.(*payload.Index)
			require.NotNil(t, lifeline.EarliestRequestID)
			require.Equal(t, secondReqID, *lifeline.EarliestRequestID)
		}
	})
}
