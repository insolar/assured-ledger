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
	"github.com/stretchr/testify/require"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func Method_PrepareObject(ctx context.Context, server *utils.Server, class reference.Global, object reference.Local) error {
	isolation := contract.ConstructorIsolation()

	pl := payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
		CallAsOf:            0,
		Caller:              server.GlobalCaller(),
		Callee:              class,
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
	msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

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

	server.WaitActiveThenIdleConveyor()

	select {
	case err := <-requestIsDone:
		return err
	case <-time.After(10 * time.Second):
		return errors.New("timeout")
	}
}

func TestVirtual_BadMethod_WithExecutor(t *testing.T) {
	t.Log("C4976")
	t.Skip("https://insolar.atlassian.net/browse/PLAT-397")

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
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: class,
			CallSiteMethod:      "random",
			CallOutgoing:        server.RandomLocalWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).Finalize()

		server.SendMessage(ctx, msg)

		// TODO fix it after implementation https://insolar.atlassian.net/browse/PLAT-397
	}
}

func TestVirtual_Method_WithExecutor(t *testing.T) {
	t.Log("C4923")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	//server.IncrementPulse(ctx)

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
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

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

func TestVirtual_Method_WithExecutor_ObjectIsNotExist(t *testing.T) {
	t.Log("C4974")
	// t.Skip("https://insolar.atlassian.net/browse/PLAT-395")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	objectRef := reference.NewSelf(server.RandomLocalWithPulse())

	msg := makeVStateReportWithState(server.GetPulse().PulseNumber, objectRef, payload.Missing, nil, server.JetCoordinatorMock.Me())
	server.SendMessage(ctx, msg)

	server.WaitActiveThenIdleConveyor()

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			Caller:              server.GlobalCaller(),
			Callee:              objectRef,
			CallSiteDeclaration: testwallet.GetClass(),
			CallSiteMethod:      "GetBalance",
			CallOutgoing:        server.RandomLocalWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

		server.SendMessage(ctx, msg)

		// TODO fix it after implementation https://insolar.atlassian.net/browse/PLAT-395
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

	objectLocal := server.RandomLocalWithPulse()
	objectRef := reference.NewSelf(objectLocal)

	err = Method_PrepareObject(ctx, server, class, objectLocal)
	require.NoError(t, err)

	{
		requestIsDone := make(chan struct{}, 2)
		typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			defer func() { requestIsDone <- struct{}{} }()

			require.Equal(t, res.ReturnArguments, []byte("345"))
			require.Equal(t, res.Callee, objectRef)

			return false // no resend msg
		})

		for i := 0; i < 2; i++ {
			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
				CallAsOf:            0,
				Caller:              server.GlobalCaller(),
				Callee:              objectRef,
				CallSiteDeclaration: class,
				CallSiteMethod:      "GetBalance",
				CallRequestFlags:    0,
				CallOutgoing:        server.RandomLocalWithPulse(),
				Arguments:           insolar.MustSerialize([]interface{}{}),
			}
			msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(server.JetCoordinatorMock.Me()).Finalize()

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

	mc.Finish()
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
			vStateReportCount++
		case *payload.VStateRequest:
			vStateRequestCount++
		default:
			require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
		}

		server.SendMessage(ctx, messages[0])
		return nil
	})

	server.IncrementPulseAndWaitIdle(ctx)

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := reference.NewRecordRef(server.RandomLocalWithPulse())
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(server.GetPulse().PulseNumber, objectRef, stateID, rawWalletState, server.JetCoordinatorMock.Me())
		server.SendMessage(ctx, msg)
	}

	server.WaitActiveThenIdleConveyor()

	// Change pulse to force send VStateRequest
	server.IncrementPulseAndWaitIdle(ctx)

	checkBalance(ctx, t, server, objectRef, testBalance)

	expectedNum := uint(1)
	require.Equal(t, expectedNum, vStateReportCount)
	require.Equal(t, uint(0), vStateRequestCount)
	require.Equal(t, expectedNum, vCallRequestCount)
	require.Equal(t, expectedNum, vCallResultCount)
}
