// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"strings"
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
				objectGlobal     = reference.NewSelf(server.RandomLocalWithPulse())
				pulseNumber      = server.GetPulse().PulseNumber
				rawWalletState   = makeRawWalletState(initialBalance)
				waitVStateReport = make(chan struct{})
			)

			// create object
			{
				server.IncrementPulseAndWaitIdle(ctx)
				Method_PrepareObject(ctx, server, payload.Ready, objectGlobal, pulseNumber)

				pulseNumber = server.GetPulse().PulseNumber
				server.IncrementPulseAndWaitIdle(ctx)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1))
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
					waitVStateReport <- struct{}{}
					return false
				})
			}

			msg := makeVStateRequestEvent(pulseNumber, objectGlobal, test.flags, server.JetCoordinatorMock.Me())
			server.SendMessage(ctx, msg)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, waitVStateReport)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

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

func TestVirtual_VStateRequest_WhenObjectIsDeactivated(t *testing.T) {
	t.Log("C5474")
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

			server, ctx := utils.NewServerWithErrorFilter(nil, t, func(s string) bool {
				return !strings.Contains(s, "(*SMExecute).stepSaveNewObject")
			})
			defer server.Stop()

			var (
				objectGlobal     = reference.NewSelf(server.RandomLocalWithPulse())
				pulseNumber      = server.GetPulse().PulseNumber
				waitVStateReport = make(chan struct{})
			)
			server.IncrementPulse(ctx)
			p2 := server.GetPulse().PulseNumber

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
				assert.Equal(t, objectGlobal, report.Object)
				assert.Equal(t, payload.Inactive, report.Status)
				assert.Empty(t, report.ProvidedContent)
				assert.Empty(t, report.LatestDirtyState)
				assert.Empty(t, report.LatestDirtyCode)
				assert.Empty(t, report.LatestValidatedCode)
				assert.Empty(t, report.LatestValidatedState)

				waitVStateReport <- struct{}{}
				return false
			})

			// Send VStateReport with Dirty, Validated states
			{
				pl := &payload.VStateReport{
					AsOf:   pulseNumber,
					Status: payload.Inactive,
					Object: objectGlobal,
					ProvidedContent: &payload.VStateReport_ProvidedContentBody{
						LatestDirtyState:     nil,
						LatestValidatedState: nil,
					},
				}
				server.SendPayload(ctx, pl)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1))
			}

			server.IncrementPulse(ctx)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, waitVStateReport)

			// VStateRequest
			{
				payload := &payload.VStateRequest{
					AsOf:             p2,
					Object:           objectGlobal,
					RequestedContent: test.requestState,
				}
				server.SendPayload(ctx, payload)
			}

			commontestutils.WaitSignalsTimed(t, 10*time.Second, waitVStateReport)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 2, typedChecker.VStateReport.Count())
		})
	}
}
