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

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	messageSender "github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/slotdebugger"
)

func TestVDelegatedCallRequest(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.PrepareMockedRunner(ctx, t)
	slotMachine.PrepareMockedMessageSender(mc)

	var (
		caller         = gen.UniqueReference()
		callee         = gen.UniqueReference()
		outgoing       = gen.UniqueIDWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		objectGlobal   = reference.NewSelf(outgoing)
		outgoingGlobal = reference.NewRecordOf(callee, outgoing)
		tokenKey       = DelegationTokenAwaitKey{outgoingGlobal}

		migrationPulse pulse.Number

		smExecute = SMExecute{
			Payload: &payload.VCallRequest{
				CallType:     payload.CTConstructor,
				CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
				CallOutgoing: outgoing,

				Caller: caller,
				Callee: callee,
			},
		}
	)

	{
		catalogWrapper := object.NewCatalogMockWrapper(mc)
		var catalog object.Catalog = catalogWrapper.Mock()
		slotMachine.AddInterfaceDependency(&catalog)

		sharedStateData := smachine.NewUnboundSharedData(&object.SharedState{
			Info: object.Info{
				Reference:      objectGlobal,
				PendingTable:   object.NewRequestTable(),
				KnownRequests:  object.NewRequestTable(),
				ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
			},
		})
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectGlobal, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.MessageSender.SendRole.Set(
		func(_ context.Context, msg payload.Marshaler, role node.DynamicRole, object reference.Global, pn pulse.Number, _ ...messageSender.SendOption) error {
			res, ok := msg.(*payload.VDelegatedCallRequest)
			require.True(t, ok)
			require.NotNil(t, res)
			require.Equal(t, objectGlobal, object)
			require.Equal(t, migrationPulse, pn)
			return nil
		})

	{
		slotMachine.RunnerMock.AddExecutionClassify(outgoingGlobal.String(), contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}, nil)
		slotMachine.RunnerMock.AddExecutionMock(outgoingGlobal.String()).AddStart(
			nil,
			&executionupdate.ContractExecutionStateUpdate{
				Type:   executionupdate.Done,
				Result: requestresult.New([]byte("123"), objectGlobal),
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
			DelegationSpec: payload.CallDelegationToken{Outgoing: outgoingGlobal},
		})
		require.True(t, ok)
	}

	{
		slotMachine.RunTil(smWrapper.AfterStep(smExecute.stepAfterTokenGet.Transition))

		require.NotNil(t, smExecute.delegationTokenSpec)
		require.Equal(t, outgoingGlobal, smExecute.delegationTokenSpec.Outgoing)
	}

	{
		// global alias should be cleanup'd here
		slotLink, _ := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(tokenKey)
		require.True(t, slotLink.IsZero())
	}

	mc.Finish()

}
