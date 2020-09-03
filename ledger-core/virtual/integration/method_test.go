// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"

	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

const initialBalance uint32 = 500

func Method_PrepareObject(
	ctx context.Context,
	server *utils.Server,
	state payload.VStateReport_StateStatus,
	object reference.Global,
	pulse payload.PulseNumber,
) {
	var (
		walletState = makeRawWalletState(initialBalance)

		content *payload.VStateReport_ProvidedContentBody
	)

	switch state {
	case payload.StateStatusMissing:
		content = nil
	case payload.StateStatusReady:
		content = &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: reference.Local{},
				Class:     testwalletProxy.GetClass(),
				State:     walletState,
			},
			LatestValidatedState: &payload.ObjectState{
				Reference: reference.Local{},
				Class:     testwalletProxy.GetClass(),
				State:     walletState,
			},
		}
	case payload.StateStatusInactive:
		content = nil
	default:
		panic("unexpected state")
	}

	vsrPayload := &payload.VStateReport{
		Status:          state,
		Object:          object,
		AsOf:            pulse,
		ProvidedContent: content,
	}

	wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	server.SendPayload(ctx, vsrPayload)

	select {
	case <-wait:
	case <-time.After(10 * time.Second):
		panic("timeout")
	}
}

func TestVirtual_BadMethod_WithExecutor(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4976")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulseAndWaitIdle(ctx)

	utils.AssertNotJumpToStep(t, server.Journal, "stepTakeLock")

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		outgoing     = server.BuildRandomOutgoingWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	// need for correct handle state report (should from prev pulse)
	server.IncrementPulse(ctx)

	Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)

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
		pl := utils.GenerateVCallRequestMethodImmutable(server)
		pl.Callee = objectGlobal
		pl.CallOutgoing = outgoing

		server.SendPayload(ctx, pl)
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	mc.Finish()
}

func TestVirtual_Method_WithExecutor_ObjectIsNotExist(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4974")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		outgoing     = server.BuildRandomOutgoingWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	// need for correct handle state report (should from prev pulse)
	server.IncrementPulse(ctx)

	Method_PrepareObject(ctx, server, payload.StateStatusMissing, objectGlobal, prevPulse)

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
		pl := utils.GenerateVCallRequestMethodImmutable(server)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "GetBalance"
		pl.CallOutgoing = outgoing

		server.SendPayload(ctx, pl)
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Method_WithoutExecutor_Unordered(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5094")

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

		objectGlobal = server.RandomGlobalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	// need for correct handle state report (should from prev pulse)
	server.IncrementPulse(ctx)

	checkExecution := func(_ execution.Context) {
		// tell the test that we know about next request
		waitInputChannel <- struct{}{}

		// wait the test result
		<-waitOutputChannel
	}

	Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	{
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, res.ReturnArguments, []byte("345"))
			require.Equal(t, res.Callee, objectGlobal)

			return false // no resend msg
		})

		countBefore := server.PublisherMock.GetCount()

		for i := 0; i < 2; i++ {
			pl := utils.GenerateVCallRequestMethodImmutable(server)
			pl.Callee = objectGlobal
			pl.CallSiteMethod = "GetBalance"

			result := requestresult.New([]byte("345"), objectGlobal)

			key := pl.CallOutgoing.String()
			runnerMock.AddExecutionMock(key).
				AddStart(checkExecution, &execution.Update{
					Type:   execution.Done,
					Result: result,
				})
			runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
				Interference: pl.GetCallFlags().GetInterference(),
				State:        pl.GetCallFlags().GetState(),
			}, nil)

			server.SendPayload(ctx, pl)
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

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

	{
		assert.Equal(t, 2, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

func TestVirtual_Method_WithoutExecutor_Ordered(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5093")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	// need for correct handle state report (should from prev pulse)
	server.IncrementPulse(ctx)

	Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	counter := 0
	awaitFullStop := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	{
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, res.ReturnArguments, []byte("345"))
			require.Equal(t, res.Callee, objectGlobal)
			return false // no resend msg
		})

		for i := int64(0); i < 2; i++ {
			pl := utils.GenerateVCallRequestMethod(server)
			pl.Callee = objectGlobal
			pl.CallSiteMethod = "ordered" + strconv.FormatInt(i, 10)
			callOutgoing := pl.CallOutgoing

			result := requestresult.New([]byte("345"), objectGlobal)

			key := callOutgoing.String()
			runnerMock.AddExecutionMock(key).
				AddStart(func(ctx execution.Context) {
					counter++
					for k := 0; k < 5; k++ {
						require.Equal(t, 1, counter)
						time.Sleep(3 * time.Millisecond)
					}
					counter--
				}, &execution.Update{
					Type:   execution.Done,
					Result: result,
				})
			runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
				Interference: pl.CallFlags.GetInterference(),
				State:        pl.CallFlags.GetState(),
			}, nil)

			server.SendPayload(ctx, pl)
		}
	}
	commontestutils.WaitSignalsTimed(t, 10*time.Second, awaitFullStop)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_CallContractFromContract_InterferenceViolation(t *testing.T) {
	insrail.LogCase(t, "C4980")

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
			defer commontestutils.LeakTester(t)

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
				class = server.RandomGlobalWithPulse()

				prevPulse     = server.GetPulse().PulseNumber
				objectAGlobal = server.RandomGlobalWithPulse()

				flags = contract.MethodIsolation{
					Interference: isolation.CallIntolerable,
					State:        isolation.CallDirty,
				}
				outgoingCall  execution.RPC
				expectedError []byte
				err           error
			)

			server.IncrementPulseAndWaitIdle(ctx)

			Method_PrepareObject(ctx, server, payload.StateStatusReady, objectAGlobal, prevPulse)

			outgoingCallRef := server.RandomGlobalWithPulse()

			switch test.outgoingCall {
			case "method":
				expectedError, err = foundation.MarshalMethodErrorResult(throw.E("interference violation: ordered call from unordered call"))
				require.NoError(t, err)
				outgoingCall = execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectAGlobal, class, "Bar", byteArguments).SetInterference(isolation.CallTolerable)
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

			pl := utils.GenerateVCallRequestMethod(server)
			pl.CallFlags = payload.BuildCallFlags(flags.Interference, flags.State)
			pl.Callee = objectAGlobal
			pl.CallSiteMethod = "Foo"

			server.SendPayload(ctx, pl)
			{
				commontestutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
				commontestutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
			}

			require.Equal(t, 0, typedChecker.VCallRequest.Count())
			require.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

