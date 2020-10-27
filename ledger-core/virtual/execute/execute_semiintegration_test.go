// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/virtualdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
	"github.com/insolar/assured-ledger/ledger-core/virtual/vnlmn"
)

func TestSMExecute_Semi_IncrementPendingCounters(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := virtualdebugger.New(ctx, t)
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		class    = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller   = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		limiter  = tool.NewRunnerLimiter(4)
		outgoing = reference.NewRecordOf(caller, slotMachine.GenerateLocal())

		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
			},
		}

		smExecute = SMExecute{
			Payload: &rms.VCallRequest{
				CallType:     rms.CallTypeConstructor,
				CallFlags:    rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
				CallOutgoing: rms.NewReference(outgoing),

				Caller:         rms.NewReference(caller),
				Callee:         rms.NewReference(class),
				CallSiteMethod: "New",
			},
			Meta: &rms.Meta{
				Sender: rms.NewReference(caller),
			},
		}

		meRef          = gen.UniqueGlobalRef()
		objectRef      = testutils.GetObjectReference(smExecute.Payload, meRef)
		catalogWrapper = object.NewCatalogMockWrapper(mc)
	)

	{
		var (
			authService authentication.Service         = authentication.NewServiceMock(t)
			catalog     object.Catalog                 = catalogWrapper.Mock()
			memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
		)
		slotMachine.AddInterfaceDependency(&authService)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddInterfaceDependency(&memoryCache)
		slotMachine.AddDependency(limiter)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.MessageSender.SendRole.SetCheckMessage(
		func(msg rmsreg.GoGoSerializable) {
			switch res := msg.(type) {
			case *rms.LRegisterRequest:
				_ = res.AnticipatedRef.Get()

				key := vnlmn.ResultAwaitKey{
					AnticipatedRef: res.AnticipatedRef,
					RequiredFlag:   rms.RegistrationFlags_Fast,
				}

				_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(key)
				bargeIn.CallWithParam(&rms.LRegisterResponse{
					Flags:              rms.RegistrationFlags_Fast,
					AnticipatedRef:     res.AnticipatedRef,
					RegistrarSignature: rms.NewBytes([]byte("dummy")),
				})
			default:
				require.FailNow(t, "unreachable")
			}
		})

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

		outgoing = reference.NewRecordOf(caller, slotMachine.GenerateLocal())

		smExecute = SMExecute{
			Payload: &rms.VCallRequest{
				CallType:     rms.CallTypeConstructor,
				CallFlags:    rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
				CallOutgoing: rms.NewReference(outgoing),

				Caller:         rms.NewReference(class),
				Callee:         rms.NewReference(callee),
				CallSiteMethod: "New",
			},
			Meta: &rms.Meta{
				Sender: rms.NewReference(caller),
			},
		}
		meRef          = gen.UniqueGlobalRef()
		objectRef      = testutils.GetObjectReference(smExecute.Payload, meRef)
		catalogWrapper = object.NewCatalogMockWrapper(mc)
	)

	{
		var (
			authService authentication.Service         = authentication.NewServiceMock(t)
			catalog     object.Catalog                 = catalogWrapper.Mock()
			memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
		)
		slotMachine.AddInterfaceDependency(&authService)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddInterfaceDependency(&memoryCache)
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
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	slotMachine.MessageSender.SendRole.SetCheckMessage(
		func(msg rmsreg.GoGoSerializable) {
			switch res := msg.(type) {
			case *rms.LRegisterRequest:
				_ = res.AnticipatedRef.Get()

				key := vnlmn.ResultAwaitKey{
					AnticipatedRef: res.AnticipatedRef,
					RequiredFlag:   rms.RegistrationFlags_Fast,
				}

				_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(key)
				bargeIn.CallWithParam(&rms.LRegisterResponse{
					Flags:              rms.RegistrationFlags_Fast,
					AnticipatedRef:     res.AnticipatedRef,
					RegistrarSignature: rms.NewBytes([]byte("dummy")),
				})
			default:
				require.FailNow(t, "unreachable")
			}
		})

	var (
		class  = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		meRef  = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())

		limiter     = tool.NewRunnerLimiter(4)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
			},
		}

		outgoing = reference.NewRecordOf(caller, slotMachine.GenerateLocal())

		smExecute = SMExecute{
			Payload: &rms.VCallRequest{
				CallType:     rms.CallTypeConstructor,
				CallFlags:    rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
				CallOutgoing: rms.NewReference(outgoing),

				Caller:         rms.NewReference(caller),
				Callee:         rms.NewReference(class),
				CallSiteMethod: "New",
			},
			Meta: &rms.Meta{
				Sender: rms.NewReference(caller),
			},
		}
		catalogWrapper = object.NewCatalogMockWrapper(mc)
		objectRef      = testutils.GetObjectReference(smExecute.Payload, meRef)
	)

	{
		var (
			authService authentication.Service         = authentication.NewServiceMock(t)
			catalog     object.Catalog                 = catalogWrapper.Mock()
			memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
		)
		slotMachine.AddInterfaceDependency(&authService)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddInterfaceDependency(&memoryCache)
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
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.PrepareRunner(ctx, mc)

	slotMachine.MessageSender.SendRole.SetCheckMessage(
		func(msg rmsreg.GoGoSerializable) {
			switch res := msg.(type) {
			case *rms.LRegisterRequest:
				_ = res.AnticipatedRef.Get()

				key := vnlmn.ResultAwaitKey{
					AnticipatedRef: res.AnticipatedRef,
					RequiredFlag:   rms.RegistrationFlags_Fast,
				}

				_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(key)
				bargeIn.CallWithParam(&rms.LRegisterResponse{
					Flags:              rms.RegistrationFlags_Fast,
					AnticipatedRef:     res.AnticipatedRef,
					RegistrarSignature: rms.NewBytes([]byte("dummy")),
				})
			default:
				require.FailNow(t, "unreachable")
			}
		})

	var (
		class       = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller      = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		outgoing    = reference.NewRecordOf(caller, slotMachine.GenerateLocal())
		limiter     = tool.NewRunnerLimiter(4)
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
			},
		}

		smExecute = SMExecute{
			Payload: &rms.VCallRequest{
				CallType:     rms.CallTypeConstructor,
				CallFlags:    rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
				CallOutgoing: rms.NewReference(outgoing),

				Caller:         rms.NewReference(caller),
				Callee:         rms.NewReference(class),
				CallSiteMethod: "New",
			},
			Meta: &rms.Meta{
				Sender: rms.NewReference(caller),
			},
		}
		meRef          = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		objectRef      = testutils.GetObjectReference(smExecute.Payload, meRef)
		catalogWrapper = object.NewCatalogMockWrapper(mc)
	)
	sharedState.SetState(object.Missing)

	{
		var (
			authService authentication.Service         = authentication.NewServiceMock(t)
			catalog     object.Catalog                 = catalogWrapper.Mock()
			memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
		)
		slotMachine.AddInterfaceDependency(&authService)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddInterfaceDependency(&memoryCache)
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

	slotMachine.MessageSender.SendRole.SetCheckMessage(
		func(msg rmsreg.GoGoSerializable) {
			switch res := msg.(type) {
			case *rms.LRegisterRequest:
				_ = res.AnticipatedRef.Get()

				key := vnlmn.ResultAwaitKey{
					AnticipatedRef: res.AnticipatedRef,
					RequiredFlag:   rms.RegistrationFlags_Fast,
				}

				_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(key)
				bargeIn.CallWithParam(&rms.LRegisterResponse{
					Flags:              rms.RegistrationFlags_Fast,
					AnticipatedRef:     res.AnticipatedRef,
					RegistrarSignature: rms.NewBytes([]byte("dummy")),
				})
			default:
				require.FailNow(t, "unreachable")
			}
		})
	slotMachine.MessageSender.SendTarget.SetCheckMessage(
		func(msg rmsreg.GoGoSerializable) {
			switch res := msg.(type) {
			case *rms.VCallResult:
				contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
				require.NoError(t, sysErr)
				require.Contains(t, contractErr.Error(), "try to call method on deactivated object")
			default:
				require.FailNow(t, "unreachable")
			}
		})

	var (
		class       = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller      = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		outgoing    = reference.NewRecordOf(caller, slotMachine.GenerateLocal())
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "ordered calls").SyncLink(),
			},
		}

		smExecute = SMExecute{
			Payload: &rms.VCallRequest{
				CallType:     rms.CallTypeConstructor,
				CallFlags:    rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
				CallOutgoing: rms.NewReference(outgoing),

				Caller:         rms.NewReference(caller),
				Callee:         rms.NewReference(class),
				CallSiteMethod: "New",
			},
			Meta: &rms.Meta{
				Sender: rms.NewReference(caller),
			},
		}
		catalogWrapper = object.NewCatalogMockWrapper(mc)
		meRef          = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		objectRef      = testutils.GetObjectReference(smExecute.Payload, meRef)
	)

	sharedState.SetState(object.Inactive)

	{
		var (
			catalog     object.Catalog                 = catalogWrapper.Mock()
			memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
		)
		authServiceMock := authentication.NewServiceMock(t)
		authServiceMock.HasToSendTokenMock.Return(false)
		authService := authentication.Service(authServiceMock)
		slotMachine.AddInterfaceDependency(&authService)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddInterfaceDependency(&memoryCache)
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
		Payload: &rms.VCallRequest{
			CallType:     rms.CallTypeMethod,
			CallFlags:    rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			CallOutgoing: rms.NewReference(outgoing),

			Caller:         rms.NewReference(caller),
			Callee:         rms.NewReference(objectRef),
			CallSiteMethod: "New",
		},
		Meta: &rms.Meta{
			Sender: rms.NewReference(caller),
		},
	}
	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var (
			authService authentication.Service         = authentication.NewServiceMock(t)
			catalog     object.Catalog                 = catalogWrapper.Mock()
			memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
		)
		slotMachine.AddInterfaceDependency(&authService)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddInterfaceDependency(&memoryCache)
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
