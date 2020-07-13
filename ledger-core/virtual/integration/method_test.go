// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

const initialBalance uint32 = 500

func Method_PrepareObject(ctx context.Context, server *utils.Server, state payload.VStateReport_StateStatus, object reference.Global) {
	var (
		walletState = makeRawWalletState(initialBalance)

		content *payload.VStateReport_ProvidedContentBody
	)

	switch state {
	case payload.Missing:
		content = nil
	case payload.Ready:
		content = &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: reference.Local{},
				Class:     testwalletProxy.GetClass(),
				State:     walletState,
			},
		}
	default:
		panic("unexpected state")
	}

	payload := &payload.VStateReport{
		Status:          state,
		Object:          object,
		ProvidedContent: content,
	}

	wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	server.SendPayload(ctx, payload)

	select {
	case <-wait:
	case <-time.After(10 * time.Second):
		panic("timeout")
	}
}

func TestVirtual_BadMethod_WithExecutor(t *testing.T) {
	t.Log("C4976")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		outgoing     = server.BuildRandomOutgoingWithPulse()
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	expectedError, err := foundation.MarshalMethodErrorResult(
		throw.W(throw.E("failed to find contracts method"), "failed to classify method"))

	require.NoError(t, err)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		assert.Equal(t, res.Callee, objectGlobal)
		assert.Equal(t, res.CallOutgoing, outgoing)
		assert.Equal(t, expectedError, res.ReturnArguments)

		return false // no resend msg
	})

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: class,
			CallSiteMethod:      "random",
			CallOutgoing:        outgoing,
		}

		server.SendPayload(ctx, &pl)
	}

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Method_WithExecutor(t *testing.T) {
	t.Log("C5088")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool { return false })

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
		Caller:              server.GlobalCaller(),
		Callee:              objectGlobal,
		CallSiteDeclaration: class,
		CallSiteMethod:      "GetBalance",
		CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
	}

	server.SendPayload(ctx, &pl)

	testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Method_WithExecutor_ObjectIsNotExist(t *testing.T) {
	t.Log("C4974")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		outgoing     = server.BuildRandomOutgoingWithPulse()
	)

	Method_PrepareObject(ctx, server, payload.Missing, objectGlobal)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	expectedError := throw.E("object does not exist", struct {
		ObjectReference string
		State           object.State
	}{
		ObjectReference: objectGlobal.String(),
		State:           object.Missing,
	})

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		assert.Equal(t, res.Callee, objectGlobal)
		assert.Equal(t, res.CallOutgoing, outgoing)

		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
		require.NoError(t, sysErr)
		require.Equal(t, expectedError.Error(), contractErr.Error())

		return false // no resend msg
	})

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: testwallet.GetClass(),
			CallSiteMethod:      "GetBalance",
			CallOutgoing:        outgoing,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		server.SendPayload(ctx, &pl)
	}

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Method_WithoutExecutor_Unordered(t *testing.T) {
	t.Log("C5094")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		waitInputChannel  = make(chan struct{}, 2)
		waitOutputChannel = make(chan struct{}, 0)

		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	checkExecution := func(_ execution.Context) {
		// tell the test that we know about next request
		waitInputChannel <- struct{}{}

		// wait the test result
		<-waitOutputChannel
	}

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	{
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, res.ReturnArguments, []byte("345"))
			require.Equal(t, res.Callee, objectGlobal)

			return false // no resend msg
		})

		countBefore := server.PublisherMock.GetCount()

		for i := 0; i < 2; i++ {
			callOutgoing := server.BuildRandomOutgoingWithPulse()

			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
				Caller:              server.GlobalCaller(),
				Callee:              objectGlobal,
				CallSiteDeclaration: class,
				CallSiteMethod:      "GetBalance",
				CallOutgoing:        callOutgoing,
			}

			result := requestresult.New([]byte("345"), objectGlobal)

			key := callOutgoing.String()
			runnerMock.AddExecutionMock(key).
				AddStart(checkExecution, &execution.Update{
					Type:   execution.Done,
					Result: result,
				})
			runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallValidated,
			}, nil)

			server.SendPayload(ctx, &pl)
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
		}

		if !server.PublisherMock.WaitCount(countBefore+2, 10*time.Second) {
			t.Error("failed to wait for result")
		}
	}

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

	{
		assert.Equal(t, 2, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

func TestVirtual_Method_WithoutExecutor_Ordered(t *testing.T) {
	t.Log("C5093")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class        = testwallet.GetClass()
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	cntr := 0
	awaitFullStop := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	{
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, res.ReturnArguments, []byte("345"))
			require.Equal(t, res.Callee, objectGlobal)
			return false // no resend msg
		})
		interferenceFlag := contract.CallTolerable
		stateFlag := contract.CallDirty

		for i := int64(0); i < 2; i++ {
			callOutgoing := server.BuildRandomOutgoingWithPulse()
			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(interferenceFlag, stateFlag),
				Caller:              server.GlobalCaller(),
				Callee:              objectGlobal,
				CallSiteDeclaration: class,
				CallSiteMethod:      "ordered" + strconv.FormatInt(i, 10),
				CallOutgoing:        callOutgoing,
			}

			result := requestresult.New([]byte("345"), objectGlobal)

			key := callOutgoing.String()
			runnerMock.AddExecutionMock(key).
				AddStart(func(ctx execution.Context) {
					cntr ++
					for k := 0; k < 5; k ++ {
						require.Equal(t, 1, cntr)
						time.Sleep(3 * time.Millisecond)
					}
					cntr --
			}, &execution.Update{
				Type:   execution.Done,
				Result: result,
			})
			runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
				Interference: interferenceFlag,
				State:        stateFlag,
			}, nil)

			server.SendPayload(ctx, &pl)
		}
	}
	testutils.WaitSignalsTimed(t, 10*time.Second, awaitFullStop)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_CallMethodAfterPulseChange(t *testing.T) {
	t.Log("C4870")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true)
	typedChecker.VCallResult.SetResend(true)
	typedChecker.VStateReport.SetResend(true)
	typedChecker.VStateRequest.SetResend(true)

	server.IncrementPulseAndWaitIdle(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	// Change pulse to force send VStateReport
	server.IncrementPulseAndWaitIdle(ctx)

	checkBalance(ctx, t, server, objectGlobal, initialBalance)

	{
		assert.Equal(t, 1, typedChecker.VCallRequest.Count())
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
		assert.Equal(t, 1, typedChecker.VStateReport.Count())
	}

	mc.Finish()
}

