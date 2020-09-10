// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/shareddata"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object/preservedstatereport"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
)

func TestSMObject_CallSummarySM(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc = minimock.NewController(t)

		smObject        = newSMObjectWithPulse()
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	smObject.globalLimiter = tool.NewRunnerLimiter(4)
	smObject.SetState(HasState)

	res1 := rms.VCallResult{
		Callee:       rms.NewReference(smObject.Reference),
		CallOutgoing: rms.NewReference(gen.UniqueGlobalRef()),
	}
	res2 := rms.VCallResult{
		Callee:       rms.NewReference(smObject.Reference),
		CallOutgoing: rms.NewReference(gen.UniqueGlobalRef()),
	}

	req1Ref := res1.CallOutgoing.GetValue()
	req2Ref := res2.CallOutgoing.GetValue()

	smObject.SharedState.KnownRequests.Add(isolation.CallTolerable, req1Ref)
	smObject.SharedState.KnownRequests.Add(isolation.CallIntolerable, req2Ref)
	smObject.SharedState.KnownRequests.SetActive(isolation.CallTolerable, req1Ref)
	smObject.SharedState.KnownRequests.SetActive(isolation.CallIntolerable, req2Ref)
	smObject.SharedState.KnownRequests.Finish(isolation.CallTolerable, req1Ref, &res1)
	smObject.SharedState.KnownRequests.Finish(isolation.CallIntolerable, req2Ref, &res2)

	pulseNumber := smObject.pulseSlot.PulseData().PulseNumber

	{
		initCtx := smachine.NewInitializationContextMock(mc).
			ShareMock.Return(sharedStateData).
			PublishMock.Expect(smObject.Reference, sharedStateData).Return(true).
			JumpMock.Return(smachine.StateUpdate{}).
			SetDefaultMigrationMock.Set(func(m1 smachine.MigrateFunc) {

		})

		smObject.Init(initCtx)

		_, inactive := smObject.SummaryDone.GetCounts()
		require.Equal(t, 0, inactive)
		limit, _ := smObject.SummaryDone.GetLimit()
		require.Equal(t, 0, limit)
	}

	{
		sdlStateReport := smachine.NewUnboundSharedData(&rms.VStateReport{})
		sdlSummarySync := smachine.NewUnboundSharedData(&smachine.SyncLink{})

		publishDuringMigrate := func(key interface{}, data interface{}) (b1 bool) {
			assert.NotNil(t, data)

			switch k := key.(type) {
			case preservedstatereport.ReportKey:
				assert.Equal(t, preservedstatereport.BuildReportKey(smObject.Reference), k)
			case callsummary.SummarySyncKey:
				assert.Equal(t, callsummary.BuildSummarySyncKey(smObject.Reference), k)
			default:
				t.Fatal("Unexpected published key")
			}

			switch data.(type) {
			case smachine.SharedDataLink:
			default:
				t.Fatal("Unexpected published data")
			}

			return true
		}

		getSharedLink := func(data interface{}, flags smachine.ShareDataFlags) (s1 smachine.SharedDataLink) {
			switch data.(type) {
			case *rms.VStateReport:
				return sdlStateReport
			case *smachine.SyncLink:
				return sdlSummarySync
			default:
				t.Fatal("unexpected shared data type")
			}

			return smachine.SharedDataLink{}
		}

		migrationCtx := smachine.NewMigrationContextMock(mc).
			LogMock.Return(smachine.Logger{}).
			UnpublishAllMock.Expect().Return().
			ShareMock.Set(getSharedLink).
			PublishMock.Set(publishDuringMigrate).
			JumpMock.Set(testutils.AssertJumpStep(t, smObject.stepPublishCallSummary))

		smObject.migrate(migrationCtx)
	}

	{
		scs := callsummary.SharedCallSummary{
			Requests: callregistry.NewObjectRequestTable(),
		}
		sdl := smachine.NewUnboundSharedData(&scs)

		firstCall := true

		getPublishedLink := func(key interface{}) (s1 smachine.SharedDataLink) {
			var link smachine.SharedDataLink
			switch key {
			case callsummary.BuildSummarySharedKey(pulseNumber):
				if firstCall {
					firstCall = false
					link = smachine.SharedDataLink{}
				} else {
					link = sdl
				}
			default:
				t.Fatal("Unexpected key")
			}

			return link
		}

		executionCtx := smachine.NewExecutionContextMock(mc).
			GetPublishedLinkMock.Set(getPublishedLink).
			InitChildMock.Set(
			func(c1 smachine.CreateFunc) (s1 smachine.SlotLink) {
				callSummarySM := c1(smachine.NewConstructionContextMock(mc))
				require.Equal(t, callsummary.NewStateMachineCallSummary(pulseNumber), callSummarySM)
				return smachine.SlotLink{}
			}).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			UnpublishMock.Expect(callsummary.BuildSummarySyncKey(smObject.Reference)).Return(true).
			ApplyAdjustmentMock.Expect(smObject.summaryDoneCtl.NewValue(true)).Return(true).
			JumpMock.Set(testutils.AssertJumpStep(t, smObject.stepFinalize))

		smObject.stepPublishCallSummary(executionCtx)

		res, ok := scs.Requests.GetObjectCallResults(smObject.Reference)

		require.True(t, ok)
		require.Equal(t, 2, len(res.CallResults))

		actRes1, ok := res.CallResults[req1Ref]
		require.True(t, ok)
		require.Equal(t, &res1, actRes1.Result)

		actRes2, ok := res.CallResults[req2Ref]
		require.True(t, ok)
		require.Equal(t, &res2, actRes2.Result)
	}

	mc.Finish()
}
