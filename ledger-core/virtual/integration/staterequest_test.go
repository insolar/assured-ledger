// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func makeVStateRequestEvent(pulseNumber pulse.Number, ref reference.Global, flags payload.StateRequestContentFlags, sender reference.Global) *message.Message {
	pl := &payload.VStateRequest{
		AsOf:             pulseNumber,
		Object:           ref,
		RequestedContent: flags,
	}

	return utils.NewRequestWrapper(pulseNumber, pl).SetSender(sender).Finalize()
}

func TestVirtual_VStateRequest(t *testing.T) {
	insrail.LogCase(t, "C4861")

	table := []struct {
		name  string
		flags payload.StateRequestContentFlags
	}{
		{name: "without flags", flags: 0},
		{name: "with RequestLatestDirtyState flag", flags: payload.RequestLatestDirtyState},
		{name: "with RequestLatestValidatedState flag", flags: payload.RequestLatestValidatedState},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			var (
				objectGlobal   = reference.NewSelf(server.RandomLocalWithPulse())
				pulseNumber    = server.GetPulse().PulseNumber
				rawWalletState = makeRawWalletState(initialBalance)
			)

			// create object
			{
				server.IncrementPulseAndWaitIdle(ctx)
				Method_PrepareObject(ctx, server, payload.Ready, objectGlobal, pulseNumber)

				pulseNumber = server.GetPulse().PulseNumber
				waitMigrate := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.IncrementPulseAndWaitIdle(ctx)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, waitMigrate)
			}

			// prepare checker
			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			{
				expectedVStateReport := &payload.VStateReport{
					Status:           payload.Ready,
					AsOf:             pulseNumber,
					Object:           objectGlobal,
					LatestDirtyState: objectGlobal,
				}
				switch test.flags {
				case payload.RequestLatestDirtyState:
					expectedVStateReport.ProvidedContent = &payload.VStateReport_ProvidedContentBody{
						LatestDirtyState: &payload.ObjectState{
							Reference: reference.Local{},
							State:     rawWalletState,
							Class:     testwallet.ClassReference,
						},
					}
				case payload.RequestLatestValidatedState:
					expectedVStateReport.ProvidedContent = &payload.VStateReport_ProvidedContentBody{
						LatestValidatedState: &payload.ObjectState{
							Reference: reference.Local{},
							State:     rawWalletState,
							Class:     testwallet.ClassReference,
						},
					}
				}
				typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
					assert.Equal(t, expectedVStateReport, report)
					return false
				})
			}

			msg := makeVStateRequestEvent(pulseNumber, objectGlobal, test.flags, server.JetCoordinatorMock.Me())
			server.SendMessage(ctx, msg)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))

			require.Equal(t, 1, typedChecker.VStateReport.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_VStateRequest_Unknown(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4863")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		pn           = server.GetPulse().PulseNumber
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, &payload.VStateReport{
			Status: payload.Missing,
			AsOf:   server.GetPrevPulse().PulseNumber,
			Object: objectGlobal,
		}, report)

		return false
	})

	server.IncrementPulse(ctx)

	countBefore := server.PublisherMock.GetCount()
	msg := makeVStateRequestEvent(pn, objectGlobal, payload.RequestLatestDirtyState, server.JetCoordinatorMock.Me())
	server.SendMessage(ctx, msg)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}
}
