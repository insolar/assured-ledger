// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"context"
	"testing"
	"time"

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
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/callflag"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/mock"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func wrapVCallRequest(pulseNumber insolar.PulseNumber, pl payload.VCallRequest) (*message.Message, error) {
	plBytes, err := pl.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal VCallRequest")
	}

	msg, err := payload.NewMessage(&payload.Meta{
		Polymorph:  uint32(payload.TypeMeta),
		Payload:    plBytes,
		Sender:     reference.Global{},
		Receiver:   reference.Global{},
		Pulse:      pulseNumber,
		ID:         nil,
		OriginHash: payload.MessageHash{},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new message")
	}

	return msg, nil
}

func Method_PrepareObject(ctx context.Context, server *utils.Server, prototype reference.Global, object reference.Local) error {
	pl := payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTConstructor,
		CallFlags:           0,
		CallAsOf:            0,
		Caller:              server.GlobalCaller(),
		Callee:              reference.Global{},
		CallSiteDeclaration: prototype,
		CallSiteMethod:      "New",
		CallSequence:        0,
		CallReason:          reference.Global{},
		RootTX:              reference.Global{},
		CallTX:              reference.Global{},
		CallRequestFlags:    0,
		KnownCalleeIncoming: reference.Global{},
		EntryHeadHash:       nil,
		CallOutgoing:        object,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}
	msg, err := wrapVCallRequest(server.GetPulse().PulseNumber, pl)
	if err != nil {
		return errors.Wrap(err, "failed to construct VCallRequest message")
	}

	requestIsDone := make(chan error, 0)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		var err error
		defer func() { requestIsDone <- err }()

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		if err != nil {
			return nil
		}

		switch pl.(type) {
		case *payload.VCallResult:
			return nil
		default:
			err = errors.Errorf("bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
			return nil
		}
	}

	server.SendMessage(ctx, msg)

	select {
	case err := <-requestIsDone:
		return err
	case <-time.After(3 * time.Second):
		return errors.New("timeout")
	}
}

func TestVirtual_Method_WithoutExecutor(t *testing.T) {

	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	prototype := testwallet.GetPrototype()
	objectLocal := server.RandomLocalWithPulse()
	objectGlobal := reference.NewSelf(objectLocal)

	err := Method_PrepareObject(ctx, server, prototype, objectLocal)
	require.NoError(t, err)

	{
		pl := payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTConstructor,
			CallFlags:           0,
			CallAsOf:            0,
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: prototype,
			CallSiteMethod:      "New",
			CallRequestFlags:    0,
			CallOutgoing:        server.RandomLocalWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		msg, err := wrapVCallRequest(server.GetPulse().PulseNumber, pl)
		require.NoError(t, err)

		requestIsDone := make(chan struct{}, 0)

		server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
			defer func() { requestIsDone <- struct{}{} }()

			pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
			require.NoError(t, err)

			switch pl.(type) {
			case *payload.VCallResult:
			default:
				require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
			}

			// var returnValue []interface{}
			// insolar.MustDeserialize(callResultPayload.(*payload.VCallResult).ReturnArguments, returnValue)

			return nil
		}

		server.SendMessage(ctx, msg)

		select {
		case <-requestIsDone:
		case <-time.After(3 * time.Second):
			require.Failf(t, "", "timeout")
		}
	}
}

func TestVirtual_Method_WithoutExecutor_Unordered(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	var (
		waitInputChannel  = make(chan struct{}, 2)
		waitOutputChannel = make(chan struct{}, 0)
	)

	executorMock := testutils.NewMachineLogicExecutorMock(t)
	executorMock.CallConstructorMock.Return(gen.Reference().AsBytes(), []byte("345"), nil)
	executorMock.CallMethodMock.Set(func(
		ctx context.Context, callContext *insolar.LogicCallContext, code reference.Global,
		data []byte, method string, args insolar.Arguments,
	) (
		newObjectState []byte, methodResults insolar.Arguments, err error,
	) {
		// tell the test that we know about next request
		waitInputChannel <- struct{}{}

		// wait the test result
		<-waitOutputChannel

		return []byte("456"), []byte("345"), nil
	})

	manager := executor.NewManager()
	err := manager.RegisterExecutor(insolar.MachineTypeBuiltin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(manager)

	prototype := gen.Reference()
	cacheMock := mock.NewDescriptorsCacheMockWrapper(t)
	cacheMock.AddPrototypeCodeDescriptor(prototype, gen.ID(), gen.Reference())
	cacheMock.IntenselyPanic = true
	server.ReplaceCache(cacheMock)

	objectLocal := gen.ID()
	err = Method_PrepareObject(ctx, server, prototype, objectLocal)
	require.NoError(t, err)

	{
		requestIsDone := make(chan struct{}, 2)
		server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
			defer func() { requestIsDone <- struct{}{} }()

			pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
			require.NoError(t, err)

			switch pl.(type) {
			case *payload.VCallResult:
			default:
				require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
			}

			return nil
		}

		for i := 0; i < 2; i++ {
			pl := payload.VCallRequest{
				Polymorph:           uint32(payload.TypeVCallRequest),
				CallType:            payload.CTMethod,
				CallFlags:           callflag.Unordered,
				CallAsOf:            0,
				Caller:              server.GlobalCaller(),
				Callee:              reference.NewSelf(objectLocal),
				CallSiteDeclaration: prototype,
				CallSiteMethod:      "GetBalance",
				CallRequestFlags:    0,
				CallOutgoing:        server.RandomLocalWithPulse(),
				Arguments:           insolar.MustSerialize([]interface{}{}),
			}
			msg, err := wrapVCallRequest(server.GetPulse().PulseNumber, pl)
			require.NoError(t, err)

			server.SendMessage(ctx, msg)
		}

		for i := 0; i < 2; i++ {
			select {
			case <-waitInputChannel:
			case <-time.After(3 * time.Second):
				require.Failf(t, "", "timeout")
			}
		}

		for i := 0; i < 2; i++ {
			waitOutputChannel <- struct{}{}

			select {
			case <-requestIsDone:
			case <-time.After(3 * time.Second):
				require.Failf(t, "", "timeout")
			}
		}
	}
}

func TestVirtual_Method_WithExecutor(t *testing.T) {
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

func TestVirtual_CallMethodAfterPulseChange(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	server.IncrementPulse(ctx)

	var (
		vCallRequestCount  uint
		vStateRequestCount uint
		vStateReportCount  uint
		vCallResultCount   uint
	)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		require.NoError(t, err)

		switch pl.(type) {
		case *payload.VCallResult:
			vCallResultCount++
		case *payload.VCallRequest:
			vCallRequestCount++
		case *payload.VStateReport:
			vStateRequestCount++
		case *payload.VStateRequest:
			vStateReportCount++
		default:
			require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
		}

		server.SendMessage(ctx, messages[0])
		return nil
	}

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := reference.NewRecordRef(server.RandomLocalWithPulse())
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(t, objectRef, stateID, rawWalletState)
		require.NoError(t, server.AddInput(ctx, msg))
	}

	// Change pulse to force send VStateRequest
	server.IncrementPulse(ctx)

	checkBalance(t, server, objectRef, testBalance)

	expectedNum := uint(1)
	require.Equal(t, expectedNum, vStateReportCount)
	require.Equal(t, expectedNum, vStateRequestCount)
	require.Equal(t, expectedNum, vCallRequestCount)
	require.Equal(t, expectedNum, vCallResultCount)
}
