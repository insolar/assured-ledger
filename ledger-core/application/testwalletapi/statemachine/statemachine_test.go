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

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func TestSMTestAPICall_MethodResends(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t)

	request := payload.VCallRequest{
		CallType:       payload.CTMethod,
		Callee:         gen.UniqueGlobalRef(),
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		CallSiteMethod: "New",
		Arguments:      []byte("some args"),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	p1 := slotMachine.PulseSlot.CurrentPulseNumber()

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	messageSent := make(chan struct{}, 1)
	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg payload.Marshaler, role affinity.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
		res := msg.(*payload.VCallRequest)
		// ensure that both times request is the same
		assert.Equal(t, APICaller, res.Caller)
		assert.Equal(t, APICaller.GetBase(), res.CallOutgoing.GetBase())
		assert.Equal(t, p1, res.CallOutgoing.GetLocal().GetPulseNumber())
		assert.Equal(t, affinity.DynamicRoleVirtualExecutor, role)
		assert.Equal(t, request.Callee, object)

		messageSent <- struct{}{}
		return nil
	})

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	testutils.WaitSignalsTimed(t, 10*time.Second, messageSent)

	slotMachine.Migrate()
	slotMachine.RunTil(smWrapper.AfterAnyMigrate())

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	testutils.WaitSignalsTimed(t, 10*time.Second, messageSent)

	response := &payload.VCallResult{
		Caller:          gen.UniqueGlobalRef(),
		Callee:          gen.UniqueGlobalRef(),
		CallAsOf:        gen.PulseNumber(),
		ReturnArguments: []byte("some results"),
	}

	outgoingRef := smRequest.requestPayload.CallOutgoing
	_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(outgoingRef)
	require.NotNil(t, bargeIn)

	// simulate received VCallResult
	require.True(t, bargeIn.CallWithParam(response))

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	require.Equal(t, []byte("some results"), smRequest.responsePayload)
	slotMachine.RunTil(smWrapper.AfterStop())
}

func TestSMTestAPICall_Constructor(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t)

	request := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		Callee:         gen.UniqueGlobalRef(),
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		CallSiteMethod: "New",
		Arguments:      []byte("some args"),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	p1 := slotMachine.PulseSlot.CurrentPulseNumber()

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	messageSent := make(chan struct{}, 1)
	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg payload.Marshaler, role affinity.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
		res := msg.(*payload.VCallRequest)

		// ensure that both times request is the same
		assert.Equal(t, APICaller, res.Caller)
		assert.Equal(t, APICaller.GetBase(), res.CallOutgoing.GetBase())
		assert.Equal(t, p1, res.CallOutgoing.GetLocal().GetPulseNumber())
		assert.Equal(t, affinity.DynamicRoleVirtualExecutor, role)
		assert.Equal(t, reference.NewSelf(res.CallOutgoing.GetLocal()), object)

		messageSent <- struct{}{}
		return nil
	})

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	testutils.WaitSignalsTimed(t, 10*time.Second, messageSent)
}

const expectedMaxRetries = 3

func TestSMTestAPICall_RetriesExceeded(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t)

	request := payload.VCallRequest{
		CallType:       payload.CTMethod,
		Callee:         gen.UniqueGlobalRef(),
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		CallSiteMethod: "New",
		Arguments:      []byte("some args"),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	messageSent := make(chan struct{}, 1)
	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg payload.Marshaler, role affinity.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
		messageSent <- struct{}{}
		return nil
	})

	for i := 0; i < expectedMaxRetries; i++ {
		slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
		testutils.WaitSignalsTimed(t, 10*time.Second, messageSent)
		slotMachine.Migrate()
		slotMachine.RunTil(smWrapper.AfterAnyMigrate())
	}

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	res, err := foundation.MarshalMethodErrorResult(throw.New("timeout: exceeded resend limit"))
	require.NoError(t, err)
	require.Equal(t, res, smRequest.responsePayload)
	slotMachine.RunTil(smWrapper.AfterStop())
}
