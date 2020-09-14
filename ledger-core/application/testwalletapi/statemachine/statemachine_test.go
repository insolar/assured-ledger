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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
)

func TestBuiltinTestAPIEchoValue(t *testing.T) {
	ref, err := reference.Decode(BuiltinTestAPIEcho)
	require.NoError(t, err)
	assert.Equal(t, pulse.BuiltinContract, ref.GetBase().Pulse())

	ref, err = reference.Decode(BuiltinTestAPIBriefEcho)
	require.NoError(t, err)
	assert.Equal(t, pulse.BuiltinContract, ref.GetBase().Pulse())
}

func TestSMTestAPICall_MethodResends(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t)

	request := rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		Callee:         rms.NewReference(gen.UniqueGlobalRef()),
		CallFlags:      rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
		CallSiteMethod: "New",
		Arguments:      rms.NewBytes([]byte("some args")),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	var memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
	slotMachine.AddInterfaceDependency(&memoryCache)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	p1 := slotMachine.PulseSlot.CurrentPulseNumber()

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	messageSent := make(chan struct{}, 1)
	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
		res := msg.(*rms.VCallRequest)

		// ensure that both times request is the same
		assert.Equal(t, APICaller, res.Caller.GetValue())
		assert.Equal(t, APICaller.GetBase(), res.CallOutgoing.GetValue().GetBase())
		assert.Equal(t, p1, res.CallOutgoing.GetPulseOfLocal())
		assert.Equal(t, affinity.DynamicRoleVirtualExecutor, role)
		assert.Equal(t, request.Callee.GetValue(), object)

		messageSent <- struct{}{}
		return nil
	})

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	testutils.WaitSignalsTimed(t, 10*time.Second, messageSent)

	slotMachine.Migrate()
	slotMachine.RunTil(smWrapper.AfterAnyMigrate())

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	testutils.WaitSignalsTimed(t, 10*time.Second, messageSent)

	response := &rms.VCallResult{
		Caller:          rms.NewReference(gen.UniqueGlobalRef()),
		Callee:          rms.NewReference(gen.UniqueGlobalRef()),
		ReturnArguments: rms.NewBytes([]byte("some results")),
	}

	outgoingRef := smRequest.requestPayload.CallOutgoing
	_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(outgoingRef)
	require.NotNil(t, bargeIn)

	// simulate received VCallResult
	require.True(t, bargeIn.CallWithParam(response))

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
	assert.Equal(t, []byte("some results"), smRequest.responsePayload)
	slotMachine.RunTil(smWrapper.AfterStop())
}

func TestSMTestAPICall_MethodEcho(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t)

	echoRef, err := reference.GlobalFromString(BuiltinTestAPIEcho)
	require.NoError(t, err)

	request := rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		Callee:         rms.NewReference(echoRef),
		CallFlags:      rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
		CallSiteMethod: "can be any",
		Arguments:      rms.NewBytes([]byte("some args")),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	var memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
	slotMachine.AddInterfaceDependency(&memoryCache)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := &SMTestAPICall{
		requestPayload: request,
	}

	ch := make(chan []byte, 1)
	_, ok := slotMachine.SlotMachine.AddNew(ctx, smRequest, smachine.CreateDefaultValues{
		TerminationHandler: func(data smachine.TerminationData) {
			ch <- data.Result.([]byte)
			close(ch)
		},
	})
	require.True(t, ok)

	slotMachine.RunTil(predicate.OnAnyRecycle)
	slotMachine.Continue()

	assert.Equal(t, request.Arguments.GetBytes(), <-ch)
}

func TestSMTestAPICall_Constructor(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.New(ctx, t)

	request := rms.VCallRequest{
		CallType:       rms.CallTypeConstructor,
		Callee:         rms.NewReference(gen.UniqueGlobalRef()),
		CallFlags:      rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
		CallSiteMethod: "New",
		Arguments:      rms.NewBytes([]byte("some args")),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	var memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
	slotMachine.AddInterfaceDependency(&memoryCache)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	p1 := slotMachine.PulseSlot.CurrentPulseNumber()

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	messageSent := make(chan struct{}, 1)
	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
		res := msg.(*rms.VCallRequest)

		// ensure that both times request is the same
		assert.Equal(t, APICaller, res.Caller.GetValue())
		assert.Equal(t, APICaller.GetBase(), res.CallOutgoing.GetValue().GetBase())
		assert.Equal(t, p1, res.CallOutgoing.GetPulseOfLocal())
		assert.Equal(t, affinity.DynamicRoleVirtualExecutor, role)
		assert.Equal(t, reference.NewSelf(res.CallOutgoing.GetValue().GetLocal()), object)

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

	request := rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		Callee:         rms.NewReference(gen.UniqueGlobalRef()),
		CallFlags:      rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
		CallSiteMethod: "New",
		Arguments:      rms.NewBytes([]byte("some args")),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	var memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)
	slotMachine.AddInterfaceDependency(&memoryCache)

	slotMachine.Start()
	defer slotMachine.Stop()

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	messageSent := make(chan struct{}, 1)
	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
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
	assert.Equal(t, res, smRequest.responsePayload)
	slotMachine.RunTil(smWrapper.AfterStop())
}
