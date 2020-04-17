// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/requestresult"
)

func TestVirtual_BasicOperations_WithoutExecutor(t *testing.T) {
	server := NewServer(t)
	ctx := inslogger.TestContext(t)

	prototype := gen.Reference()

	requestResult := requestresult.New([]byte("123"), gen.Reference())
	requestResult.SetActivate(gen.Reference(), prototype, []byte("234"))

	executorMock := testutils.NewMachineLogicExecutorMock(t)
	executorMock.CallConstructorMock.Return(nil, []byte("345"), nil)
	manager := executor.NewManager()
	if err := manager.RegisterExecutor(insolar.MachineTypeBuiltin, executorMock); err != nil {
		panic(err)
	}
	server.ReplaceMachinesManager(manager)

	cacheMock := descriptor.NewCacheMock(t)
	server.ReplaceCache(cacheMock)
	cacheMock.ByPrototypeRefMock.Return(
		descriptor.NewPrototypeDescriptor(gen.Reference(), gen.ID(), gen.Reference()),
		descriptor.NewCodeDescriptor(nil, insolar.MachineTypeBuiltin, gen.Reference()),
		nil,
	)

	pl := payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTConstructor,
		CallFlags:           nil,
		CallAsOf:            0,
		Caller:              insolar.Reference{},
		Callee:              gen.Reference(),
		CallSiteDeclaration: prototype,
		CallSiteMethod:      "test",
		CallSequence:        0,
		CallReason:          insolar.Reference{},
		RootTX:              insolar.Reference{},
		CallTX:              insolar.Reference{},
		CallRequestFlags:    0,
		KnownCalleeIncoming: insolar.Reference{},
		EntryHeadHash:       nil,
		CallOutgoing:        reference.Local{},
		Arguments:           nil,
	}

	plBytes, err := pl.Marshal()
	if err != nil {
		panic(err)
	}

	msg := payload.MustNewMessage(&payload.Meta{
		Polymorph:  uint32(payload.TypeMeta),
		Payload:    plBytes,
		Sender:     insolar.Reference{},
		Receiver:   insolar.Reference{},
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

func TestVirtual_BasicOperations_WithExecutor(t *testing.T) {
	server := NewServer(t)
	ctx := inslogger.TestContext(t)

	for i := 0; i < 10; i++ {

		pl := payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTConstructor,
			CallFlags:           nil,
			CallAsOf:            0,
			Caller:              insolar.Reference{},
			Callee:              gen.Reference(),
			CallSiteDeclaration: testwallet.GetPrototype(),
			CallSiteMethod:      "New",
			CallSequence:        0,
			CallReason:          insolar.Reference{},
			RootTX:              insolar.Reference{},
			CallTX:              insolar.Reference{},
			CallRequestFlags:    0,
			KnownCalleeIncoming: insolar.Reference{},
			EntryHeadHash:       nil,
			CallOutgoing:        gen.ID(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}

		plBytes, err := pl.Marshal()
		if err != nil {
			panic(err)
		}

		msg := payload.MustNewMessage(&payload.Meta{
			Polymorph:  uint32(payload.TypeMeta),
			Payload:    plBytes,
			Sender:     insolar.Reference{},
			Receiver:   insolar.Reference{},
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

// nolint:unused
type walletCreateResponse struct {
	Err     string `json:"error"`
	Ref     string `json:"reference"`
	TraceID string `json:"traceID"`
}

func unmarshalWalletCreateResponse(resp []byte) (walletCreateResponse, error) { // nolint:unused,deadcode
	result := walletCreateResponse{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return walletCreateResponse{}, errors.Wrap(err, "problem with unmarshaling response")
	}
	return result, nil
}

func TestAPICreate(t *testing.T) {
	server := NewServer(t)
	ctx := inslogger.TestContext(t)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		// verify and decode incoming message
		require.Len(t, messages, 1)

		metaPl := messages[0].Payload

		metaPlType, err := payload.UnmarshalType(metaPl)
		assert.NoError(t, err)
		assert.Equal(t, payload.TypeMeta, metaPlType)

		metaPayload, err := payload.Unmarshal(metaPl)
		assert.NoError(t, err)
		assert.IsType(t, &payload.Meta{}, metaPayload)

		callRequestPl := metaPayload.(*payload.Meta).Payload
		callRequestPlType, err := payload.UnmarshalType(callRequestPl)
		assert.NoError(t, err)
		assert.Equal(t, payload.TypeVCallRequest, callRequestPlType)

		callRequestPayloadBase, err := payload.Unmarshal(callRequestPl)
		assert.NoError(t, err)
		assert.IsType(t, &payload.VCallRequest{}, callRequestPayloadBase)

		callRequestPayload := callRequestPayloadBase.(*payload.VCallRequest)

		// construct and send VCallResult message to SMs
		callResultPayload := payload.VCallResult{
			Polymorph:          uint32(payload.TypeVCallResult),
			CallType:           callRequestPayload.CallType,
			CallFlags:          callRequestPayload.CallFlags,
			CallAsOf:           callRequestPayload.CallAsOf,
			Caller:             callRequestPayload.Caller,
			Callee:             callRequestPayload.Callee,
			ResultFlags:        nil,
			CallOutgoing:       callRequestPayload.CallOutgoing,
			CallIncoming:       reference.Local{},
			CallIncomingResult: reference.Local{},
			ReturnArguments:    nil,
		}

		callResultPayloadBytes, err := callResultPayload.Marshal()
		if err != nil {
			panic(err)
		}

		msg := payload.MustNewMessage(&payload.Meta{
			Polymorph:  uint32(payload.TypeMeta),
			Payload:    callResultPayloadBytes,
			Sender:     insolar.Reference{},
			Receiver:   insolar.Reference{},
			Pulse:      server.GetPulse().PulseNumber,
			ID:         nil,
			OriginHash: payload.MessageHash{},
		})

		server.SendMessage(ctx, msg)

		return nil
	}

	code, byteBuffer := server.CallCreateWallet()
	if !assert.Equal(t, 200, code) {
		t.Log(string(byteBuffer))
	} else {
		walletResponse, err := unmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)
		assert.NotEmpty(t, walletResponse.Err)
		assert.Empty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)
	}

}

func TestAPICreate_WithExecutor(t *testing.T) {
	server := NewServer(t)
	ctx := inslogger.TestContext(t)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		// verify and decode incoming message
		assert.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])

		return nil
	}

	code, byteBuffer := server.CallCreateWallet()
	if !assert.Equal(t, 200, code) {
		t.Log(string(byteBuffer))
	} else {
		walletResponse, err := unmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, walletResponse.Err)
		assert.NotEmpty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)
	}
}
