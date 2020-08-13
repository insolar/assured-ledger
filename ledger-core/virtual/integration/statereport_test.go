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

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func makeVStateReportWithState(
	objectRef reference.Global,
	stateStatus payload.VStateReport_StateStatus,
	state *payload.ObjectState,
	asOf payload.PulseNumber,
) *payload.VStateReport {
	res := payload.VStateReport{
		Status: stateStatus,
		Object: objectRef,
		AsOf:   asOf,
	}
	if state != nil {
		res.ProvidedContent = &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: state,
		}
	}
	return &res

}

func makeRawWalletState(balance uint32) []byte {
	return insolar.MustSerialize(testwallet.Wallet{
		Balance: balance,
	})
}

func TestVirtual_VStateReport_StateAlreadyExists(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4865")

	table := []struct {
		name   string
		status payload.VStateReport_StateStatus
	}{
		{name: "ready state", status: payload.Ready},
		{name: "inactive state", status: payload.Inactive},
		{name: "missing state", status: payload.Missing},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			var (
				initState    = []byte("init state")
				initRef      = server.RandomLocalWithPulse()
				prevPulse    = server.GetPulse().PulseNumber
				objectGlobal = server.RandomGlobalWithPulse()
				class        = server.RandomGlobalWithPulse()
			)

			// send first VStateReport
			{
				server.IncrementPulse(ctx)

				pl := &payload.VStateReport{
					Status: payload.Ready,
					Object: objectGlobal,
					AsOf:   prevPulse,
					ProvidedContent: &payload.VStateReport_ProvidedContentBody{
						LatestDirtyState: &payload.ObjectState{
							Reference: initRef,
							Class:     class,
							State:     initState,
						},
						LatestValidatedState: &payload.ObjectState{
							Reference: initRef,
							Class:     class,
							State:     initState,
						},
					},
				}
				waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, pl)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, waitReport)
			}

			// add checker
			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			{
				typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
					assert.NotNil(t, report.ProvidedContent)
					assert.Equal(t, payload.Ready, report.Status)
					assert.Equal(t, initRef, report.ProvidedContent.LatestDirtyState.Reference)
					assert.Equal(t, initRef, report.ProvidedContent.LatestValidatedState.Reference)
					assert.Equal(t, initState, report.ProvidedContent.LatestDirtyState.State)
					assert.Equal(t, initState, report.ProvidedContent.LatestValidatedState.State)
					return false
				})
			}

			// send second VStateReport
			{
				pl := &payload.VStateReport{
					Status: testCase.status,
					Object: objectGlobal,
					AsOf:   prevPulse,
				}
				if testCase.status == payload.Ready {
					pl.ProvidedContent = &payload.VStateReport_ProvidedContentBody{
						LatestDirtyState: &payload.ObjectState{
							Reference: server.RandomLocalWithPulse(),
							Class:     class,
							State:     []byte("new state"),
						},
					}
				}
				waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, pl)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, waitReport)
			}

			// increment pulse and check VStateReport
			{
				server.IncrementPulse(ctx)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
				assert.Equal(t, 1, typedChecker.VStateReport.Count())
			}

			mc.Finish()
		})
	}
}