// A.Foo calls ordered B1.Bar, B2.Bar, B3.Bar
func TestVirtual_CallMultipleContractsFromContract_Ordered(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5114")

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

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		objectA        = server.RandomGlobalWithPulse()
		objectB1Global = server.RandomGlobalWithPulse()
		objectB2Global = server.RandomGlobalWithPulse()
		objectB3Global = server.RandomGlobalWithPulse()
		prevPulse      = server.GetPulse().PulseNumber
	)

	server.IncrementPulseAndWaitIdle(ctx)

	// create objects
	{
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectA, prevPulse)
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectB1Global, prevPulse)
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectB2Global, prevPulse)
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectB3Global, prevPulse)
	}

	p := server.GetPulse().PulseNumber

	var (
		flags     = contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty}
		callFlags = payload.BuildCallFlags(flags.Interference, flags.State)

		classB1 = server.RandomGlobalWithPulse()
		classB2 = server.RandomGlobalWithPulse()
		classB3 = server.RandomGlobalWithPulse()

		outgoingCallRef = server.BuildRandomOutgoingWithPulse()
	)

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
			assert.Equal(t, payload.CallTypeMethod, request.CallType)
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
			assert.Equal(t, payload.CallTypeMethod, res.CallType)
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

	pl := utils.GenerateVCallRequestMethod(server)
	pl.CallFlags = callFlags
	pl.Callee = objectA
	pl.CallOutgoing = outgoingCallRef

	server.SendPayload(ctx, pl)

	// wait for all calls and SMs
	commontestutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 3, typedChecker.VCallRequest.Count())
	require.Equal(t, 4, typedChecker.VCallResult.Count())

	mc.Finish()
}

