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
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/shareddata"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/slotdebugger"
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

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
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
			SetDefaultMigrationMock.Set(testutils.AssertMigration(t, smExecute.migrationDefault)).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepCheckRequest))

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

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
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

	assert.Equal(t, uint8(0), smObject.PotentialOrderedPendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialUnorderedPendingCount)

	assert.Zero(t, smObject.KnownRequests.Len())

	{ // updateCounters after
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			SetDefaultMigrationMock.Return().
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepExecuteStart))

		smExecute.stepStartRequestProcessing(execCtx)
	}

	assert.Equal(t, uint8(1), smObject.PotentialOrderedPendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialUnorderedPendingCount)

	assert.Equal(t, smObject.KnownRequests.Len(), 1)
	assert.True(t, smObject.KnownRequests.GetList(contract.CallTolerable).Exist(smExecute.execution.Outgoing))

	{
		// call same request
		// expect duplicate detection and SM stop
		wasStoped := false

		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			LogMock.Return(smachine.Logger{}).
			StopMock.Set(
			func() (s1 smachine.StateUpdate) {
				wasStoped = true
				return smachine.StateUpdate{}
			})

		smExecute.stepStartRequestProcessing(execCtx)
		assert.Equal(t, true, wasStoped)
	}

	mc.Finish()
}

func TestSMExecute_Deduplication(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
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

	{
		// duplicate pending request exists
		// expect SM stop
		pendingList := smObject.PendingTable.GetList(contract.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		wasStoped := false

		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			LogMock.Return(smachine.Logger{}).
			StopMock.Set(
			func() (s1 smachine.StateUpdate) {
				wasStoped = true
				return smachine.StateUpdate{}
			})

		smExecute.stepDeduplicate(execCtx)
		require.Equal(t, true, wasStoped)
	}

	{
		// start deduplication before getting all pending requests
		// expecting going sleep
		smObject.PendingTable = object.NewRequestTable()

		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			AcquireForThisStepMock.Return(false).
			SleepMock.Return(smachine.NewStateConditionalBuilderMock(mc).ThenRepeatMock.Return(smachine.StateUpdate{}))

		smExecute.stepDeduplicate(execCtx)
	}

	{
		// start deduplication after getting all pending requests
		// expecting jump
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepStartRequestProcessing))

		smExecute.stepDeduplicate(execCtx)
	}

	mc.Finish()
}

func TestSMExecute_Semi_IncrementPendingCounters(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		class       = gen.UniqueReference()
		caller      = gen.UniqueReference()
		callee      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		objectRef   = reference.NewSelf(outgoing)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewRequestTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:    uint32(payload.TypeVCallRequest),
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:              caller,
			Callee:              callee,
			CallSiteDeclaration: class,
			CallSiteMethod:      "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var catalog object.Catalog = catalogWrapper.Mock()
		slotMachine.AddInterfaceDependency(&catalog)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)

	require.Equal(t, uint8(0), sharedState.PotentialOrderedPendingCount)
	require.Equal(t, uint8(0), sharedState.PotentialUnorderedPendingCount)

	slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepExecuteStart))

	require.Equal(t, uint8(1), sharedState.PotentialOrderedPendingCount)
	require.Equal(t, uint8(0), sharedState.PotentialUnorderedPendingCount)

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}

func TestSMExecute_MigrateBeforeLock(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		class       = gen.UniqueReference()
		caller      = gen.UniqueReference()
		callee      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		objectRef   = reference.NewSelf(outgoing)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewRequestTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:    uint32(payload.TypeVCallRequest),
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:              caller,
			Callee:              callee,
			CallSiteDeclaration: class,
			CallSiteMethod:      "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var catalog object.Catalog = catalogWrapper.Mock()
		slotMachine.AddInterfaceDependency(&catalog)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)

	require.False(t, smExecute.migrationHappened)

	slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepTakeLock))

	slotMachine.Migrate()

	slotMachine.RunTil(smWrapper.AfterStop())

	require.False(t, smExecute.migrationHappened)

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}

func TestSMExecute_StepTakeLockGoesToDeduplicationForRequestWithRepeatedCallFlag(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
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
		CallRequestFlags:    payload.BuildCallRequestFlags(payload.SendResultDefault, payload.RepeatedCall),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	smExecute = expectedInitState(ctx, smExecute)

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			AcquireMock.Return(true).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepDeduplicate))

		smExecute.stepTakeLock(execCtx)
	}
}

func TestSMExecute_MigrateAfterLock(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		class       = gen.UniqueReference()
		caller      = gen.UniqueReference()
		callee      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		objectRef   = reference.NewSelf(outgoing)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewRequestTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:    uint32(payload.TypeVCallRequest),
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:              caller,
			Callee:              callee,
			CallSiteDeclaration: class,
			CallSiteMethod:      "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var catalog object.Catalog = catalogWrapper.Mock()
		slotMachine.AddInterfaceDependency(&catalog)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)

	require.False(t, smExecute.migrationHappened)

	slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepExecuteStart))

	slotMachine.Migrate()

	slotMachine.RunTil(smWrapper.AfterAnyMigrate())

	assert.True(t, smExecute.migrationHappened)

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}
