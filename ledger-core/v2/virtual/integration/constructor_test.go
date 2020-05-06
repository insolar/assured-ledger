// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func TestVirtual_Constructor_WithoutExecutor(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	prototype := gen.Reference()

	requestResult := requestresult.New([]byte("123"), gen.Reference())
	requestResult.SetActivate(gen.Reference(), prototype, []byte("234"))

	executorMock := testutils.NewMachineLogicExecutorMock(t)
	executorMock.CallConstructorMock.Return(nil, []byte("345"), nil)
	manager := executor.NewManager()
	err := manager.RegisterExecutor(insolar.MachineTypeBuiltin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(manager)

	cacheMock := descriptor.NewCacheMock(t)
	server.ReplaceCache(cacheMock)
	cacheMock.ByPrototypeRefMock.Return(
		descriptor.NewPrototype(gen.Reference(), gen.ID(), gen.Reference()),
		descriptor.NewCode(nil, insolar.MachineTypeBuiltin, gen.Reference()),
		nil,
	)

	pl := payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTConstructor,
		CallFlags:           0,
		CallAsOf:            0,
		Caller:              reference.Global{},
		Callee:              gen.Reference(),
		CallSiteDeclaration: prototype,
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

	plBytes, err := pl.Marshal()
	require.NoError(t, err)

	msg := payload.MustNewMessage(&payload.Meta{
		Polymorph:  uint32(payload.TypeMeta),
		Payload:    plBytes,
		Sender:     reference.Global{},
		Receiver:   reference.Global{},
		Pulse:      server.GetPulse().PulseNumber,
		ID:         nil,
		OriginHash: payload.MessageHash{},
	})

	testIsDone := make(chan struct{}, 0)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		assert.Len(t, messages, 1)

		var (
			_      = messages[0].Metadata
			metaPl = messages[0].Payload
		)

		metaPlType, err := payload.UnmarshalType(metaPl)
		assert.NoError(t, err)
		assert.Equal(t, payload.TypeMeta, metaPlType)

		metaPayload, err := payload.Unmarshal(metaPl)
		assert.NoError(t, err)
		assert.IsType(t, &payload.Meta{}, metaPayload)

		callResultPl := metaPayload.(*payload.Meta).Payload
		callResultPlType, err := payload.UnmarshalType(callResultPl)
		assert.NoError(t, err)
		assert.Equal(t, payload.TypeVCallResult, callResultPlType)

		callResultPayload, err := payload.Unmarshal(callResultPl)
		assert.NoError(t, err)
		assert.IsType(t, &payload.VCallResult{}, callResultPayload)

		assert.Equal(t, callResultPayload.(*payload.VCallResult).ReturnArguments, []byte("345"))

		testIsDone <- struct{}{}

		return nil
	}

	server.SendMessage(ctx, msg)

	<-testIsDone
}

func TestVirtual_Constructor_WithExecutor(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	for i := 0; i < 10; i++ {

		pl := payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTConstructor,
			CallFlags:           0,
			CallAsOf:            0,
			Caller:              reference.Global{},
			Callee:              gen.Reference(),
			CallSiteDeclaration: testwallet.GetPrototype(),
			CallSiteMethod:      "New",
			CallSequence:        0,
			CallReason:          reference.Global{},
			RootTX:              reference.Global{},
			CallTX:              reference.Global{},
			CallRequestFlags:    0,
			KnownCalleeIncoming: reference.Global{},
			EntryHeadHash:       nil,
			CallOutgoing:        gen.ID(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}

		plBytes, err := pl.Marshal()
		require.NoError(t, err)

		msg := payload.MustNewMessage(&payload.Meta{
			Polymorph:  uint32(payload.TypeMeta),
			Payload:    plBytes,
			Sender:     reference.Global{},
			Receiver:   reference.Global{},
			Pulse:      server.GetPulse().PulseNumber,
			ID:         nil,
			OriginHash: payload.MessageHash{},
		})

		testIsDone := make(chan struct{}, 0)

		server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
			assert.Len(t, messages, 1)

			var (
				_      = messages[0].Metadata
				metaPl = messages[0].Payload
			)

			metaPlType, err := payload.UnmarshalType(metaPl)
			assert.NoError(t, err)
			assert.Equal(t, payload.TypeMeta, metaPlType)

			metaPayload, err := payload.Unmarshal(metaPl)
			assert.NoError(t, err)
			assert.IsType(t, &payload.Meta{}, metaPayload)

			callResultPl := metaPayload.(*payload.Meta).Payload
			callResultPlType, err := payload.UnmarshalType(callResultPl)
			assert.NoError(t, err)
			assert.Equal(t, payload.TypeVCallResult, callResultPlType)

			callResultPayload, err := payload.Unmarshal(callResultPl)
			assert.NoError(t, err)
			assert.IsType(t, &payload.VCallResult{}, callResultPayload)

			testIsDone <- struct{}{}

			return nil
		}

		server.SendMessage(ctx, msg)

		<-testIsDone
	}
}