func TestVirtual_CallMethodAfterMultiplePulseChanges(t *testing.T) {
	t.Log("C4918")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true)
	typedChecker.VCallResult.SetResend(true)
	typedChecker.VStateReport.SetResend(true)
	typedChecker.VStateRequest.SetResend(true)

	server.IncrementPulseAndWaitIdle(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	numPulseChanges := 5
	for i := 0; i < numPulseChanges; i++ {
		wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.IncrementPulseAndWaitIdle(ctx)
		testutils.WaitSignalsTimed(t, 10*time.Second, wait)
	}

	checkBalance(ctx, t, server, objectGlobal, initialBalance)

	{
		assert.Equal(t, 1, typedChecker.VCallRequest.Count())
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
		assert.Equal(t, numPulseChanges, typedChecker.VStateReport.Count())
	}

	mc.Finish()
}

func TestVirtual_CallContractFromContract_InterferenceViolation(t *testing.T) {
	t.Log("C4980")
	table := []struct {
		name         string
		caseId       string
		outgoingCall string
	}{
		{
			name:         "unordered A.Foo sends ordered outgoing and receives error, call method",
			outgoingCall: "method",
		}, {
			name:         "unordered A.Foo sends ordered outgoing and receives error, call constructor",
			outgoingCall: "constructor",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			logger := inslogger.FromContext(ctx)

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)
			server.IncrementPulseAndWaitIdle(ctx)
			var (
				class = gen.UniqueGlobalRef()

				objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())

				flags = contract.MethodIsolation{
					Interference: contract.CallIntolerable,
					State:        contract.CallDirty,
				}
				outgoingCall  execution.RPC
				expectedError []byte
				err           error
			)

			Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

			outgoingCallRef := gen.UniqueGlobalRef()

			switch test.outgoingCall {
			case "method":
				expectedError, err = foundation.MarshalMethodErrorResult(throw.E("interference violation: ordered call from unordered call"))
				require.NoError(t, err)
				outgoingCall = execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectAGlobal, class, "Bar", byteArguments).SetInterference(contract.CallTolerable)
			case "constructor":
				expectedError, err = foundation.MarshalMethodErrorResult(throw.E("interference violation: constructor call from unordered call"))
				require.NoError(t, err)
				outgoingCall = execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallConstructor(class, "Bar", byteArguments)
			default:
				assert.Fail(t, "unexpected outgoingCall type")

			}
			expectedResult := []byte("finish A.Foo")
			objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
			objectAExecutionMock.AddStart(
				func(ctx execution.Context) {
					assert.Equal(t, objectAGlobal, ctx.Object)
					assert.Equal(t, flags, ctx.Isolation)
					logger.Debug("ExecutionStart [A.Foo]")
				},
				&execution.Update{
					Type:     execution.OutgoingCall,
					Error:    nil,
					Outgoing: outgoingCall,
				},
			).AddContinue(func(result []byte) {
				assert.Equal(t, expectedError, result)

			}, &execution.Update{
				Type:   execution.Done,
				Result: requestresult.New(expectedResult, objectAGlobal),
			})

			runnerMock.AddExecutionClassify("Foo", flags, nil)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
				switch res.Callee {
				case objectAGlobal:
					assert.Equal(t, expectedResult, res.ReturnArguments)
				default:
					assert.Fail(t, "unexpected VCallResult")
				}
				return false
			})

			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(flags.Interference, flags.State),
				Caller:              server.GlobalCaller(),
				Callee:              objectAGlobal,
				CallSiteDeclaration: class,
				CallSiteMethod:      "Foo",
				CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
			}

			server.SendPayload(ctx, &pl)
			{
				testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
				testutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
			}

			require.Equal(t, 0, typedChecker.VCallRequest.Count())
			require.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

