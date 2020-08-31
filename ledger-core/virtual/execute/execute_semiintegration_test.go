// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/virtualdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
)

func TestSMExecute_Semi_IncrementPendingCounters(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := virtualdebugger.New(ctx, t)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		class   = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller  = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		limiter = tool.NewRunnerLimiter(4)

		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
			},
		}
	)

	outgoing := reference.NewRecordOf(caller, slotMachine.GenerateLocal())
	objectRef := reference.NewSelf(outgoing.GetLocal())

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CallTypeConstructor,
			CallFlags:    payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
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
		slotMachine.AddDependency(limiter)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)

	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepExecuteStart))

	require.Equal(t, 1, sharedState.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}

func TestSMExecute_MigrateBeforeLock(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := virtualdebugger.New(ctx, t)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		class       = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller      = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		callee      = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		limiter     = tool.NewRunnerLimiter(4)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
			},
		}
	)

	outgoing := reference.NewRecordOf(caller, slotMachine.GenerateLocal())
	objectRef := reference.NewSelf(outgoing.GetLocal())

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CallTypeConstructor,
			CallFlags:    payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
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
		slotMachine.AddDependency(limiter)

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
	defer executeLeakCheck(t)

	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := virtualdebugger.New(ctx, t)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		class       = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller      = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		limiter     = tool.NewRunnerLimiter(4)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
			},
		}
	)

	outgoing := reference.NewRecordOf(caller, slotMachine.GenerateLocal())
	objectRef := reference.NewSelf(outgoing.GetLocal())

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CallTypeConstructor,
			CallFlags:    payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
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
		slotMachine.AddDependency(limiter)

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
	defer executeLeakCheck(t)

	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := virtualdebugger.New(ctx, t)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		class       = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller      = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		outgoing    = reference.NewRecordOf(caller, slotMachine.GenerateLocal())
		objectRef   = reference.NewSelf(outgoing.GetLocal())
		limiter     = tool.NewRunnerLimiter(4)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
			},
		}
	)

	sharedState.SetState(object.Missing)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CallTypeConstructor,
			CallFlags:    payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
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
		slotMachine.AddDependency(limiter)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)

	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepExecuteStart))

	require.Equal(t, 1, sharedState.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}

func TestSMExecute_Semi_ConstructorOnBadObject(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		mc      = minimock.NewController(t)
		ctx     = instestlogger.TestContext(t)
		limiter = tool.NewRunnerLimiter(4)
	)

	slotMachine := virtualdebugger.New(ctx, t)
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	slotMachine.MessageSender.SendTarget.Set(func(_ context.Context, msg payload.Marshaler, target reference.Global, _ ...messagesender.SendOption) error {
		res := msg.(*payload.VCallResult)
		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
		require.NoError(t, sysErr)
		require.Contains(t, contractErr.Error(), "try to call method on deactivated object")
		return nil
	})
	var (
		class       = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller      = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		outgoing    = reference.NewRecordOf(caller, slotMachine.GenerateLocal())
		objectRef   = reference.NewSelf(outgoing.GetLocal())
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "ordered calls").SyncLink(),
			},
		}
	)

	sharedState.SetState(object.Inactive)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CallTypeConstructor,
			CallFlags:    payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
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
			catalog object.Catalog = catalogWrapper.Mock()
		)
		authServiceMock := authentication.NewServiceMock(t)
		authServiceMock.HasToSendTokenMock.Return(false)
		authService := authentication.Service(authServiceMock)
		slotMachine.AddInterfaceDependency(&authService)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddDependency(limiter)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)

	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	slotMachine.RunTil(smWrapper.AfterStop())

	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}

func TestSMExecute_Semi_MethodOnEmptyObject(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		mc      = minimock.NewController(t)
		ctx     = instestlogger.TestContext(t)
		limiter = tool.NewRunnerLimiter(4)
	)

	slotMachine := virtualdebugger.NewWithErrorFilter(ctx, t, func(s string) bool {
		return !strings.Contains(s, "async call: runtime error: invalid memory address or nil pointer dereference")
	})
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		caller      = slotMachine.GenerateGlobal()
		objectRef   = slotMachine.GenerateGlobal()
		outgoing    = reference.NewRecordOf(slotMachine.GenerateGlobal(), slotMachine.GenerateLocal())
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:               callregistry.NewRequestTable(),
				KnownRequests:              callregistry.NewWorkingTable(),
				ReadyToWork:                smsync.NewConditionalBool(true, "ReadyToWork").SyncLink(),
				PendingConstructorFinished: smsync.NewConditionalBool(false, "pendingConstructorFinished").SyncLink(),
			},
		}
	)

	sharedState.SetState(object.Empty)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CallTypeMethod,
			CallFlags:    payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
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
		slotMachine.AddDependency(limiter)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)

	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	slotMachine.RunTil(predicate.AfterCustomEventType(reflect.TypeOf(markerPendingConstructorWait{})))

	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	require.Equal(t, 0, sharedState.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	slotMachine.Migrate()

	slotMachine.RunTil(smWrapper.AfterStop())

	require.NoError(t, catalogWrapper.CheckDone())

	mc.Finish()
}