// twice ( A.Foo -> B.Bar, B.Bar )
func TestVirtual_CallContractTwoTimes(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5183")

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
		flags     = contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty}
		callFlags = payload.BuildCallFlags(flags.Interference, flags.State)

		classB = server.RandomGlobalWithPulse()

		objectAGlobal = server.RandomGlobalWithPulse()
		objectBGlobal = server.RandomGlobalWithPulse()

		prevPulse = server.GetPulse().PulseNumber
	)

	server.IncrementPulseAndWaitIdle(ctx)

	// create objects
	{
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectAGlobal, prevPulse)
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectBGlobal, prevPulse)
	}

	var (
		outgoingFirstCall  = server.BuildRandomOutgoingWithPulse()
		outgoingSecondCall = server.BuildRandomOutgoingWithPulse()
	)

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
				CallIncoming:    server.RandomGlobalWithPulse(),
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

	{ // send first VCallRequest A.Foo
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = callFlags
		pl.Callee = objectAGlobal
		pl.CallSiteMethod = "Foo"
		pl.CallOutgoing = outgoingFirstCall
		pl.Arguments = []byte("call foo")

		server.SendPayload(ctx, pl)
	}

	server.WaitActiveThenIdleConveyor()

	{ // send second VCallRequest A.Foo
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = callFlags
		pl.Callee = objectAGlobal
		pl.CallSiteMethod = "Foo"
		pl.CallOutgoing = outgoingSecondCall
		pl.Arguments = []byte("call foo")

		server.SendPayload(ctx, pl)
	}

	// wait for all calls and SMs
	{
		commontestutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
		commontestutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	require.Equal(t, 4, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}

func Test_CallMethodWithBadIsolationFlags(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4979")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	utils.AssertNotJumpToStep(t, server.Journal, "stepTakeLock")

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	server.IncrementPulseAndWaitIdle(ctx)

	Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	expectedError := throw.W(throw.IllegalValue(), "failed to negotiate call isolation params", struct {
		methodIsolation contract.MethodIsolation
		callIsolation   contract.MethodIsolation
	}{
		methodIsolation: contract.MethodIsolation{
			Interference: isolation.CallTolerable,
			State:        isolation.CallDirty,
		},
		callIsolation: contract.MethodIsolation{
			Interference: isolation.CallIntolerable,
			State:        isolation.CallValidated,
		},
	})

	outgoing := server.BuildRandomOutgoingWithPulse()

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
		pl := utils.GenerateVCallRequestMethodImmutable(server)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "Accept"
		pl.CallOutgoing = outgoing

		server.SendPayload(ctx, pl)
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_FutureMessageAddedToSlot(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5318")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	jetCoordinatorMock := affinity.NewHelperMock(mc)
	auth := authentication.NewService(ctx, jetCoordinatorMock)
	server.ReplaceAuthenticationService(auth)

	server.Init(ctx)
	server.IncrementPulse(ctx)

	// legitimate sender
	jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{server.GlobalCaller()}, nil)
	jetCoordinatorMock.MeMock.Return(server.GlobalCaller())

	var (
		objectGlobal      = server.RandomGlobalWithPulse()
		class             = server.RandomGlobalWithPulse()
		dirtyStateRef     = server.RandomLocalWithPulse()
		dirtyState        = reference.NewSelf(dirtyStateRef)
		validatedStateRef = server.RandomLocalWithPulse()
		validatedState    = reference.NewSelf(validatedStateRef)
		prevPulse         = server.GetPulse().PulseNumber
	)

	const (
		validatedMem = "12345"
		dirtyMem     = "54321"
	)

	server.IncrementPulseAndWaitIdle(ctx)

	Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)

	p := server.GetPulse().PulseNumber

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool { return false })
	typedChecker.VStateReport.Set(func(res *payload.VStateReport) bool { return false })
	typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
		t.FailNow()
		// TODO add asserts and check counter after https://insolar.atlassian.net/browse/PLAT-753
		return false
	})
	typedChecker.VStateRequest.Set(func(res *payload.VStateRequest) bool {
		report := &payload.VStateReport{
			Status:               payload.StateStatusReady,
			AsOf:                 p,
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

	pl := utils.GenerateVCallRequestMethodImmutable(server)
	pl.Callee = objectGlobal
	pl.CallSiteMethod = "Accept"

	// now we are not legitimate sender
	jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{server.RandomGlobalWithPulse()}, nil)
	jetCoordinatorMock.MeMock.Return(server.RandomGlobalWithPulse())

	server.WaitIdleConveyor()
	server.SendPayloadAsFuture(ctx, pl)
	// new request goes to future pulse slot
	server.WaitActiveThenIdleConveyor()

	// legitimate sender
	jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{server.GlobalCaller()}, nil)
	jetCoordinatorMock.MeMock.Return(server.GlobalCaller())

	// switch pulse and start processing request from future slot
	execDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	server.IncrementPulseAndWaitIdle(ctx)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, execDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	mc.Finish()
}

