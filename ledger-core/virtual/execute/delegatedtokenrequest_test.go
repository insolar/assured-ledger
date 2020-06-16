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
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/slotdebugger"
)

func TestVirtual_CDelegatedCallRequest(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.PrepareRunner(ctx, mc)
	slotMachine.PrepareMockedMessageSender(mc)

	var (
		caller         = gen.UniqueReference()
		callee         = gen.UniqueReference()
		outgoing       = gen.UniqueIDWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		objectRef      = reference.NewSelf(outgoing)
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

	catalogWrapper := object.NewCatalogMockWrapper(mc)

	{
		var (
			catalog     object.Catalog         = catalogWrapper.Mock()
			authService authentication.Service = authentication.NewServiceMock(t)
			sharedState                        = &object.SharedState{
				Info: object.Info{
					Reference:      objectRef,
					PendingTable:   object.NewRequestTable(),
					KnownRequests:  object.NewRequestTable(),
					ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
					OrderedExecute: smsync.NewConditional(1, "MutableExecution").SyncLink(),
				},
			}
		)
		slotMachine.AddInterfaceDependency(&catalog)
		slotMachine.AddInterfaceDependency(&authService)

		sharedStateData := smachine.NewUnboundSharedData(sharedState)
		smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

		catalogWrapper.AddObject(objectRef, smObjectAccessor)
		catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
	}

	slotMachine.MessageSender.SendRole.Set(
		func(_ context.Context, msg payload.Marshaler, role node.DynamicRole, object reference.Global, pn pulse.Number, _ ...messageSender.SendOption) error {
			res, ok := msg.(*payload.VDelegatedCallRequest)
			require.True(t, ok)
			require.NotNil(t, res)
			require.Equal(t, objectRef, object)
			require.Equal(t, migrationPulse, pn)
			return nil
		})

	slotMachine.Start()
	defer slotMachine.Stop()
	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)
	{

		slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepExecuteStart))
		slotMachine.Migrate()
		migrationPulse = slotMachine.PulseSlot.CurrentPulseNumber()
		slotMachine.RunTil(smWrapper.AfterStep(smExecute.stepGetDelegationToken))
		var smDelegatedTokenRequest SMDelegatedTokenRequest
		slotMachine.RunTil(smWrapper.AfterStep(smDelegatedTokenRequest.stepSendRequest))
		slotLink, bargeInHolder := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(DelegationTokenAwaitKey{smExecute.execution.Outgoing})

		require.False(t, slotLink.IsZero())
		require.True(t, bargeInHolder.CallWithParam(&payload.VDelegatedCallResponse{
			DelegationSpec: payload.CallDelegationToken{Outgoing: smExecute.execution.Outgoing},
		}))
	}
	{
		slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepAfterTokenGet.Transition))

		require.NotNil(t, smExecute.delegationTokenSpec)
		require.Equal(t, smExecute.execution.Outgoing, smExecute.delegationTokenSpec.Outgoing)
	}
	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()

}
