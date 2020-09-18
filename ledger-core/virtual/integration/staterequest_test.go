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

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_VStateRequest(t *testing.T) {
	insrail.LogCase(t, "C4861")

	table := []struct {
		name  string
		flags rms.StateRequestContentFlags
	}{
		{name: "without flags", flags: 0},
		{name: "with RequestLatestDirtyState flag", flags: rms.RequestLatestDirtyState},
		{name: "with RequestLatestValidatedState flag", flags: rms.RequestLatestValidatedState},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			var (
				object         = server.RandomGlobalWithPulse()
				pulseNumber    = server.GetPulse().PulseNumber
				rawWalletState = makeRawWalletState(initialBalance)
			)

			// create object
			{
				server.IncrementPulseAndWaitIdle(ctx)
				Method_PrepareObject(ctx, server, rms.StateStatusReady, object, pulseNumber)

				pulseNumber = server.GetPulse().PulseNumber
				waitMigrate := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.IncrementPulseAndWaitIdle(ctx)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, waitMigrate)
			}

			// prepare checker
			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			{
				// vStateID := reference.NewRecordOf(object, server.RandomLocalWithPulse())
				// dStateID := reference.NewRecordOf(object, server.RandomLocalWithPulse())

				expectedVStateReport := &rms.VStateReport{
					Status:           rms.StateStatusReady,
					AsOf:             pulseNumber,
					Object:           rms.NewReference(object),
					LatestDirtyState: rms.NewReference(object),
				}
				switch test.flags {
				case rms.RequestLatestDirtyState:
					expectedVStateReport.ProvidedContent = &rms.VStateReport_ProvidedContentBody{
						LatestDirtyState: &rms.ObjectState{
							Reference: rms.NewReference(reference.Global{}),
							State:     rms.NewBytes(rawWalletState),
							Class:     rms.NewReference(testwallet.ClassReference),
						},
					}

				case rms.RequestLatestValidatedState:
					expectedVStateReport.ProvidedContent = &rms.VStateReport_ProvidedContentBody{
						LatestValidatedState: &rms.ObjectState{
							Reference: rms.NewReference(reference.Global{}),
							State:     rms.NewBytes(rawWalletState),
							Class:     rms.NewReference(testwallet.ClassReference),
						},
					}
				}
				typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
					if c := report.ProvidedContent; c != nil {
						if c.LatestValidatedState != nil {
							c.LatestValidatedState.Reference = rms.NewReference(reference.Global{})
						}
						if c.LatestDirtyState != nil {
							c.LatestDirtyState.Reference = rms.NewReference(reference.Global{})
						}
					}

					utils.AssertVStateReportsEqual(t, expectedVStateReport, report)
					return false
				})
			}

			pl := &rms.VStateRequest{
				AsOf:             pulseNumber,
				Object:           rms.NewReference(object),
				RequestedContent: test.flags,
			}
			server.SendPayload(ctx, pl)

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
		pn     = server.GetPulse().PulseNumber
		object = server.RandomGlobalWithPulse()
	)

	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	{
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			assert.Equal(t, &rms.VStateReport{
				AsOf:   pn,
				Object: rms.NewReference(object),
				Status: rms.StateStatusMissing,
			}, report)

			return false
		})
	}

	{
		pl := &rms.VStateRequest{
			AsOf:             pn,
			Object:           rms.NewReference(object),
			RequestedContent: rms.RequestLatestDirtyState,
		}

		server.SendPayload(ctx, pl)
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
}

func TestVirtual_VStateRequest_WhenObjectIsDeactivated(t *testing.T) {
	insrail.LogCase(t, "C5474")
	table := []struct {
		name         string
		requestState rms.StateRequestContentFlags
	}{
		{name: "Request_State = dirty",
			requestState: rms.RequestLatestDirtyState},
		{name: "Request_State = validated",
			requestState: rms.RequestLatestValidatedState},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			var (
				objectGlobal = server.RandomGlobalWithPulse()
				pulseNumber  = server.GetPulse().PulseNumber
				vStateReport = &rms.VStateReport{
					AsOf:            pulseNumber,
					Status:          rms.StateStatusInactive,
					Object:          rms.NewReference(objectGlobal),
					ProvidedContent: nil,
				}
			)
			server.IncrementPulse(ctx)
			p2 := server.GetPulse().PulseNumber

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
				vStateReport.AsOf = p2
				assert.Equal(t, vStateReport, report)
				return false
			})

			// Send VStateReport
			{
				server.SendPayload(ctx, vStateReport)
				reportSend := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, reportSend)
			}

			server.IncrementPulse(ctx)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))

			// VStateRequest
			{
				payload := &rms.VStateRequest{
					AsOf:             p2,
					Object:           rms.NewReference(objectGlobal),
					RequestedContent: test.requestState,
				}
				server.SendPayload(ctx, payload)
			}
			commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 2))
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 2, typedChecker.VStateReport.Count())
		})
	}
}
