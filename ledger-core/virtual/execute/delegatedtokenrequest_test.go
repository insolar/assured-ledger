// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	messageSender "github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/virtualdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
	"github.com/insolar/assured-ledger/ledger-core/virtual/vnlmn"
)

func TestVDelegatedCallRequest(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := virtualdebugger.NewWithIgnoreAllError(ctx, t)
	slotMachine.PrepareMockedRunner(ctx, t)
	slotMachine.PrepareMockedMessageSender(mc)

	var (
		caller   = slotMachine.GenerateGlobal()
		callee   = slotMachine.GenerateGlobal()
		outgoing = reference.NewRecordOf(caller, slotMachine.GenerateLocal())
		tokenKey = DelegationTokenAwaitKey{outgoing}
		meRef    = gen.UniqueGlobalRef()

		migrationPulse pulse.Number

		smExecute = SMExecute{
			Payload: &rms.VCallRequest{
				CallType:     rms.CallTypeConstructor,
				CallFlags:    rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
				CallOutgoing: rms.NewReference(outgoing),
				Caller:       rms.NewReference(caller),
				Callee:       rms.NewReference(callee),
			},
		}

		objectRef = testutils.GetObjectReference(smExecute.Payload, meRef)
	)

	{
		catalogWrapper := object.NewCatalogMockWrapper(mc)
		var (
			catalog     object.Catalog                 = catalogWrapper.Mock()
			authService authentication.Service         = authentication.NewServiceMock(t)
			memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
		)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddInterfaceDependency(&authService)
		slotMachine.AddInterfaceDependency(&memoryCache)
		limiter := tool.NewRunnerLimiter(4)
		slotMachine.AddDependency(limiter)

		sharedStateData := smachine.NewUnboundSharedData(&object.SharedState{
			Info: object.Info{
				Reference:      objectRef,
				PendingTable:   callregistry.NewRequestTable(),
				KnownRequests:  callregistry.NewWorkingTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
			},
		})

		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.MessageSender.SendRole.Set(
		func(_ context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, _ ...messageSender.SendOption) error {
			switch res := msg.(type) {
			case *rms.VDelegatedCallRequest:
				require.NotNil(t, res)
				require.Equal(t, objectRef, object)
				require.Equal(t, migrationPulse, pn)
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
			return nil
		})

	{
		slotMachine.RunnerMock.AddExecutionClassify(outgoing, contract.MethodIsolation{
			Interference: isolation.CallTolerable,
			State:        isolation.CallDirty,
		}, nil)
		slotMachine.RunnerMock.AddExecutionMock(outgoing).AddStart(
			nil,
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("123"), outgoing),
			})
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)
	{

		slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepExecuteStart))
		slotMachine.Migrate()
		migrationPulse = slotMachine.PulseSlot.CurrentPulseNumber()

		var smDelegatedTokenRequest SMDelegatedTokenRequest
		slotMachine.RunTil(smWrapper.AfterStep(smDelegatedTokenRequest.stepSendRequest))

		slotLink, bargeInHolder := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(tokenKey)
		require.False(t, slotLink.IsZero())

		ok := bargeInHolder.CallWithParam(&rms.VDelegatedCallResponse{
			ResponseDelegationSpec: rms.CallDelegationToken{Outgoing: rms.NewReference(outgoing)},
		})
		require.True(t, ok)
	}

	{
		slotMachine.RunTil(smWrapper.AfterStep(smExecute.stepAfterTokenGet.Transition))

		require.NotNil(t, smExecute.delegationTokenSpec)
		require.Equal(t, outgoing, smExecute.delegationTokenSpec.Outgoing.GetValue())
	}

	{
		// global alias should be cleanup'd here
		slotLink, _ := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(tokenKey)
		require.True(t, slotLink.IsZero())
	}

	mc.Finish()

}
