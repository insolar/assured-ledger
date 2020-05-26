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
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/slotmachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/stepchecker"
)

func TestSMExecute_IncreasePendingCounter(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		catalog         = object.NewCatalogMockWrapper(mc)
		smObjectID      = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
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

	stepChecker := stepchecker.New()
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
		catalog         = object.NewCatalogMockWrapper(mc)
		smObjectID      = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)
	defer mc.Finish()

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	catalog.AddObject(smGlobalRef, smObjectAccessor)

	callee := gen.UniqueReference()
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

	stepChecker := stepchecker.New()
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

func TestSMExecute_Semi_IncrementPendingCounters(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		prototype   = gen.UniqueReference()
		caller      = gen.UniqueReference()
		callee      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		objectRef   = reference.NewSelf(outgoing)
		sharedState = &object.SharedState{
			Info: object.Info{
				KnownRequests:  make(map[reference.Global]struct{}),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				MutableExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	slotMachine := slotmachine.NewControlledSlotMachine(ctx, t, true)
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.PrepareRunner(mc)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:    uint32(payload.TypeVCallRequest),
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:              caller,
			Callee:              callee,
			CallSiteDeclaration: prototype,
			CallSiteMethod:      "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}

	{
		catalogWrapper := object.NewCatalogMockWrapper(mc)
		var catalog object.Catalog = catalogWrapper.Mock()
		slotMachine.AddInterfaceDependency(&catalog)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
	}

	{
		pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot := conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		slotMachine.AddDependency(&pulseSlot)
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)

	require.Equal(t, uint8(0), sharedState.PotentialMutablePendingCount)
	require.Equal(t, uint8(0), sharedState.PotentialImmutablePendingCount)

	stepToWait := smWrapper.StateMachine().(*SMExecute).stepWaitObjectReady
	if !slotMachine.StepUntil(smWrapper.WaitStep(stepToWait)) {
		panic(throw.FailHere("slotmachine stopped"))
	}

	require.Equal(t, uint8(1), sharedState.PotentialMutablePendingCount)
	require.Equal(t, uint8(0), sharedState.PotentialImmutablePendingCount)

	mc.Finish()
}
