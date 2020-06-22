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

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/slotdebugger"
)

func TestSMExecute_Semi_IncrementPendingCounters(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		class       = gen.UniqueReference()
		caller      = gen.UniqueReference()
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	outgoing := gen.UniqueIDWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
	objectRef := reference.NewSelf(outgoing)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:         caller,
			Callee:         class,
			CallSiteMethod: "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var (
			authService authentication.Service = authentication.NewServiceMock(t)
			catalog     object.Catalog         = catalogWrapper.Mock()
		)
		slotMachine.AddInterfaceDependency(&authService)
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
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	outgoing := gen.UniqueIDWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
	objectRef := reference.NewSelf(outgoing)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:         class,
			Callee:         callee,
			CallSiteMethod: "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var (
			authService authentication.Service = authentication.NewServiceMock(t)
			catalog     object.Catalog         = catalogWrapper.Mock()
		)
		slotMachine.AddInterfaceDependency(&authService)
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

func TestSMExecute_MigrateAfterLock(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		class       = gen.UniqueReference()
		caller      = gen.UniqueReference()
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	outgoing := gen.UniqueIDWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
	objectRef := reference.NewSelf(outgoing)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:         caller,
			Callee:         class,
			CallSiteMethod: "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var (
			authService authentication.Service = authentication.NewServiceMock(t)
			catalog     object.Catalog         = catalogWrapper.Mock()
		)
		slotMachine.AddInterfaceDependency(&authService)
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

func TestSMExecute_Semi_ConstructorOnMissingObject(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		class       = gen.UniqueReference()
		caller      = gen.UniqueReference()
		outgoing    = slotMachine.GenerateLocal()
		objectRef   = reference.NewSelf(outgoing)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	sharedState.SetState(object.Missing)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:         caller,
			Callee:         class,
			CallSiteMethod: "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var (
			authService authentication.Service = authentication.NewServiceMock(t)
			catalog     object.Catalog         = catalogWrapper.Mock()
		)
		slotMachine.AddInterfaceDependency(&authService)
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

func TestSMExecute_Semi_ConstructorOnBadObject(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		class       = gen.UniqueReference()
		caller      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		objectRef   = reference.NewSelf(outgoing)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	sharedState.SetState(object.Inactive)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:         caller,
			Callee:         class,
			CallSiteMethod: "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var (
			authService authentication.Service = authentication.NewServiceMock(t)
			catalog     object.Catalog         = catalogWrapper.Mock()
		)
		slotMachine.AddInterfaceDependency(&authService)
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

	slotMachine.RunTil(smWrapper.AfterStop())

	require.Equal(t, uint8(0), sharedState.PotentialOrderedPendingCount)
	require.Equal(t, uint8(0), sharedState.PotentialUnorderedPendingCount)

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}

func TestSMExecute_Semi_MethodOnEmptyObject(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		caller      = slotMachine.GenerateGlobal()
		outgoing    = slotMachine.GenerateLocal()
		objectRef   = reference.NewSelf(outgoing)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		}
	)

	sharedState.SetState(object.Empty)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CTMethod,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:         caller,
			Callee:         objectRef,
			CallSiteMethod: "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var (
			authService authentication.Service = authentication.NewServiceMock(t)
			catalog     object.Catalog         = catalogWrapper.Mock()
		)
		slotMachine.AddInterfaceDependency(&authService)
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

	slotMachine.RunTil(smWrapper.AfterStop())

	require.Equal(t, uint8(0), sharedState.PotentialOrderedPendingCount)
	require.Equal(t, uint8(0), sharedState.PotentialUnorderedPendingCount)

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}
