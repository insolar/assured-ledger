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
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func makeVStateReportWithState(
	objectRef reference.Global,
	stateStatus rms.VStateReport_StateStatus,
	state *rms.ObjectState,
	asOf rms.PulseNumber,
) *rms.VStateReport {
	res := rms.VStateReport{
		Status: stateStatus,
		Object: rms.NewReference(objectRef),
		AsOf:   asOf,
	}
	if state != nil {
		res.ProvidedContent = &rms.VStateReport_ProvidedContentBody{
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
		status rms.VStateReport_StateStatus
	}{
		{name: "ready state", status: rms.StateStatusReady},
		{name: "inactive state", status: rms.StateStatusInactive},
		{name: "missing state", status: rms.StateStatusMissing},
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

				pl := &rms.VStateReport{
					Status: rms.StateStatusReady,
					Object: rms.NewReference(objectGlobal),
					AsOf:   prevPulse,
					ProvidedContent: &rms.VStateReport_ProvidedContentBody{
						LatestDirtyState: &rms.ObjectState{
							Reference: rms.NewReferenceLocal(initRef),
							Class:     rms.NewReference(class),
							State:     rms.NewBytes(initState),
						},
						LatestValidatedState: &rms.ObjectState{
							Reference: rms.NewReferenceLocal(initRef),
							Class:     rms.NewReference(class),
							State:     rms.NewBytes(initState),
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
				typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
					assert.NotNil(t, report.ProvidedContent)
					assert.Equal(t, rms.StateStatusReady, report.Status)
					assert.Equal(t, initRef, report.ProvidedContent.LatestDirtyState.Reference.GetValueWithoutBase())
					assert.Equal(t, initRef, report.ProvidedContent.LatestValidatedState.Reference.GetValueWithoutBase())
					assert.Equal(t, initState, report.ProvidedContent.LatestDirtyState.State.GetBytes())
					assert.Equal(t, initState, report.ProvidedContent.LatestValidatedState.State.GetBytes())
					return false
				})
			}

			// send second VStateReport
			{
				pl := &rms.VStateReport{
					Status: testCase.status,
					Object: rms.NewReference(objectGlobal),
					AsOf:   prevPulse,
				}
				if testCase.status == rms.StateStatusReady {
					pl.ProvidedContent = &rms.VStateReport_ProvidedContentBody{
						LatestDirtyState: &rms.ObjectState{
							Reference: rms.NewReferenceLocal(server.RandomLocalWithPulse()),
							Class:     rms.NewReference(class),
							State:     rms.NewBytes([]byte("new state")),
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