func Test_MethodCall_HappyPath(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5089")

	const (
		origDirtyObjectMem     = "dirty original object memory"
		origValidatedObjectMem = "validated original object memory"
		changedObjectMem       = "new object memory"
		callResult             = "call result"
	)
	cases := []struct {
		name           string
		isolation      contract.MethodIsolation
		canChangeState bool
	}{
		{
			name:           "Tolerable call can change object state",
			canChangeState: true,
			isolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallDirty,
			},
		},
		{
			name:           "Intolerable call cannot change object state",
			canChangeState: false,
			isolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallValidated,
			},
		},
		{
			name:           "Intolerable call on Dirty state cannot change object state",
			canChangeState: false,
			isolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallDirty,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})
			{
				server.ReplaceRunner(runnerMock)
				server.Init(ctx)
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			var (
				class     = server.RandomGlobalWithPulse()
				objectRef = server.RandomGlobalWithPulse()
				p1        = server.GetPulse().PulseNumber
			)

			server.IncrementPulseAndWaitIdle(ctx)
			outgoing := server.BuildRandomOutgoingWithPulse()

			// add ExecutionMock to runnerMock
			{
				runnerMock.AddExecutionClassify("SomeMethod", testCase.isolation, nil)
				requestResult := requestresult.New([]byte(callResult), objectRef)
				if testCase.canChangeState {
					newObjDescriptor := descriptor.NewObject(
						reference.Global{}, reference.Local{}, class, []byte(""), false,
					)
					requestResult.SetAmend(newObjDescriptor, []byte(changedObjectMem))
				}

				objectExecutionMock := runnerMock.AddExecutionMock("SomeMethod")
				objectExecutionMock.AddStart(
					func(ctx execution.Context) {
						require.Equal(t, objectRef, ctx.Request.Callee)
						require.Equal(t, outgoing, ctx.Outgoing)
						expectedMemory := origValidatedObjectMem
						if testCase.isolation.State == isolation.CallDirty {
							expectedMemory = origDirtyObjectMem
						}
						require.Equal(t, []byte(expectedMemory), ctx.ObjectDescriptor.Memory())
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestResult,
					},
				)
			}
			// setup type checker
			{
				typedChecker.VStateRequest.Set(func(req *payload.VStateRequest) bool {
					require.Equal(t, p1, req.AsOf)
					require.Equal(t, objectRef, req.Object)

					flags := payload.RequestLatestDirtyState | payload.RequestLatestValidatedState |
						payload.RequestOrderedQueue | payload.RequestUnorderedQueue
					require.Equal(t, flags, req.RequestedContent)

					content := &payload.VStateReport_ProvidedContentBody{
						LatestDirtyState: &payload.ObjectState{
							Reference: reference.Local{},
							Class:     class,
							State:     []byte(origDirtyObjectMem),
						},
						LatestValidatedState: &payload.ObjectState{
							Reference: reference.Local{},
							Class:     class,
							State:     []byte(origValidatedObjectMem),
						},
					}

					report := payload.VStateReport{
						Status:          payload.StateStatusReady,
						AsOf:            req.AsOf,
						Object:          objectRef,
						ProvidedContent: content,
					}
					server.SendPayload(ctx, &report)
					return false
				})
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					require.Equal(t, objectRef, res.Callee)
					require.Equal(t, outgoing, res.CallOutgoing)
					require.Equal(t, []byte(callResult), res.ReturnArguments)
					return false
				})
				typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
					require.Equal(t, objectRef, report.Object)
					require.Equal(t, payload.StateStatusReady, report.Status)
					require.True(t, report.DelegationSpec.IsZero())
					require.Equal(t, int32(0), report.UnorderedPendingCount)
					require.Equal(t, int32(0), report.OrderedPendingCount)
					require.NotNil(t, report.ProvidedContent)
					switch testCase.isolation.Interference {
					case isolation.CallIntolerable:
						require.Equal(t, []byte(origDirtyObjectMem), report.ProvidedContent.LatestDirtyState.State)
					case isolation.CallTolerable:
						require.Equal(t, []byte(changedObjectMem), report.ProvidedContent.LatestValidatedState.State)
					default:
						t.Fatal("StateStatusInvalid test case isolation interference")
					}
					return false
				})
				typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
					t.FailNow()
					// TODO add asserts and check counter after https://insolar.atlassian.net/browse/PLAT-753
					return false
				})
			}

			// VCallRequest
			{
				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = payload.BuildCallFlags(testCase.isolation.Interference, testCase.isolation.State)
				pl.Callee = objectRef
				pl.CallSiteMethod = "SomeMethod"
				pl.CallOutgoing = outgoing

				server.SendPayload(ctx, pl)
			}

			commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

			server.IncrementPulseAndWaitIdle(ctx)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VStateReport.Count())
			require.Equal(t, 1, typedChecker.VStateRequest.Count())
			require.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_Method_ForObjectWithMissingState(t *testing.T) {
	insrail.LogCase(t, "C5106")

	testCases := []struct {
		name             string
		outgoingFromPast bool
	}{
		{
			name:             "Call method with prev outgoing.Pulse",
			outgoingFromPast: true,
		},
		{
			name:             "Call method with current outgoing.Pulse",
			outgoingFromPast: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			utils.AssertNotJumpToStep(t, server.Journal, "stepTakeLock")

			server.IncrementPulseAndWaitIdle(ctx)

			execDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
			stateHandled := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)

			var (
				objectRef = server.RandomGlobalWithPulse()
				prevPulse = server.GetPulse().PulseNumber
			)

			server.IncrementPulseAndWaitIdle(ctx)

			if testCase.outgoingFromPast {
				prevPulse = server.GetPulse().PulseNumber
				server.IncrementPulseAndWaitIdle(ctx)
			}

			state := &payload.VStateReport{
				Status: payload.StateStatusMissing,
				AsOf:   prevPulse,
				Object: objectRef,
			}

			server.SendPayload(ctx, state)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, stateHandled)

			outgoing := server.BuildRandomOutgoingWithPulse()

			expectedError := throw.E("object does not exist", struct {
				ObjectReference string
				State           object.State
			}{
				ObjectReference: objectRef.String(),
				State:           object.Missing,
			})

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
				require.Equal(t, payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty), result.CallFlags)
				require.Equal(t, objectRef, result.Callee)
				require.Equal(t, outgoing, result.CallOutgoing)
				require.Equal(t, server.GlobalCaller(), result.Caller)
				require.True(t, result.DelegationSpec.IsZero())
				contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments)
				require.NoError(t, sysErr)
				require.Equal(t, expectedError.Error(), contractErr.Error())
				return false
			})

			pl := utils.GenerateVCallRequestMethod(server)
			pl.CallFlags = payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
			pl.Callee = objectRef
			pl.CallSiteMethod = "Method"
			pl.CallOutgoing = outgoing

			server.SendPayload(ctx, pl)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, execDone)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallResult.Count())
			typedChecker.MinimockWait(10 * time.Second)

			mc.Finish()
		})
	}
}

