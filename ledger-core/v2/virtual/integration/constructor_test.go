// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/rms"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func TestVirtual_Constructor_WithoutExecutor(t *testing.T) {
	t.Log("C4835")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	class := gen.UniqueReference()

	requestResult := requestresult.New([]byte("123"), gen.UniqueReference())
	requestResult.SetActivate(gen.UniqueReference(), class, []byte("234"))

	executorMock := machine.NewExecutorMock(t)
	executorMock.CallConstructorMock.Return(nil, []byte("345"), nil)
	mgr := machine.NewManager()
	err := mgr.RegisterExecutor(machine.Builtin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(mgr)

	cacheMock := descriptor.NewCacheMock(t)
	server.ReplaceCache(cacheMock)
	cacheMock.ByClassRefMock.Return(
		descriptor.NewClass(gen.UniqueReference(), gen.UniqueID(), gen.UniqueReference()),
		descriptor.NewCode(nil, machine.Builtin, gen.UniqueReference()),
		nil,
	)

	isolation := contract.ConstructorIsolation()

	pl := payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
		CallAsOf:            0,
		Caller:              reference.Global{},
		Callee:              class,
		CallSiteMethod:      "test",
		CallSequence:        0,
		CallReason:          reference.Global{},
		RootTX:              reference.Global{},
		CallTX:              reference.Global{},
		CallRequestFlags:    0,
		KnownCalleeIncoming: reference.Global{},
		EntryHeadHash:       nil,
		CallOutgoing:        reference.Local{},
		Arguments:           nil,
	}

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		assert.Len(t, messages, 1)

		var (
			_      = messages[0].Metadata
			metaPl = messages[0].Payload
		)

		metaType, metaPayload, err := rms.Unmarshal(metaPl)

		assert.NoError(t, err)
		assert.Equal(t, uint64(payload.TypeMetaPolymorthID), metaType)
		assert.NoError(t, err)
		assert.IsType(t, &payload.Meta{}, metaPayload)

		callResultPl := metaPayload.(*payload.Meta).Payload
		callResultType, callResultPayload, err := rms.Unmarshal(callResultPl)

		assert.NoError(t, err)
		assert.Equal(t, uint64(payload.TypeVCallResultPolymorthID), callResultType)
		assert.IsType(t, &payload.VCallResult{}, callResultPayload)

		assert.Equal(t, callResultPayload.(*payload.VCallResult).ReturnArguments, []byte("345"))

		return nil
	})

	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()
	server.SendMessage(ctx, msg)

	assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))
}

func TestVirtual_Constructor_WithExecutor(t *testing.T) {
	t.Log("C4835")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	isolation := contract.ConstructorIsolation()

	for i := 0; i < 10; i++ {
		pl := payload.VCallRequest{
			CallType:            payload.CTConstructor,
			CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
			CallAsOf:            0,
			Caller:              reference.Global{},
			Callee:              testwallet.GetClass(),
			CallSiteMethod:      "New",
			CallSequence:        0,
			CallReason:          reference.Global{},
			RootTX:              reference.Global{},
			CallTX:              reference.Global{},
			CallRequestFlags:    0,
			KnownCalleeIncoming: reference.Global{},
			EntryHeadHash:       nil,
			CallOutgoing:        gen.UniqueID(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}

		server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
			assert.Len(t, messages, 1)

			var (
				_      = messages[0].Metadata
				metaPl = messages[0].Payload
			)

			metaType, metaPayload, err := rms.Unmarshal(metaPl)

			assert.NoError(t, err)
			assert.Equal(t, uint64(payload.TypeMetaPolymorthID), metaType)
			assert.NoError(t, err)
			assert.IsType(t, &payload.Meta{}, metaPayload)

			callResultPl := metaPayload.(*payload.Meta).Payload
			callResultType, callResultPayload, err := rms.Unmarshal(callResultPl)

			assert.NoError(t, err)
			assert.Equal(t, uint64(payload.TypeVCallResultPolymorthID), callResultType)
			assert.IsType(t, &payload.VCallResult{}, callResultPayload)

			return nil
		})

		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()
		server.SendMessage(ctx, msg)

		assert.True(t, server.PublisherMock.WaitCount(1, 10*time.Second))
	}
}
