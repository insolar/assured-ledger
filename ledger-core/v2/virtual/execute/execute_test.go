// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
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
		catalog         = object.NewCatalogWrapperMock(mc)
		smObjectID      = gen.IDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallRequestFlags(contract.CallTolerable, contract.CallDirty)
	)
	defer mc.Finish()

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	catalog.AddObject(smGlobalRef, smObjectAccessor)

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
		objectCatalog: catalog.Mock(),
		pulseSlot:     &pulseSlot,
	}

	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMExecute{}
		stepChecker.AddStep(exec.stepCheckRequest)
		stepChecker.AddStep(exec.stepUpdateKnownRequests)
		stepChecker.AddStep(exec.stepUpdatePendingCounters)
		stepChecker.AddStep(exec.stepWaitObjectReady)
	}
	defer func() { require.NoError(t, stepChecker.CheckDone()) }()

	{ // initialization
		initCtx := smachine.NewInitializationContextMock(mc).
			GetContextMock.Return(ctx).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.Init(initCtx)
	}

	{ // getObject
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(CallSharedDataAccessor).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.stepGetObject(execCtx)
	}

	assert.Equal(t, uint8(0), smObject.PotentialMutablePendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialImmutablePendingCount)

	{ // updateCounters after
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(CallSharedDataAccessor).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.stepUpdateKnownRequests(execCtx)
	}

	{ // updateCounters after
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(CallSharedDataAccessor).
			SetDefaultMigrationMock.Return().
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.stepUpdatePendingCounters(execCtx)
	}

	assert.Equal(t, uint8(1), smObject.PotentialMutablePendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialImmutablePendingCount)
}

func TestSMExecute_UpdateKnownRequests(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		catalog         = object.NewCatalogWrapperMock(mc)
		smObjectID      = gen.IDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallRequestFlags(contract.CallTolerable, contract.CallDirty)
	)
	defer mc.Finish()

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	catalog.AddObject(smGlobalRef, smObjectAccessor)

	callee := gen.Reference()
	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTConstructor,
			CallFlags:           callFlags,
			CallSiteDeclaration: testwallet.GetPrototype(),
			CallSiteMethod:      "New",
			CallOutgoing:        smObjectID,
			Callee:              callee,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		},
		objectCatalog: catalog.Mock(),
		pulseSlot:     &pulseSlot,
	}

	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMExecute{}
		stepChecker.AddStep(exec.stepCheckRequest)
		stepChecker.AddStep(exec.stepUpdateKnownRequests)
		stepChecker.AddStep(exec.stepUpdatePendingCounters)
	}
	defer func() { require.NoError(t, stepChecker.CheckDone()) }()

	{ // initialization
		initCtx := smachine.NewInitializationContextMock(mc).
			GetContextMock.Return(ctx).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.Init(initCtx)
	}

	{ // get object
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(CallSharedDataAccessor).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.stepGetObject(execCtx)
	}

	assert.Empty(t, smObject.KnownRequests)

	{ // update known requests successful
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(CallSharedDataAccessor).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.stepUpdateKnownRequests(execCtx)
	}

	assert.Len(t, smObject.KnownRequests, 1)
	outgoing := reference.NewRecordOf(callee, smObjectID)
	_, ok := smObject.KnownRequests[outgoing]
	assert.True(t, ok)

	{ // update known requests panics
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(CallSharedDataAccessor)

		checkerFunc := func() {
			smExecute.stepUpdateKnownRequests(execCtx)
		}
		assert.Panics(t, checkerFunc, "panic with not implemented deduplication algorithm should be here")
	}
}
