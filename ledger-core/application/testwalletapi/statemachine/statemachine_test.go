// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func TestSMTestAPICall_Migrate_After_RegisterBargeIn(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.NewWithIgnoreAllErrors(ctx, t)

	request := payload.VCallRequest{
		CallType:            payload.CTMethod,
		Callee:              gen.UniqueGlobalRef(),
		Caller:              gen.UniqueGlobalRef(),
		CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        gen.UniqueGlobalRef(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepRegisterBargeIn))

	require.NotEqual(t, APICaller, smRequest.requestPayload.Caller)
	slotMachine.RunTil(smWrapper.AfterStep(smRequest.stepRegisterBargeIn))

	outgoingRef := smRequest.requestPayload.CallOutgoing
	_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(outgoingRef)
	require.NotNil(t, bargeIn)

	require.Equal(t, APICaller, smRequest.requestPayload.Caller)
	require.NotEmpty(t, smRequest.requestPayload.CallOutgoing)

	{
		slotMachine.Migrate()
		slotMachine.RunTil(smWrapper.AfterAnyMigrate())
		// check that we remove bargein after migrate
		_, bargeIn = slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(outgoingRef)
		require.Nil(t, bargeIn)
	}

	slotMachine.RunTil(smWrapper.AfterStep(smRequest.stepRegisterBargeIn))
	newOutgoingRef := smRequest.requestPayload.CallOutgoing
	var newBargeIn smachine.BargeInHolder
	{
		require.NotEqual(t, newOutgoingRef, outgoingRef)
		// check that we create new bargein
		_, newBargeIn = slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(newOutgoingRef)
		require.NotNil(t, newBargeIn)
	}
}

func TestSMTestAPICall_Migrate_After_SendRequest(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.NewWithIgnoreAllErrors(ctx, t)

	request := payload.VCallRequest{
		CallType:            payload.CTMethod,
		Callee:              gen.UniqueGlobalRef(),
		Caller:              gen.UniqueGlobalRef(),
		CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        gen.UniqueGlobalRef(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	slotMachine.RunTil(smWrapper.AfterStep(smRequest.stepRegisterBargeIn))

	outgoingRef := smRequest.requestPayload.CallOutgoing
	_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(outgoingRef)
	require.NotNil(t, bargeIn)

	messageSent := make(chan struct{}, 1)
	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg payload.Marshaler, role node.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
		res := msg.(*payload.VCallRequest)
		// ensure that both times request is the same
		assert.Equal(t, outgoingRef, res.CallOutgoing)
		assert.Equal(t, node.DynamicRoleVirtualExecutor, role)
		assert.Equal(t, request.Callee, object)

		messageSent <- struct{}{}
		return nil
	})

	slotMachine.RunTil(smWrapper.AfterStep(smRequest.stepSendRequest))
	testutils.WaitSignalsTimed(t, 10*time.Second, messageSent)
	slotMachine.Migrate()

	slotMachine.RunTil(smWrapper.AfterStep(smRequest.stepSendRequest))
	testutils.WaitSignalsTimed(t, 10*time.Second, messageSent)

	response := &payload.VCallResult{
		Caller:   gen.UniqueGlobalRef(),
		Callee:   gen.UniqueGlobalRef(),
		CallAsOf: gen.PulseNumber(),
	}

	// simulate received VCallResult
	require.True(t, bargeIn.CallWithParam(response))

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	require.Equal(t, *response, smRequest.responsePayload)
	slotMachine.RunTil(smWrapper.AfterStop())
}