func TestVirtual_Method_ForbiddenIsolation(t *testing.T) {
	table := []struct {
		name         string
		testRailCase string

		callFlags                  payload.CallFlags
		dirtyStateBuilder          func(objectRef, classRef reference.Global, pn pulse.Number) descriptor.Object
		validatedStateBuilder      func(objectRef, classRef reference.Global, pn pulse.Number) descriptor.Object
		callResult                 []byte
		expectedUnImplementedError bool
	}{
		{
			name:         "Method tolerable + validated cannot be executed",
			testRailCase: "C5449",
			callFlags:    payload.BuildCallFlags(isolation.CallTolerable, isolation.CallValidated),
			dirtyStateBuilder: func(objectRef, classRef reference.Global, pn pulse.Number) descriptor.Object {
				return descriptor.NewObject(
					objectRef,
					execute.NewStateID(pn, []byte("ok case")),
					classRef,
					[]byte("ok case"),
					false,
				)
			},
			validatedStateBuilder: func(objectRef, classRef reference.Global, pn pulse.Number) descriptor.Object {
				return descriptor.NewObject(
					objectRef,
					execute.NewStateID(pn, []byte("not ok case")),
					classRef,
					[]byte("not ok case"),
					false,
				)
			},
			callResult:                 []byte("bad case"),
			expectedUnImplementedError: true,
		},
		{
			name:         "Method intolerable + validated cannot be executed if no validated state",
			testRailCase: "C5475",
			callFlags:    payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated),
			dirtyStateBuilder: func(objectRef, classRef reference.Global, pn pulse.Number) descriptor.Object {
				return descriptor.NewObject(
					objectRef,
					execute.NewStateID(pn, []byte("ok case")),
					classRef,
					[]byte("ok case"),
					false,
				)
			},
			validatedStateBuilder: func(objectRef, classRef reference.Global, pn pulse.Number) descriptor.Object {
				return nil
			},
			callResult:                 []byte("bad case"),
			expectedUnImplementedError: true,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)
			insrail.LogCase(t, test.testRailCase)

			var (
				mc     = minimock.NewController(t)
				ctx    context.Context
				server *utils.Server
			)

			if test.expectedUnImplementedError {
				server, ctx = utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
					return !strings.Contains(s, "execution: not implemented")
				})
			} else {
				server, ctx = utils.NewUninitializedServer(nil, t)
			}

			defer server.Stop()

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			runnerMock := logicless.NewServiceMock(ctx, t, nil)
			server.ReplaceRunner(runnerMock)

			server.Init(ctx)
			server.IncrementPulse(ctx)

			var (
				class       = server.RandomGlobalWithPulse()
				pulseNumber = server.GetPulse().PulseNumber
				objectRef   = server.RandomGlobalWithPulse()

				validatedStateHeadRef reference.Global
				latestValidatedState  *payload.ObjectState
			)

			dirtyState := test.dirtyStateBuilder(objectRef, class, pulseNumber)
			validatedState := test.validatedStateBuilder(objectRef, class, pulseNumber)
			if validatedState != nil {
				validatedStateHeadRef = validatedState.HeadRef()
				latestValidatedState = &payload.ObjectState{
					Reference: validatedState.StateID(),
					Class:     class,
					State:     validatedState.Memory(),
				}
			}

			{ // send object state to server
				pl := payload.VStateReport{
					AsOf:                 pulseNumber,
					Status:               payload.StateStatusReady,
					Object:               objectRef,
					LatestValidatedState: validatedStateHeadRef,
					LatestDirtyState:     dirtyState.HeadRef(),
					ProvidedContent: &payload.VStateReport_ProvidedContentBody{
						LatestValidatedState: latestValidatedState,
						LatestDirtyState: &payload.ObjectState{
							Reference: dirtyState.StateID(),
							Class:     class,
							State:     dirtyState.Memory(),
						},
					},
				}

				server.IncrementPulseAndWaitIdle(ctx)
				server.SendPayload(ctx, &pl)
				server.WaitActiveThenIdleConveyor()
			}

			outgoingRef := server.BuildRandomOutgoingWithPulse()

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			{
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					require.Equal(t, objectRef, res.Callee)
					assert.Equal(t, test.callResult, res.ReturnArguments)
					return false // no resend msg
				})

				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = test.callFlags
				pl.Callee = objectRef
				pl.CallOutgoing = outgoingRef

				key := pl.CallOutgoing.String()

				runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
					Interference: test.callFlags.GetInterference(),
					State:        test.callFlags.GetState(),
				}, nil)

				// if we use forbidden isolation, then execute should stop before Start happen
				if !test.expectedUnImplementedError {
					result := requestresult.New(test.callResult, outgoingRef)
					result.SetAmend(dirtyState, []byte("new stuff"))
					runnerMock.AddExecutionMock(key).
						AddStart(nil, &execution.Update{
							Type:   execution.Done,
							Result: result,
						})

				}

				server.SendPayload(ctx, pl)
			}

			commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			if test.expectedUnImplementedError {
				assert.Equal(t, 0, typedChecker.VCallResult.Count())
			} else {
				assert.Equal(t, 1, typedChecker.VCallResult.Count())
			}

			mc.Finish()
		})
	}
}