// A.Foo calls ordered B1.Bar, B2.Bar, B3.Bar
func TestVirtual_CallMultipleContractsFromContract_Ordered(t *testing.T) {
	t.Log("C5114")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	logger := inslogger.FromContext(ctx)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 4)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.Callee.String()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	p := server.GetPulse().PulseNumber

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		flags     = contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		callFlags = payload.BuildCallFlags(flags.Interference, flags.State)

		objectA = reference.NewSelf(server.RandomLocalWithPulse())

		classB1        = gen.UniqueGlobalRef()
		classB2        = gen.UniqueGlobalRef()
		classB3        = gen.UniqueGlobalRef()
		objectB1Global = reference.NewSelf(server.RandomLocalWithPulse())
		objectB2Global = reference.NewSelf(server.RandomLocalWithPulse())
		objectB3Global = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingCallRef = reference.NewRecordOf(
			server.GlobalCaller(), server.RandomLocalWithPulse(),
		)
	)

	// create objects
	{
		Method_PrepareObject(ctx, server, payload.Ready, objectA)
		Method_PrepareObject(ctx, server, payload.Ready, objectB1Global)
		Method_PrepareObject(ctx, server, payload.Ready, objectB2Global)
		Method_PrepareObject(ctx, server, payload.Ready, objectB3Global)
	}

	// add ExecutionMocks to runnerMock
	{
		builder := execution.NewRPCBuilder(outgoingCallRef, objectA)
		objectAExecutionMock := runnerMock.AddExecutionMock(objectA.String())
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [A.Foo]")
				require.Equal(t, server.GlobalCaller(), ctx.Request.Caller)
				require.Equal(t, objectA, ctx.Request.Callee)
				require.Equal(t, outgoingCallRef, ctx.Request.CallOutgoing)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: builder.CallMethod(objectB1Global, classB1, "Bar", []byte("B1")),
			},
		)

		objectAExecutionMock.AddContinue(
			func(result []byte) {
				logger.Debug("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B1.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: builder.CallMethod(objectB2Global, classB2, "Bar", []byte("B2")),
			},
		)
		objectAExecutionMock.AddContinue(
			func(result []byte) {
				logger.Debug("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B2.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: builder.CallMethod(objectB3Global, classB3, "Bar", []byte("B3")),
			},
		)
		objectAExecutionMock.AddContinue(
			func(result []byte) {
				logger.Debug("ExecutionContinue [A.Foo]")
				require.Equal(t, []byte("finish B3.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectA),
			},
		)

		runnerMock.AddExecutionMock(objectB1Global.String()).AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [B1.Bar]")
				require.Equal(t, objectB1Global, ctx.Request.Callee)
				require.Equal(t, objectA, ctx.Request.Caller)
				require.Equal(t, []byte("B1"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish B1.Bar"), objectB1Global),
			},
		)

		runnerMock.AddExecutionMock(objectB2Global.String()).AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [B2.Bar]")
				require.Equal(t, objectB2Global, ctx.Request.Callee)
				require.Equal(t, objectA, ctx.Request.Caller)
				require.Equal(t, []byte("B2"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish B2.Bar"), objectB2Global),
			},
		)

		runnerMock.AddExecutionMock(objectB3Global.String()).AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [B3.Bar]")
				require.Equal(t, objectB3Global, ctx.Request.Callee)
				require.Equal(t, objectA, ctx.Request.Caller)
				require.Equal(t, []byte("B3"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish B3.Bar"), objectB3Global),
			},
		)

		runnerMock.AddExecutionClassify(objectA.String(), flags, nil)
		runnerMock.AddExecutionClassify(objectB1Global.String(), flags, nil)
		runnerMock.AddExecutionClassify(objectB2Global.String(), flags, nil)
		runnerMock.AddExecutionClassify(objectB3Global.String(), flags, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, objectA, request.Caller)
			assert.Equal(t, payload.CTMethod, request.CallType)
			assert.Equal(t, outgoingCallRef, request.CallReason)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, p, request.CallOutgoing.GetLocal().Pulse())

			switch request.Callee {
			case objectB1Global:
				require.Equal(t, []byte("B1"), request.Arguments)
				require.Equal(t, uint32(1), request.CallSequence)
			case objectB2Global:
				require.Equal(t, []byte("B2"), request.Arguments)
				require.Equal(t, uint32(2), request.CallSequence)
			case objectB3Global:
				require.Equal(t, []byte("B3"), request.Arguments)
				require.Equal(t, uint32(3), request.CallSequence)
			default:
				t.Fatal("wrong Callee")
			}
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, payload.CTMethod, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee {
			case objectA:
				require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
				require.Equal(t, server.GlobalCaller(), res.Caller)
				require.Equal(t, outgoingCallRef, res.CallOutgoing)
			case objectB1Global:
				require.Equal(t, []byte("finish B1.Bar"), res.ReturnArguments)
				require.Equal(t, objectA, res.Caller)
				require.Equal(t, p, res.CallOutgoing.GetLocal().Pulse())
			case objectB2Global:
				require.Equal(t, []byte("finish B2.Bar"), res.ReturnArguments)
				require.Equal(t, objectA, res.Caller)
				require.Equal(t, p, res.CallOutgoing.GetLocal().Pulse())
			case objectB3Global:
				require.Equal(t, []byte("finish B3.Bar"), res.ReturnArguments)
				require.Equal(t, objectA, res.Caller)
				require.Equal(t, p, res.CallOutgoing.GetLocal().Pulse())
			default:
				t.Fatal("wrong Callee")
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == objectA
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      callFlags,
		Caller:         server.GlobalCaller(),
		Callee:         objectA,
		CallSiteMethod: "Foo",
		CallOutgoing:   outgoingCallRef,
	}
	server.SendPayload(ctx, &pl)

	// wait for all calls and SMs
	testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 3, typedChecker.VCallRequest.Count())
	require.Equal(t, 4, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Method_Have_ObjectState(t *testing.T) {
	type runnerObjectChecker func(objectState *payload.VStateReport_ProvidedContentBody, runnerObjectState descriptor.Object) bool
	table := []struct {
		name string
		code string
		skip string

		state  contract.StateFlag
		checks []runnerObjectChecker
	}{
		{
			name:  "Method with CallFlags.Dirty must be called with dirty object state",
			code:  "C5184",
			skip:  "",
			state: contract.CallDirty,
		},
		{
			name:  "Method with CallFlags.Validated must be called with validated object state",
			code:  "C5123",
			skip:  "https://insolar.atlassian.net/browse/PLAT-404",
			state: contract.CallValidated,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			t.Log(test.code)
			if len(test.skip) > 0 {
				t.Skip(test.skip)
			}

			var (
				mc = minimock.NewController(t)
			)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			runnerMock := logicless.NewServiceMock(ctx, t, nil)
			server.ReplaceRunner(runnerMock)

			server.Init(ctx)
			server.IncrementPulse(ctx)

			var (
				class             = gen.UniqueGlobalRef()
				objectRef         = server.BuildRandomOutgoingWithPulse()
				outgoingRef       = server.BuildRandomOutgoingWithPulse()
				dirtyStateRef     = server.RandomLocalWithPulse()
				dirtyState        = reference.NewSelf(dirtyStateRef)
				validatedStateRef = server.RandomLocalWithPulse()
				validatedState    = reference.NewSelf(validatedStateRef)
			)
			const (
				validatedMem = "12345"
				dirtyMem     = "54321"
			)

			{ // send object state to server
				pl := payload.VStateReport{
					Status:               payload.Ready,
					Object:               objectRef,
					LatestValidatedState: validatedState,
					LatestDirtyState:     dirtyState,
					ProvidedContent: &payload.VStateReport_ProvidedContentBody{
						LatestValidatedState: &payload.ObjectState{
							Reference: validatedStateRef,
							Class:     class,
							State:     []byte(validatedMem),
						},
						LatestDirtyState: &payload.ObjectState{
							Reference: dirtyStateRef,
							Class:     class,
							State:     []byte(dirtyMem),
						},
					},
					UnorderedPendingEarliestPulse: pulse.OfNow(),
				}

				server.WaitIdleConveyor()
				server.SendPayload(ctx, &pl)
				server.WaitActiveThenIdleConveyor()
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			{
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					require.Equal(t, []byte("345"), res.ReturnArguments)
					require.Equal(t, objectRef, res.Callee)

					return false // no resend msg
				})

				pl := payload.VCallRequest{
					CallType:            payload.CTMethod,
					CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, test.state),
					Caller:              server.GlobalCaller(),
					Callee:              objectRef,
					CallSiteDeclaration: class,
					CallSiteMethod:      "Test",
					CallOutgoing:        outgoingRef,
				}

				key := pl.CallOutgoing.String()
				runnerMock.AddExecutionMock(key).
					AddStart(func(ctx execution.Context) {
						require.Equal(t, objectRef, ctx.Object)
						require.Equal(t, test.state, ctx.Request.CallFlags.GetState())
						require.Equal(t, test.state, ctx.Isolation.State)
						require.Equal(t, objectRef, ctx.ObjectDescriptor.HeadRef())
						stateClass, err := ctx.ObjectDescriptor.Class()
						require.NoError(t, err)
						require.Equal(t, class, stateClass)

						if test.state == contract.CallValidated {
							require.Equal(t, validatedStateRef, ctx.ObjectDescriptor.StateID())
							require.Equal(t, []byte(validatedMem), ctx.ObjectDescriptor.Memory())
						} else {
							require.Equal(t, dirtyStateRef, ctx.ObjectDescriptor.StateID())
							require.Equal(t, []byte(dirtyMem), ctx.ObjectDescriptor.Memory())
						}
					}, &execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("345"), outgoingRef),
					})
				runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
					Interference: contract.CallIntolerable,
					State:        test.state,
				}, nil)

				server.SendPayload(ctx, &pl)
			}

			testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

