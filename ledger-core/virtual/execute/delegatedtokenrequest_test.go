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
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	messageSender "github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/virtualdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
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
		caller    = slotMachine.GenerateGlobal()
		callee    = slotMachine.GenerateGlobal()
		outgoing  = reference.NewRecordOf(caller, slotMachine.GenerateLocal())
		objectRef = reference.NewSelf(outgoing.GetLocal())
		tokenKey  = DelegationTokenAwaitKey{outgoing}

		migrationPulse pulse.Number

		smExecute = SMExecute{
			Payload: &payload.VCallRequest{
				CallType:     payload.CallTypeConstructor,
				CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
				CallOutgoing: outgoing,

				Caller: caller,
				Callee: callee,
			},
		}
	)

	{
		catalogWrapper := object.NewCatalogMockWrapper(mc)
		var (
			catalog     object.Catalog         = catalogWrapper.Mock()
			authService authentication.Service = authentication.NewServiceMock(t)
			memoryCache memorycache.Service    = memorycache.NewServiceMock(t)
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
		func(_ context.Context, msg payload.Marshaler, role affinity.DynamicRole, object reference.Global, pn pulse.Number, _ ...messageSender.SendOption) error {
			res, ok := msg.(*payload.VDelegatedCallRequest)
			require.True(t, ok)
			require.NotNil(t, res)
			require.Equal(t, objectRef, object)
			require.Equal(t, migrationPulse, pn)
			return nil
		})

	{
		slotMachine.RunnerMock.AddExecutionClassify(outgoing.String(), contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}, nil)
		slotMachine.RunnerMock.AddExecutionMock(outgoing.String()).AddStart(
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

		ok := bargeInHolder.CallWithParam(&payload.VDelegatedCallResponse{
			ResponseDelegationSpec: payload.CallDelegationToken{Outgoing: outgoing},
		})
		require.True(t, ok)
	}

	{
		slotMachine.RunTil(smWrapper.AfterStep(smExecute.stepAfterTokenGet.Transition))

		require.NotNil(t, smExecute.delegationTokenSpec)
		require.Equal(t, outgoing, smExecute.delegationTokenSpec.Outgoing)
	}

	{
		// global alias should be cleanup'd here
		slotLink, _ := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(tokenKey)
		require.True(t, slotLink.IsZero())
	}

	mc.Finish()

}
