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
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

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
				object         = server.RandomGlobalWithPulse()
				pulseNumber    = server.GetPulse().PulseNumber
				rawWalletState = makeRawWalletState(initialBalance)
			)

			// create object
			{
				server.IncrementPulseAndWaitIdle(ctx)
				Method_PrepareObject(ctx, server, payload.StateStatusReady, object, pulseNumber)

				pulseNumber = server.GetPulse().PulseNumber
				waitMigrate := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.IncrementPulseAndWaitIdle(ctx)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, waitMigrate)
			}

			// prepare checker
			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			{
				expectedVStateReport := &payload.VStateReport{
					Status:           payload.StateStatusReady,
					AsOf:             pulseNumber,
					Object:           object,
					LatestDirtyState: object,
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

			pl := &payload.VStateRequest{
				AsOf:             pulseNumber,
				Object:           object,
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

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			assert.Equal(t, &payload.VStateReport{
				AsOf:   pn,
				Object: object,
				Status: payload.StateStatusMissing,
			}, report)

			return false
		})
	}

	{
		pl := &payload.VStateRequest{
			AsOf:             pn,
			Object:           object,
			RequestedContent: payload.RequestLatestDirtyState,
		}

		server.SendPayload(ctx, pl)
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
}

func TestVirtual_VStateRequest_WhenObjectIsDeactivated(t *testing.T) {
	insrail.LogCase(t, "C5474")
	table := []struct {
		name         string
		requestState payload.StateRequestContentFlags
	}{
		{name: "Request_State = dirty",
			requestState: payload.RequestLatestDirtyState},
		{name: "Request_State = validated",
			requestState: payload.RequestLatestValidatedState},
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
				vStateReport = &payload.VStateReport{
					AsOf:            pulseNumber,
					Status:          payload.StateStatusInactive,
					Object:          objectGlobal,
					ProvidedContent: nil,
				}
			)
			server.IncrementPulse(ctx)
			p2 := server.GetPulse().PulseNumber

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
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
				payload := &payload.VStateRequest{
					AsOf:             p2,
					Object:           objectGlobal,
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