func TestVirtual_Method_IntolerableCallChangeState(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5463")

	const (
		origObjectMem    = "original object memory"
		changedObjectMem = "new object memory"
	)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	{
		server.ReplaceRunner(runnerMock)
		server.Init(ctx)
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		class     = server.RandomGlobalWithPulse()
		objectRef = server.RandomGlobalWithPulse()
		p1        = server.GetPulse().PulseNumber
		isolation = contract.MethodIsolation{
			Interference: isolation.CallIntolerable,
			State:        isolation.CallValidated,
		}
	)

	server.IncrementPulseAndWaitIdle(ctx)
	outgoing := server.BuildRandomOutgoingWithPulse()

	{
		runnerMock.AddExecutionClassify("SomeMethod", isolation, nil)
		requestResult := requestresult.New([]byte("call result"), server.RandomGlobalWithPulse())
		newObjDescriptor := descriptor.NewObject(
			reference.Global{}, reference.Local{}, class, []byte(""), false,
		)
		requestResult.SetAmend(newObjDescriptor, []byte(changedObjectMem))

		objectExecutionMock := runnerMock.AddExecutionMock("SomeMethod")
		objectExecutionMock.AddStart(
			func(ctx execution.Context) {
				require.Equal(t, objectRef, ctx.Request.Callee)
				require.Equal(t, []byte(origObjectMem), ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
	}
	{
		typedChecker.VStateRequest.Set(func(req *payload.VStateRequest) bool {
			require.Equal(t, p1, req.AsOf)
			require.Equal(t, objectRef, req.Object)

			flags := payload.RequestLatestDirtyState | payload.RequestLatestValidatedState |
				payload.RequestOrderedQueue | payload.RequestUnorderedQueue
			require.Equal(t, flags, req.RequestedContent)

			content := &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: reference.Local{},
					Class:     testwalletProxy.GetClass(),
					State:     []byte(origObjectMem),
				},
				LatestValidatedState: &payload.ObjectState{
					Reference: reference.Local{},
					Class:     testwalletProxy.GetClass(),
					State:     []byte(origObjectMem),
				},
			}
			report := payload.VStateReport{
				Status:          payload.StateStatusReady,
				AsOf:            req.AsOf,
				Object:          objectRef,
				ProvidedContent: content,
			}
			server.SendPayload(ctx, &report)
			return false // no resend msg
		})

		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, objectRef, res.Callee)
			require.Equal(t, outgoing, res.CallOutgoing)
			contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
			require.NoError(t, sysErr)
			require.Equal(t, "intolerable call trying to change object state", contractErr.Error())
			return false // no resend msg
		})
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			require.Equal(t, objectRef, report.Object)
			require.Equal(t, payload.StateStatusReady, report.Status)
			require.True(t, report.DelegationSpec.IsZero())
			require.Equal(t, int32(0), report.UnorderedPendingCount)
			require.Equal(t, int32(0), report.OrderedPendingCount)
			require.NotNil(t, report.ProvidedContent)
			require.Equal(t, []byte(origObjectMem), report.ProvidedContent.LatestDirtyState.State)
			return false
		})
		typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
			t.FailNow()
			// TODO add asserts and check counter after https://insolar.atlassian.net/browse/PLAT-753
			return false
		})
	}
	{
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)
		pl.Callee = objectRef
		pl.CallSiteMethod = "SomeMethod"
		pl.CallOutgoing = outgoing

		server.SendPayload(ctx, pl)
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

	server.IncrementPulseAndWaitIdle(ctx)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VStateReport.Count())
	require.Equal(t, 1, typedChecker.VStateRequest.Count())
	require.Equal(t, 1, typedChecker.VCallResult.Count())
	typedChecker.MinimockFinish()

	server.Stop()
	mc.Finish()
}

