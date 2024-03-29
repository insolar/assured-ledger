package integration

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

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
				prevPulse    = server.GetPulse().PulseNumber
				objectGlobal = server.RandomGlobalWithPulse()
				initRef      = server.RandomRecordOf(objectGlobal)
				class        = server.RandomGlobalWithPulse()
			)

			// send first VStateReport
			{
				report := server.StateReportBuilder().Object(objectGlobal).Ready().
					StateRef(initRef).Memory(initState).Report()

				server.IncrementPulse(ctx)

				waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, &report)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, waitReport)
			}

			// add checker
			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			{
				typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
					assert.NotNil(t, report.ProvidedContent)
					assert.Equal(t, rms.StateStatusReady, report.Status)
					assert.Equal(t, initRef, report.ProvidedContent.LatestDirtyState.Reference.GetValue())
					assert.Equal(t, initRef, report.ProvidedContent.LatestValidatedState.Reference.GetValue())
					assert.Equal(t, initState, report.ProvidedContent.LatestDirtyState.Memory.GetBytes())
					assert.Equal(t, initState, report.ProvidedContent.LatestValidatedState.Memory.GetBytes())
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
							Reference: rms.NewReference(server.RandomRecordOfWithGivenPulse(objectGlobal, prevPulse)),
							Class:     rms.NewReference(class),
							Memory:    rms.NewBytes([]byte("new state")),
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