// twice ( A.Foo -> B.Bar, B.Bar )
func TestVirtual_CallContractTwoTimes(t *testing.T) {
	t.Log("C5183")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallOutgoing.String()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		flags     = contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		callFlags = payload.BuildCallFlags(flags.Interference, flags.State)

		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		classB        = gen.UniqueGlobalRef()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		outgoingFirstCall  = server.BuildRandomOutgoingWithPulse()
		outgoingSecondCall = server.BuildRandomOutgoingWithPulse()
	)

	// create objects
	{
		Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
		Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)
	}

	// add ExecutionMocks to runnerMock
	{
		firstBuilder := execution.NewRPCBuilder(outgoingFirstCall, objectAGlobal)
		objectAExecutionFirstMock := runnerMock.AddExecutionMock(outgoingFirstCall.String())
		objectAExecutionFirstMock.AddStart(nil,
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: firstBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("first")),
			},
		)
		objectAExecutionFirstMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: firstBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("second")),
			},
		)
		objectAExecutionFirstMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		secondBuilder := execution.NewRPCBuilder(outgoingSecondCall, objectAGlobal)
		objectAExecutionSecondMock := runnerMock.AddExecutionMock(outgoingSecondCall.String())
		objectAExecutionSecondMock.AddStart(nil,
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: secondBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("first")),
			},
		)
		objectAExecutionSecondMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: secondBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("second")),
			},
		)
		objectAExecutionSecondMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		runnerMock.AddExecutionClassify(outgoingFirstCall.String(), flags, nil)
		runnerMock.AddExecutionClassify(outgoingSecondCall.String(), flags, nil)
	}

	// add publish checker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			switch string(request.Arguments[0]) {
			case "f":
				require.Equal(t, []byte("first"), request.Arguments)
				require.Equal(t, uint32(1), request.CallSequence)
			case "s":
				require.Equal(t, []byte("second"), request.Arguments)
				require.Equal(t, uint32(2), request.CallSequence)
			default:
				t.Fatal("wrong call args")
			}

			result := payload.VCallResult{
				CallType:        request.CallType,
				CallFlags:       request.CallFlags,
				Caller:          request.Caller,
				Callee:          request.Callee,
				CallOutgoing:    request.CallOutgoing,
				ReturnArguments: []byte("finish B.Bar"),
			}
			msg := server.WrapPayload(&result).Finalize()
			server.SendMessage(ctx, msg)

			return false
		})
		typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
			require.Equal(t, []byte("finish A.Foo"), result.ReturnArguments)
			return false
		})
	}

	// send first VCallRequest A.Foo
	{
		pl := payload.VCallRequest{
			CallType:       payload.CTMethod,
			CallFlags:      callFlags,
			Caller:         server.GlobalCaller(),
			Callee:         objectAGlobal,
			CallSiteMethod: "Foo",
			CallOutgoing:   outgoingFirstCall,
			Arguments:      []byte("call foo"),
		}
		msg := server.WrapPayload(&pl).Finalize()
		server.SendMessage(ctx, msg)
		server.WaitActiveThenIdleConveyor()
	}
	// send second VCallRequest A.Foo
	{
		pl := payload.VCallRequest{
			CallType:       payload.CTMethod,
			CallFlags:      callFlags,
			Caller:         server.GlobalCaller(),
			Callee:         objectAGlobal,
			CallSiteMethod: "Foo",
			CallOutgoing:   outgoingSecondCall,
			Arguments:      []byte("call foo"),
		}
		msg := server.WrapPayload(&pl).Finalize()
		server.SendMessage(ctx, msg)
	}

	// wait for all calls and SMs
	{
		testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
		testutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	require.Equal(t, 4, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}

func Test_CallMethodWithBadIsolationFlags(t *testing.T) {
	t.Log("C4979")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		outgoing     = server.BuildRandomOutgoingWithPulse()
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	expectedError := throw.W(throw.IllegalValue(), "failed to negotiate call isolation params", struct {
		methodIsolation contract.MethodIsolation
		callIsolation   contract.MethodIsolation
	}{
		methodIsolation: contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		},
		callIsolation: contract.MethodIsolation{
			Interference: contract.CallIntolerable,
			State:        contract.CallValidated,
		},
	})

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		assert.Equal(t, res.Callee, objectGlobal)
		assert.Equal(t, res.CallOutgoing, outgoing)

		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
		require.NoError(t, sysErr)
		require.NotNil(t, contractErr)
		require.Equal(t, expectedError.Error(), contractErr.Error())

		return false // no resend msg
	})

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			Caller:              server.GlobalCaller(),
			Callee:              objectGlobal,
			CallSiteDeclaration: testwallet.GetClass(),
			CallSiteMethod:      "Accept",
			CallOutgoing:        outgoing,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		server.SendPayload(ctx, &pl)
	}

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_FutureMessageAddedToSlot(t *testing.T) {
	t.Log("C5318")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	jetCoordinatorMock := jet.NewAffinityHelperMock(mc)
	auth := authentication.NewService(ctx, jetCoordinatorMock)
	server.ReplaceAuthenticationService(auth)

	server.Init(ctx)
	server.IncrementPulse(ctx)

	// legitimate sender
	jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{server.GlobalCaller()}, nil)
	jetCoordinatorMock.MeMock.Return(server.GlobalCaller())

	var (
		objectLocal       = server.RandomLocalWithPulse()
		objectGlobal      = reference.NewSelf(objectLocal)
		outgoing          = server.BuildRandomOutgoingWithPulse()
		class             = gen.UniqueGlobalRef()
		dirtyStateRef     = server.RandomLocalWithPulse()
		dirtyState        = reference.NewSelf(dirtyStateRef)
		validatedStateRef = server.RandomLocalWithPulse()
		validatedState    = reference.NewSelf(validatedStateRef)
	)

	const (
		validatedMem = "12345"
		dirtyMem     = "54321"
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool { return false })
	typedChecker.VStateReport.Set(func(res *payload.VStateReport) bool { return false })
	typedChecker.VStateRequest.Set(func(res *payload.VStateRequest) bool {
		report := &payload.VStateReport{
			Status:               payload.Ready,
			AsOf:                 0,
			Object:               objectGlobal,
			LatestValidatedState: validatedState,
			LatestDirtyState:     dirtyState,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestValidatedState: &payload.ObjectState{
					Reference: validatedStateRef,
					Class:     class,
					State:     []byte(validatedMem),
				},
				LatestDirtyState: &payload.ObjectState{
					Reference: dirtyStateRef,
					Class:     class,
					State:     []byte(dirtyMem),
				},
			},
		}
		server.SendPayload(ctx, report)
		return false
	})

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
		Caller:              server.GlobalCaller(),
		Callee:              objectGlobal,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "Accept",
		CallOutgoing:        outgoing,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	// now we are not legitimate sender
	jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{server.RandomGlobalWithPulse()}, nil)
	jetCoordinatorMock.MeMock.Return(server.RandomGlobalWithPulse())

	server.WaitIdleConveyor()
	server.SendPayloadAsFuture(ctx, &pl)
	// new request goes to future pulse slot
	server.WaitActiveThenIdleConveyor()

	// legitimate sender
	jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{server.GlobalCaller()}, nil)
	jetCoordinatorMock.MeMock.Return(server.GlobalCaller())

	// switch pulse and start processing request from future slot
	server.IncrementPulseAndWaitIdle(ctx)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&execute.SMExecute{}, 1))
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	mc.Finish()
}

