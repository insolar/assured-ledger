// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object/finalizedstate"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tables"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/shareddata"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSMObject_CallSummarySM(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject        = newSMObjectWithPulse()
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	smObject.SetState(HasState)

	res1 := payload.VCallResult{
		Callee:       smObject.Reference,
		CallOutgoing: gen.UniqueLocalRef(),
	}
	res2 := payload.VCallResult{
		Callee:       smObject.Reference,
		CallOutgoing: gen.UniqueLocalRef(),
	}

	req1Ref := reference.NewRecordOf(res1.Callee, res1.CallOutgoing)
	req2Ref := reference.NewRecordOf(res2.Callee, res2.CallOutgoing)

	smObject.SharedState.KnownRequests.GetList(contract.CallTolerable).Add(req1Ref)
	smObject.SharedState.KnownRequests.GetList(contract.CallIntolerable).Add(req2Ref)
	smObject.SharedState.KnownRequests.GetList(contract.CallTolerable).Finish(req1Ref, &res1)
	smObject.SharedState.KnownRequests.GetList(contract.CallIntolerable).Finish(req2Ref, &res2)

	pulseNumber := smObject.pulseSlot.PulseData().PulseNumber

	{
		initCtx := smachine.NewInitializationContextMock(mc).
			ShareMock.Return(sharedStateData).
			PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
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
		sdlStateReport := smachine.NewUnboundSharedData(&payload.VStateReport{})
		sdlSummarySync := smachine.NewUnboundSharedData(&smachine.SyncLink{})

		migrationCtx := smachine.NewMigrationContextMock(mc).
			LogMock.Return(smachine.Logger{}).
			UnpublishAllMock.Expect().Return().
			ShareMock.Set(func(data interface{}, flags smachine.ShareDataFlags) (s1 smachine.SharedDataLink) {
			switch data.(type) {
			case *payload.VStateReport:
				return sdlStateReport
			case *smachine.SyncLink:
				return sdlSummarySync
			default:
			}
			panic("unexpected shared data")
		}).
			PublishMock.Set(func(key interface{}, data interface{}) (b1 bool) {
			switch d := key.(type) {
			case finalizedstate.ReportKey:
				require.Equal(t, finalizedstate.BuildReportKey(smObject.Reference), d)
			case callsummary.SummarySyncKey:
				require.Equal(t, callsummary.BuildSummarySyncKey(smObject.Reference), d)
			}
			return true
		}).
			JumpMock.Set(testutils.AssertJumpStep(t, smObject.stepPublishCallSummary))

		smObject.stepMigrate(migrationCtx)
	}

	{
		scs := callsummary.SharedCallSummary{
			Requests: tables.NewObjectRequestTable(),
		}
		sdl := smachine.NewUnboundSharedData(&scs)

		firsCall := true

		executionCtx := smachine.NewExecutionContextMock(mc).
			GetPublishedLinkMock.Set(func(key interface{}) (s1 smachine.SharedDataLink) {
			if key == callsummary.BuildSummarySharedKey(pulseNumber) {
				if firsCall {
					firsCall = false
					return smachine.SharedDataLink{}
				} else {
					return sdl
				}
			}
			panic("Unexpected key")
		}).
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

		requests, ok := scs.Requests.GetObjectsKnownRequests(smObject.Reference)

		require.True(t, ok)
		require.Equal(t, 2, requests.Len())

		actRes1, ok := requests.GetList(contract.CallTolerable).GetResult(req1Ref)
		require.True(t, ok)
		require.Equal(t, &res1, actRes1)

		actRes2, ok := requests.GetList(contract.CallIntolerable).GetResult(req2Ref)
		require.True(t, ok)
		require.Equal(t, &res2, actRes2)
	}

	mc.Finish()
}
