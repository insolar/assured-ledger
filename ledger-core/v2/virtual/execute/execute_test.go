// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"context"
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
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/reflectkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/slotmachine"
)

func expectedInitState(ctx context.Context, sm SMExecute) SMExecute {
	sm.execution.Context = ctx
	sm.execution.Sequence = 0
	sm.execution.Request = sm.Payload
	sm.execution.Pulse = sm.pulseSlot.PulseData()

	if sm.Payload.CallType == payload.CTConstructor {
		sm.isConstructor = true
		sm.execution.Object = reference.NewSelf(sm.Payload.CallOutgoing)
	} else {
		sm.execution.Object = sm.Payload.Callee
	}

	sm.execution.Incoming = reference.NewRecordOf(sm.Payload.Caller, sm.Payload.CallOutgoing)
	sm.execution.Outgoing = reference.NewRecordOf(sm.Payload.Callee, sm.Payload.CallOutgoing)

	sm.execution.Isolation = contract.MethodIsolation{
		Interference: sm.Payload.CallFlags.GetInterference(),
		State:        sm.Payload.CallFlags.GetState(),
	}
	return sm
}

func TestSMExecute_Init(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallRequestFlags(contract.CallTolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetPrototype(),
		CallSiteMethod:      "New",
		CallOutgoing:        smObjectID,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	initializedSMExecute := expectedInitState(ctx, smExecute)

	{
		initCtx := smachine.NewInitializationContextMock(mc).
			GetContextMock.Return(ctx).
			SetDefaultMigrationMock.Set(func(fn smachine.MigrateFunc) {
			require.Equal(t, reflectkit.CodeOf(smExecute.defaultMigration), reflectkit.CodeOf(fn))
			return
		}).
			JumpMock.Set(func(s1 smachine.StateFunc) (s2 smachine.StateUpdate) {
			require.Equal(t, reflectkit.CodeOf(smExecute.stepCheckRequest), reflectkit.CodeOf(s1))
			return smachine.StateUpdate{}
		})

		smExecute.Init(initCtx)
	}

	require.Equal(t, initializedSMExecute, smExecute)

	mc.Finish()
}

func TestSMExecute_StartRequestProcessing(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallRequestFlags(contract.CallTolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetPrototype(),
		CallSiteMethod:      "New",
		CallOutgoing:        smObjectID,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	smExecute = expectedInitState(ctx, smExecute)

	assert.Equal(t, uint8(0), smObject.PotentialMutablePendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialImmutablePendingCount)

	assert.Empty(t, smObject.KnownRequests)

	{ // updateCounters after
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(CallSharedDataAccessor).
			SetDefaultMigrationMock.Return().
			JumpMock.Set(func(s1 smachine.StateFunc) (s2 smachine.StateUpdate) {
			require.Equal(t, reflectkit.CodeOf(smExecute.stepExecuteStart), reflectkit.CodeOf(s1))
			return smachine.StateUpdate{}
		})

		smExecute.stepStartRequestProcessing(execCtx)
	}

	assert.Equal(t, uint8(1), smObject.PotentialMutablePendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialImmutablePendingCount)

	assert.Len(t, smObject.KnownRequests, 1)
	_, ok := smObject.KnownRequests[smExecute.execution.Outgoing]
	assert.True(t, ok)

	{ // update known requests panics
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(CallSharedDataAccessor)

		checkerFunc := func() {
			smExecute.stepStartRequestProcessing(execCtx)
		}
		assert.Panics(t, checkerFunc, "panic with not implemented deduplication algorithm should be here")
	}

	mc.Finish()
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
			CallFlags:    payload.BuildCallRequestFlags(contract.CallTolerable, contract.CallDirty),
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

	stepToWait := smExecute.stepExecuteStart
	if !slotMachine.StepUntil(smWrapper.WaitStep(stepToWait)) {
		panic(throw.FailHere("slotmachine stopped"))
	}

	require.Equal(t, uint8(1), sharedState.PotentialMutablePendingCount)
	require.Equal(t, uint8(0), sharedState.PotentialImmutablePendingCount)

	mc.Finish()
}

func TestSMExecute_MigrateBeforeLock(t *testing.T) {
	t.Skip()

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
			CallFlags:    payload.BuildCallRequestFlags(contract.CallTolerable, contract.CallDirty),
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

	stepToWait := smExecute.stepTakeLock
	if !slotMachine.StepUntil(smWrapper.WaitStep(stepToWait)) {
		t.Error("slotmachine stopped")
		t.FailNow()
	}

	// {
	// 	migrateCtx := smachine.NewMigrationContextMock(mc)
	// 	slotMachine.Migrate(migrateCtx)
	// }

	require.Equal(t, uint8(1), sharedState.PotentialMutablePendingCount)
	require.Equal(t, uint8(0), sharedState.PotentialImmutablePendingCount)

	mc.Finish()
}
