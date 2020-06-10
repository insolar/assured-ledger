// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_Constructor_WithoutExecutor(t *testing.T) {
	t.Log("C4835")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	class := gen.UniqueReference()

	requestResult := requestresult.New([]byte("123"), gen.UniqueReference())
	requestResult.SetActivate(gen.UniqueReference(), class, []byte("234"))

	executorMock := machine.NewExecutorMock(mc)
	executorMock.CallConstructorMock.Return(nil, []byte("345"), nil)
	mgr := machine.NewManager()
	err := mgr.RegisterExecutor(machine.Builtin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(mgr)

	cacheMock := descriptor.NewCacheMock(mc)
	server.ReplaceCache(cacheMock)
	cacheMock.ByClassRefMock.Return(
		descriptor.NewClass(gen.UniqueReference(), gen.UniqueID(), gen.UniqueReference()),
		descriptor.NewCode(nil, machine.Builtin, gen.UniqueReference()),
		nil,
	)

	isolation := contract.ConstructorIsolation()

	outgoing := server.RandomLocalWithPulse()
	objectRef := reference.NewSelf(outgoing)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "test",
		CallOutgoing:   outgoing,
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, res.ReturnArguments, []byte("345"))
		require.Equal(t, res.Callee, objectRef)
		require.Equal(t, res.CallOutgoing, outgoing)

		return false // no resend msg
	})

	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()
	server.SendMessage(ctx, msg)

	assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))

	mc.Finish()
}

func TestVirtual_Constructor_WithExecutor(t *testing.T) {
	t.Log("C4835")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	isolation := contract.ConstructorIsolation()
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	outgoing := server.RandomLocalWithPulse()
	objectRef := reference.NewSelf(outgoing)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         testwallet.GetClass(),
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}

	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, res.Callee, objectRef)
		require.Equal(t, res.CallOutgoing, outgoing)

		return false // no resend msg
	})

	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()
	server.SendMessage(ctx, msg)

	assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))

	mc.Finish()
}

func TestVirtual_Constructor_HasStateWithMissingStatus(t *testing.T) {
	// VE has object's state record with Status==Missing
	// Constructor call should work on top of such entry
	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	class := gen.UniqueReference()

	requestResult := requestresult.New([]byte("123"), gen.UniqueReference())
	requestResult.SetActivate(gen.UniqueReference(), class, []byte("234"))

	executorMock := machine.NewExecutorMock(mc)
	executorMock.CallConstructorMock.
		Inspect(func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) {
			require.Equal(t, "New", name)
			require.Equal(t, []byte("arguments"), args)
		}).
		Return(nil, []byte("345"), nil)

	mgr := machine.NewManager()
	err := mgr.RegisterExecutor(machine.Builtin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(mgr)

	cacheMock := descriptor.NewCacheMock(mc)
	server.ReplaceCache(cacheMock)
	cacheMock.ByClassRefMock.Return(
		descriptor.NewClass(gen.UniqueReference(), gen.UniqueID(), gen.UniqueReference()),
		descriptor.NewCode(nil, machine.Builtin, gen.UniqueReference()),
		nil,
	)

	outgoing := server.RandomLocalWithPulse()
	objectRef := reference.NewSelf(outgoing)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, res.ReturnArguments, []byte("345"))
		require.Equal(t, res.Callee, objectRef)
		require.Equal(t, res.CallOutgoing, outgoing)

		return false // no resend msg
	})

	msg := makeVStateReportWithState(server.GetPulse().PulseNumber, objectRef, payload.Missing, nil)
	server.SendMessage(ctx, msg)

	isolation := contract.ConstructorIsolation()

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "New",
		CallSequence:   0,
		CallOutgoing:   outgoing,
		Arguments:      []byte("arguments"),
	}

	msg = utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()
	server.SendMessage(ctx, msg)

	assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))

	mc.Finish()
}
