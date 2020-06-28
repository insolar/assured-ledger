// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tables"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/shareddata"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSMExecute_PublishVCallResultToCallSummarySM(t *testing.T) {
	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	ref := gen.UniqueGlobalRef()

	request := &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        smObjectID,
		Callee:              ref,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Payload:   request,
		pulseSlot: &pulseSlot,
	}

	ref = reference.NewRecordOf(request.Callee, request.CallOutgoing)

	smExecute = expectedInitState(ctx, smExecute)

	smExecute.execution.Result = &payload.VCallResult{}
	smExecute.migrationHappened = true

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepPublishCallSummaryData))

		smExecute.stepFinishRequest(execCtx)
	}

	sharedCallSummary := callsummary.SharedCallSummary{
		Requests: tables.NewObjectRequestTable(),
	}

	{
		workingTable := tables.NewWorkingTable()
		workingTable.GetList(contract.CallTolerable).Add(ref)
		sharedCallSummary.Requests.AddObjectRequests(smGlobalRef, workingTable)
	}

	{

		sdlSummarySync := smachine.NewUnboundSharedData(&smachine.SyncLink{})
		sdlSummaryShare := smachine.NewUnboundSharedData(&sharedCallSummary)

		execCtx := smachine.NewExecutionContextMock(mc).
			GetPublishedLinkMock.Set(func(key interface{}) (s1 smachine.SharedDataLink) {
			switch key.(type) {
			case callsummary.SummarySyncKey:
				return sdlSummarySync
			case callsummary.SummarySharedKey:
				return sdlSummaryShare
			}
			panic("Unexpected get link call")
		}).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			AcquireForThisStepMock.Expect(smachine.SyncLink{}).Return(true).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepSendDelegatedRequestFinished))

		smExecute.stepPublishCallSummaryData(execCtx)
	}

	workingTable, ok := sharedCallSummary.Requests.GetObjectsKnownRequests(smGlobalRef)
	require.Equal(t, 1, workingTable.GetList(contract.CallTolerable).Count())

	result, ok := workingTable.GetList(contract.CallTolerable).GetResult(ref)

	require.True(t, ok)
	require.NotNil(t, result)

	mc.Finish()
}
