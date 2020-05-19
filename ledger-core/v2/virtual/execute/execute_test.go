// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

func TestSMExecute_IncreasePendingCounter(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		catalog         = object.NewCatalogMock(mc)
		smObjectID      = gen.IDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags payload.CallRequestFlags
	)

	defer mc.Finish()

	callFlags.SetTolerance(payload.CallTolerable)
	callFlags.SetState(payload.CallDirty)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTConstructor,
			CallFlags:           callFlags,
			CallSiteDeclaration: testwallet.GetPrototype(),
			CallSiteMethod:      "New",
			CallOutgoing:        smObjectID,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		},
		objectCatalog: catalog,
		pulseSlot:     &pulseSlot,
	}

	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMExecute{}
		stepChecker.AddStep(exec.stepCheckRequest)
		stepChecker.AddStep(exec.stepUpdateSawRequests)
		stepChecker.AddStep(exec.stepUpdatePendingCounters)
		stepChecker.AddStep(exec.stepWaitObjectReady)
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		GetContextMock.Return(ctx).
		JumpMock.Set(testutils.CheckWrapper(stepChecker, t)).
		UseSharedMock.Set(CallSharedDataAccessor)

	execCtx.SetDefaultMigrationMock.Return()

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	catalog.GetOrCreateMock.Expect(execCtx, smGlobalRef).Return(smObjectAccessor)

	smExecute.Init(execCtx)
	smExecute.stepGetObject(execCtx)

	assert.Equal(t, uint8(0), smObject.PotentialMutablePendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialImmutablePendingCount)

	smExecute.stepUpdateSawRequests(execCtx)
	smExecute.stepUpdatePendingCounters(execCtx)

	assert.Equal(t, uint8(1), smObject.PotentialMutablePendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialImmutablePendingCount)
}

func TestSMExecute_UpdateSawRequests(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		catalog         = object.NewCatalogMock(mc)
		smObjectID      = gen.IDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags payload.CallRequestFlags
	)

	defer mc.Finish()

	callFlags.SetTolerance(payload.CallTolerable)
	callFlags.SetState(payload.CallDirty)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTConstructor,
			CallFlags:           callFlags,
			CallSiteDeclaration: testwallet.GetPrototype(),
			CallSiteMethod:      "New",
			CallOutgoing:        smObjectID,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		},
		objectCatalog: catalog,
		pulseSlot:     &pulseSlot,
	}

	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMExecute{}
		stepChecker.AddStep(exec.stepCheckRequest)
		stepChecker.AddStep(exec.stepUpdateSawRequests)
		stepChecker.AddStep(exec.stepUpdatePendingCounters)
		stepChecker.AddStep(exec.stepWaitObjectReady)
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		GetContextMock.Return(ctx).
		JumpMock.Set(testutils.CheckWrapper(stepChecker, t)).
		UseSharedMock.Set(CallSharedDataAccessor)



	// execCtx.SetDefaultMigrationMock.Return()

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	catalog.GetOrCreateMock.Expect(execCtx, smGlobalRef).Return(smObjectAccessor)

	smExecute.Init(execCtx)
	smExecute.stepGetObject(execCtx)


	assert.Empty(t,  smObject.SawRequests)

	smExecute.stepUpdateSawRequests(execCtx)
	// smExecute.stepUpdatePendingCounters(execCtx)

	// _, ok := smObject.SawRequests[smGlobalRef]
	// fmt.Println("smGlobalRef ", smGlobalRef.String())
	// assert.True (t, ok, "reference not found in SawRequests")
	assert.Len(t, smObject.SawRequests, 1)

	// getBalance SM ==================================

	smExecuteGetBalance := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTMethod,
			CallFlags:           callFlags,
			CallSiteDeclaration: testwallet.GetPrototype(),
			CallSiteMethod:      "GetBalance",
			CallOutgoing:        smObjectID,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		},
		objectCatalog: catalog,
		pulseSlot:     &pulseSlot,
	}

	stepChecker2 := testutils.NewSMStepChecker()
	{
		exec := SMExecute{}
		stepChecker2.AddStep(exec.stepCheckRequest)
		stepChecker2.AddStep(exec.stepUpdateSawRequests)
		stepChecker2.AddStep(exec.stepUpdatePendingCounters)
		stepChecker2.AddStep(exec.stepWaitObjectReady)
	}

	execCtx2 := smachine.NewExecutionContextMock(mc).
		GetContextMock.Return(ctx).
		JumpMock.Set(testutils.CheckWrapper(stepChecker2, t)).
		UseSharedMock.Set(CallSharedDataAccessor)

	smExecuteGetBalance.Init(execCtx2)
	smExecuteGetBalance.stepGetObject(execCtx2)
	smExecuteGetBalance.stepUpdateSawRequests(execCtx2)
	// smExecuteGetBalance.stepUpdatePendingCounters(execCtx2)

	assert.Len(t, smObject.SawRequests, 2)
}