// Send VStateReport with Dirty == Validated state
// Get Validated, Dirty state and check (GetValidatedMethod1, GetDirtyMethod1)
// Set new Dirty state (ChangeMethod)
// Get Validated, Dirty state and check, Dirty state must be new (GetValidatedMethod2, GetDirtyMethod2)
// Change pulse and check VStateReport, Validated state == Dirty == newState
func TestVirtual_Method_CheckValidatedState(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5124")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})

	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		class        = server.RandomGlobalWithPulse()
		initialState = []byte("initial state")
		newState     = []byte("updated state")
		prevPulse    = server.GetPulse().PulseNumber
	)

	// add ExecutionMock to runnerMock
	{
		objectDescriptor := descriptor.NewObject(
			objectGlobal,
			execute.NewStateID(server.GetPulse().PulseNumber, newState),
			class,
			newState,
			false,
		)
		requestResult := requestresult.New([]byte("done"), server.RandomGlobalWithPulse())
		requestResult.SetAmend(objectDescriptor, newState)

		runnerMock.AddExecutionMock("ChangeMethod").AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, initialState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
		runnerMock.AddExecutionClassify("ChangeMethod", tolerableFlags(), nil)

		runnerMock.AddExecutionMock("GetValidatedMethod1").AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, isolation.CallValidated, ctx.Isolation.State)
				assert.Equal(t, initialState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("get validated info 1"), server.RandomGlobalWithPulse()),
			},
		)
		runnerMock.AddExecutionMock("GetDirtyMethod1").AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, isolation.CallDirty, ctx.Isolation.State)
				assert.Equal(t, initialState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("get dirty info 1"), server.RandomGlobalWithPulse()),
			},
		)
		runnerMock.AddExecutionMock("GetValidatedMethod2").AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, isolation.CallValidated, ctx.Isolation.State)
				assert.Equal(t, initialState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("get validated info 2"), server.RandomGlobalWithPulse()),
			},
		)
		runnerMock.AddExecutionMock("GetDirtyMethod2").AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, isolation.CallDirty, ctx.Isolation.State)
				assert.Equal(t, newState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("get dirty info 2"), server.RandomGlobalWithPulse()),
			},
		)
		runnerMock.AddExecutionClassify("GetValidatedMethod1", intolerableFlags(), nil)
		runnerMock.AddExecutionClassify("GetValidatedMethod2", intolerableFlags(), nil)
		runnerMock.AddExecutionClassify("GetDirtyMethod1", contract.MethodIsolation{Interference: isolation.CallIntolerable, State: isolation.CallDirty}, nil)
		runnerMock.AddExecutionClassify("GetDirtyMethod2", contract.MethodIsolation{Interference: isolation.CallIntolerable, State: isolation.CallDirty}, nil)
	}

	// add typedChecker mock
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			assert.Equal(t, newState, report.ProvidedContent.LatestDirtyState.State)
			assert.Equal(t, newState, report.ProvidedContent.LatestValidatedState.State)
			return false
		})
		typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
			t.FailNow()
			// TODO add asserts and check counter after https://insolar.atlassian.net/browse/PLAT-753
			return false
		})
		typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
			switch string(result.ReturnArguments) {
			case "get validated info 1":
			case "get validated info 2":
			case "get dirty info 1":
			case "get dirty info 2":
			case "done":
			default:
				t.Fatalf("unexpected result")
			}
			return false
		})
	}

	// initial object state
	{
		// need for correct handle state report (should from prev pulse)
		server.IncrementPulse(ctx)

		report := &payload.VStateReport{
			Status: payload.StateStatusReady,
			Object: objectGlobal,
			AsOf:   prevPulse,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: reference.Local{},
					Class:     class,
					State:     initialState,
				},
				LatestValidatedState: &payload.ObjectState{
					Reference: reference.Local{},
					Class:     class,
					State:     initialState,
				},
			},
		}
		waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, report)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, waitReport)
	}

	// get object state
	{
		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "GetValidatedMethod1"

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

		executeDone = server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		pl = utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "GetDirtyMethod1"

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	}

	// change object state
	{
		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "ChangeMethod"

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	}

	// check object state
	{
		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "GetValidatedMethod2"

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

		executeDone = server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

		pl = utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "GetDirtyMethod2"

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	}

	// increment pulse and check VStateReport
	{
		server.IncrementPulse(ctx)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
		// wait for all VCallResults
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

		require.Equal(t, 1, typedChecker.VStateReport.Count())
		require.Equal(t, 5, typedChecker.VCallResult.Count())
	}
	mc.Finish()
}

