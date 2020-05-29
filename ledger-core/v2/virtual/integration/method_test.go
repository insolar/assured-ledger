// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/rms"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

func wrapVCallRequest(pulseNumber pulse.Number, pl payload.VCallRequest) (*message.Message, error) {
	plBytes, err := pl.Marshal()
	if err != nil {
		return nil, errors.W(err, "failed to marshal VCallRequest")
	}

	msg, err := payload.NewMessage(&payload.Meta{
		Payload:    plBytes,
		Sender:     reference.Global{},
		Receiver:   reference.Global{},
		Pulse:      pulseNumber,
		ID:         nil,
		OriginHash: payload.MessageHash{},
	})
	if err != nil {
		return nil, errors.W(err, "failed to create new message")
	}

	return msg, nil
}

func Method_PrepareObject(ctx context.Context, server *utils.Server, class reference.Global, object reference.Local) error {
	isolation := contract.ConstructorIsolation()

	pl := payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
		CallAsOf:            0,
		Caller:              server.GlobalCaller(),
		Callee:              reference.Global{},
		CallSiteDeclaration: class,
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
		return errors.W(err, "failed to construct VCallRequest message")
	}

	requestIsDone := make(chan error, 0)

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
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
	})

	server.SendMessage(ctx, msg)

	select {
	case err := <-requestIsDone:
		return err
	case <-time.After(10 * time.Second):
		return errors.New("timeout")
	}
}

func TestVirtual_Method_WithoutExecutor(t *testing.T) {
	t.Log("C4923")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	class := testwallet.GetClass()
	objectLocal := server.RandomLocalWithPulse()
	objectGlobal := reference.NewSelf(objectLocal)

	err := Method_PrepareObject(ctx, server, class, objectLocal)
	require.NoError(t, err)

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			CallAsOf:            0,
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: class,
			CallSiteMethod:      "GetBalance",
			CallRequestFlags:    0,
			CallOutgoing:        server.RandomLocalWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		msg, err := wrapVCallRequest(server.GetPulse().PulseNumber, pl)
		require.NoError(t, err)

		requestIsDone := make(chan struct{}, 0)

		server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
			defer func() { requestIsDone <- struct{}{} }()

			pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
			require.NoError(t, err)

			switch pl.(type) {
			case *payload.VCallResult:
			default:
				require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
			}

			return nil
		})

		server.SendMessage(ctx, msg)

		select {
		case <-requestIsDone:
		case <-time.After(10 * time.Second):
			require.Failf(t, "", "timeout")
		}
	}
}

func TestVirtual_Method_WithoutExecutor_Unordered(t *testing.T) {
	t.Log("C4930")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		waitInputChannel  = make(chan struct{}, 2)
		waitOutputChannel = make(chan struct{}, 0)
	)

	mc := minimock.NewController(t)

	executorMock := machine.NewExecutorMock(mc)
	executorMock.CallConstructorMock.Return(gen.UniqueReference().AsBytes(), []byte("345"), nil)
	executorMock.ClassifyMethodMock.Return(contract.MethodIsolation{
		Interference: contract.CallIntolerable,
		State:        contract.CallValidated,
	}, nil)
	executorMock.CallMethodMock.Set(func(
		ctx context.Context, callContext *call.LogicContext, code reference.Global,
		data []byte, method string, args []byte,
	) (
		newObjectState []byte, methodResults []byte, err error,
	) {
		// tell the test that we know about next request
		waitInputChannel <- struct{}{}

		// wait the test result
		<-waitOutputChannel

		return []byte("456"), []byte("345"), nil
	})

	manager := machine.NewManager()
	err := manager.RegisterExecutor(machine.Builtin, executorMock)
	require.NoError(t, err)
	server.ReplaceMachinesManager(manager)

	class := gen.UniqueReference()
	cacheMock := testutils.NewDescriptorsCacheMockWrapper(mc)
	cacheMock.AddClassCodeDescriptor(class, gen.UniqueID(), gen.UniqueReference())
	cacheMock.IntenselyPanic = true
	server.ReplaceCache(cacheMock.Mock())

	objectLocal := gen.UniqueID()
	err = Method_PrepareObject(ctx, server, class, objectLocal)
	require.NoError(t, err)

	{
		requestIsDone := make(chan struct{}, 2)
		server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
			defer func() { requestIsDone <- struct{}{} }()

			pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
			require.NoError(t, err)

			switch pl.(type) {
			case *payload.VCallResult:
			default:
				require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
			}

			return nil
		})

		for i := 0; i < 2; i++ {
			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
				CallAsOf:            0,
				Caller:              server.GlobalCaller(),
				Callee:              reference.NewSelf(objectLocal),
				CallSiteDeclaration: class,
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
			case <-time.After(10 * time.Second):
				require.Failf(t, "", "timeout")
			}
		}

		for i := 0; i < 2; i++ {
			waitOutputChannel <- struct{}{}

			select {
			case <-requestIsDone:
			case <-time.After(10 * time.Second):
				require.Failf(t, "", "timeout")
			}
		}
	}
}

func TestVirtual_Method_WithExecutor(t *testing.T) {
	t.Log("C4923")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	class := testwallet.GetClass()
	objectLocal := server.RandomLocalWithPulse()
	objectGlobal := reference.NewSelf(objectLocal)

	err := Method_PrepareObject(ctx, server, class, objectLocal)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			CallAsOf:            0,
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: class,
			CallSiteMethod:      "GetBalance",
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

		plBytes, err := pl.Marshal()
		require.NoError(t, err)

		msg := payload.MustNewMessage(&payload.Meta{
			Payload:    plBytes,
			Sender:     reference.Global{},
			Receiver:   reference.Global{},
			Pulse:      server.GetPulse().PulseNumber,
			ID:         nil,
			OriginHash: payload.MessageHash{},
		})

		testIsDone := make(chan struct{}, 0)

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

			testIsDone <- struct{}{}

			return nil
		})

		server.SendMessage(ctx, msg)

		<-testIsDone
	}
}

func TestVirtual_CallMethodAfterPulseChange(t *testing.T) {
	t.Log("C4870")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		vCallRequestCount  uint
		vStateRequestCount uint
		vStateReportCount  uint
		vCallResultCount   uint
	)

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
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
	})

	server.IncrementPulse(ctx)

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

	checkBalance(ctx, t, server, objectRef, testBalance)

	expectedNum := uint(1)
	require.Equal(t, expectedNum, vStateReportCount)
	require.Equal(t, expectedNum, vStateRequestCount)
	require.Equal(t, expectedNum, vCallRequestCount)
	require.Equal(t, expectedNum, vCallResultCount)
}