func Test_MethodCall_HappyPath(t *testing.T) {
	t.Log("C5089")
	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	logger := inslogger.FromContext(ctx)
	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	{
		server.ReplaceRunner(runnerMock)
		server.Init(ctx)
		server.IncrementPulseAndWaitIdle(ctx)
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		class  = gen.UniqueGlobalRef()
		object = reference.NewSelf(server.RandomLocalWithPulse())
		p1     = server.GetPulse().PulseNumber
	)

	server.IncrementPulseAndWaitIdle(ctx)
	outgoingP2 := server.BuildRandomOutgoingWithPulse()

	// add ExecutionMock to runnerMock
	{
		runnerMock.AddExecutionClassify("SomeMethod", intolerableFlags(), nil)
		requestResult := requestresult.New([]byte("call result"), gen.UniqueGlobalRef())

		objectExecutionMock := runnerMock.AddExecutionMock("SomeMethod")
		objectExecutionMock.AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [SomeMethod]")
				require.Equal(t, object, ctx.Request.Callee)
				require.Equal(t, []byte("new object memory"), ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
	}

	typedChecker.VStateRequest.Set(func(req *payload.VStateRequest) bool {
		require.Equal(t, p1, req.AsOf)
		require.Equal(t, object, req.Object)

		flags := payload.StateRequestContentFlags(0)
		flags.Set(
			payload.RequestLatestDirtyState,
			payload.RequestLatestValidatedState,
			payload.RequestOrderedQueue,
			payload.RequestUnorderedQueue,
		)
		require.Equal(t, flags, req.RequestedContent)

		content := &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: reference.Local{},
				Class:     testwalletProxy.GetClass(),
				State:     []byte("new object memory"),
			},
		}

		report := payload.VStateReport{
			Status:          payload.Ready,
			AsOf:            req.AsOf,
			Object:          object,
			ProvidedContent: content,
		}
		server.SendPayload(ctx, &report)
		return false // no resend msg
	})

	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, []byte("call result"), res.ReturnArguments)
		require.Equal(t, object, res.Callee)
		require.Equal(t, outgoingP2, res.CallOutgoing)
		return false // no resend msg
	})

	// VCallRequest
	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated),
			Caller:              server.GlobalCaller(),
			Callee:              object,
			CallSiteDeclaration: class,
			CallSiteMethod:      "SomeMethod",
			CallOutgoing:        outgoingP2,
		}
		server.SendPayload(ctx, &pl)
	}

	testutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VStateRequest.Count())
	require.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Method_ForObjectWithMissingState(t *testing.T) {
	testCases := []struct {
		name             string
		testRailID       string
		outgoingFromPast bool
	}{
		{
			name:             "Call method with prev outgoing.Pulse",
			testRailID:       "C5106",
			outgoingFromPast: true,
		},
		{
			name:             "Call method with current outgoing.Pulse",
			testRailID:       "C5321",
			outgoingFromPast: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Log(testCase.testRailID)
			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()
			server.IncrementPulseAndWaitIdle(ctx)

			execDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
			stateHandled := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)

			var (
				objectRef = server.RandomGlobalWithPulse()
				outgoing  = server.BuildRandomOutgoingWithPulse()
			)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
				require.Equal(t, payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty), result.CallFlags)
				require.Equal(t, objectRef, result.Callee)
				require.Equal(t, outgoing, result.CallOutgoing)
				require.Equal(t, server.GlobalCaller(), result.Caller)
				require.True(t, result.DelegationSpec.IsZero())
				contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments)
				require.NoError(t, sysErr)
				require.Contains(t, contractErr.Error(), "object does not exist")
				return false
			})

			if testCase.outgoingFromPast {
				server.IncrementPulseAndWaitIdle(ctx)
			}

			state := &payload.VStateReport{
				Status: payload.Missing,
				Object: objectRef,
			}

			server.SendPayload(ctx, state)
			testutils.WaitSignalsTimed(t, 10*time.Second, stateHandled)

			pl := payload.VCallRequest{
				CallType:       payload.CTMethod,
				CallFlags:      payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
				Caller:         server.GlobalCaller(),
				Callee:         objectRef,
				CallSiteMethod: "Method",
				CallOutgoing:   outgoing,
			}

			server.SendPayload(ctx, &pl)

			testutils.WaitSignalsTimed(t, 10*time.Second, execDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallResult.Count())
			typedChecker.MinimockWait(10 * time.Second)

			mc.Finish()
		})
	}
}