func TestVirtual_Method_TwoUnorderedCalls(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5499")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)

	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	// init object
	{
		// need for correct handle state report (should from prev pulse)
		server.IncrementPulse(ctx)
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)
	}

	var (
		firstOutgoing  = server.BuildRandomOutgoingWithPulse()
		secondOutgoing = server.BuildRandomOutgoingWithPulse()
	)

	synchronizeExecution := synchronization.NewPoint(1)
	defer synchronizeExecution.Done()

	// add ExecutionMock to runnerMock
	{
		runnerMock.AddExecutionMock(firstOutgoing.String()).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, objectGlobal, ctx.Object)
				assert.Equal(t, isolation.CallIntolerable, ctx.Isolation.Interference)
				synchronizeExecution.Synchronize()
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("first result"), objectGlobal),
			},
		)

		runnerMock.AddExecutionMock(secondOutgoing.String()).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, objectGlobal, ctx.Object)
				assert.Equal(t, isolation.CallIntolerable, ctx.Isolation.Interference)
				synchronizeExecution.WakeUp()
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("second result"), objectGlobal),
			},
		)

		runnerMock.AddExecutionClassify(firstOutgoing.String(), intolerableFlags(), nil)
		runnerMock.AddExecutionClassify(secondOutgoing.String(), intolerableFlags(), nil)
	}

	// add typedChecker mock
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
			switch result.CallOutgoing {
			case firstOutgoing:
				assert.Equal(t, []byte("first result"), result.ReturnArguments)
			case secondOutgoing:
				assert.Equal(t, []byte("second result"), result.ReturnArguments)
			default:
				t.Fatalf("unexpected outgoing")
			}
			return false
		})
	}

	// 2 Tolerable VCallRequests will be converted to Intolerable
	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)
	{
		// send first
		pl := utils.GenerateVCallRequestMethod(server)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "GetMethod"
		pl.CallOutgoing = firstOutgoing

		server.SendPayload(ctx, pl)
		// wait for the first
		commontestutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
		// send second
		pl = utils.GenerateVCallRequestMethod(server)
		pl.Callee = objectGlobal
		pl.CallSiteMethod = "GetMethod"
		pl.CallOutgoing = secondOutgoing

		server.SendPayload(ctx, pl)
	}

	// wait for all VCallResults
	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}
