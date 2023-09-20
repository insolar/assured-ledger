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
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/recordchecker"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/vnlmn"

	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
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
	state rms.VStateReport_StateStatus,
	object reference.Global,
	pulse rms.PulseNumber,
) {
	var (
		walletState = makeRawWalletState(initialBalance)
	)

	builder := utils.NewStateReportBuilder().Pulse(pulse).Object(object)

	switch state {
	case rms.StateStatusMissing:
		builder = builder.Missing()
	case rms.StateStatusReady:
		builder = builder.Ready().Class(testwalletProxy.GetClass()).Memory(walletState)
	case rms.StateStatusInactive:
		builder = builder.Inactive()
	default:
		panic("unexpected state")
	}

	wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	server.SendPayload(ctx, builder.ReportPtr())

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

	Method_PrepareObject(ctx, server, rms.StateStatusReady, objectGlobal, prevPulse)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	expectedError, err := foundation.MarshalMethodErrorResult(
		throw.W(throw.E("failed to find contracts method"), "failed to classify method"))

	require.NoError(t, err)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		assert.Equal(t, objectGlobal, res.Callee.GetValue())
		assert.Equal(t, outgoing, res.CallOutgoing.GetValue())
		assert.Equal(t, expectedError, res.ReturnArguments.GetBytes())

		return false // no resend msg
	})

	{
		pl := utils.GenerateVCallRequestMethodImmutable(server)
		pl.Callee.Set(objectGlobal)
		pl.CallOutgoing.Set(outgoing)

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

	Method_PrepareObject(ctx, server, rms.StateStatusMissing, objectGlobal, prevPulse)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	expectedError := throw.E("object does not exist", struct {
		ObjectReference string
		State           object.State
	}{
		ObjectReference: objectGlobal.String(),
		State:           object.Missing,
	})

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		assert.Equal(t, objectGlobal, res.Callee.GetValue())
		assert.Equal(t, outgoing, res.CallOutgoing.GetValue())

		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
		require.NoError(t, sysErr)
		assert.Equal(t, expectedError.Error(), contractErr.Error())

		return false // no resend msg
	})

	{
		pl := utils.GenerateVCallRequestMethodImmutable(server)
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "GetBalance"
		pl.CallOutgoing.Set(outgoing)

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

	runnerMock := logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)

	var (
		parallelCount     = 2
		waitInputChannel  = make(chan struct{}, parallelCount)
		waitOutputChannel = make(chan struct{}, 0)
		executeDone       = server.Journal.WaitStopOf(&execute.SMExecute{}, parallelCount)

		objectGlobal = server.RandomGlobalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	// need for correct handle state report (should from prev pulse)
	server.IncrementPulse(ctx)

	Method_PrepareObject(ctx, server, rms.StateStatusReady, objectGlobal, prevPulse)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	{
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, []byte("345"), res.ReturnArguments.GetBytes())
			assert.Equal(t, objectGlobal, res.Callee.GetValue())

			return false // no resend msg
		})

		countBefore := server.PublisherMock.GetCount()

		for i := 0; i < parallelCount; i++ {
			pl := utils.GenerateVCallRequestMethodImmutable(server)
			pl.Callee.Set(objectGlobal)
			pl.CallSiteMethod = "GetBalance"

			result := requestresult.New([]byte("345"), objectGlobal)

			key := pl.CallOutgoing.GetValue()
			runnerMock.AddExecutionMock(key).AddStart(func(_ execution.Context) {
				// tell the test that we know about next request
				waitInputChannel <- struct{}{}

				// wait the test result
				<-waitOutputChannel
			}, &execution.Update{
				Type:   execution.Done,
				Result: result,
			})

			runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
				Interference: pl.GetCallFlags().GetInterference(),
				State:        pl.GetCallFlags().GetState(),
			}, nil)

			server.SendPayload(ctx, pl)
		}

		for i := 0; i < parallelCount; i++ {
			select {
			case <-waitInputChannel:
			case <-time.After(10 * time.Second):
				require.Failf(t, "", "timeout")
			}
		}

		for i := 0; i < parallelCount; i++ {
			waitOutputChannel <- struct{}{}
		}

		if !server.PublisherMock.WaitCount(countBefore+2, 10*time.Second) {
			t.Error("failed to wait for result")
		}
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, parallelCount, typedChecker.VCallResult.Count())
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

	var (
		objectGlobal  = server.RandomGlobalWithPulse()
		prevPulse     = server.GetPulse().PulseNumber
		parallelCount = 2
		awaitFullStop = server.Journal.WaitStopOf(&execute.SMExecute{}, parallelCount)
	)

	// need for correct handle state report (should from prev pulse)
	server.IncrementPulse(ctx)

	Method_PrepareObject(ctx, server, rms.StateStatusReady, objectGlobal, prevPulse)
	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	counter := 0

	{
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, []byte("345"), res.ReturnArguments.GetBytes())
			assert.Equal(t, objectGlobal, res.Callee.GetValue())
			return false // no resend msg
		})

		for i := 0; i < parallelCount; i++ {
			pl := utils.GenerateVCallRequestMethod(server)
			pl.Callee.Set(objectGlobal)
			pl.CallSiteMethod = "ordered-" + strconv.FormatInt(int64(i), 10)

			result := requestresult.New([]byte("345"), objectGlobal)

			key := pl.CallOutgoing.GetValue()
			runnerMock.AddExecutionMock(key).AddStart(func(ctx execution.Context) {
				counter++
				for k := 0; k < 5; k++ {
					require.Equal(t, 1, counter)
					time.Sleep(10 * time.Millisecond)
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

	assert.Equal(t, parallelCount, typedChecker.VCallResult.Count())

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

			runnerMock := logicless.NewServiceMock(ctx, mc, nil)
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

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

			Method_PrepareObject(ctx, server, rms.StateStatusReady, objectAGlobal, prevPulse)

			outgoingCallRef := server.RandomGlobalWithPulse()
			outgoing := server.BuildRandomOutgoingWithPulse()

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
			objectAExecutionMock := runnerMock.AddExecutionMock(outgoing)
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

			runnerMock.AddExecutionClassify(outgoing, flags, nil)

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
				switch res.Callee.GetValue() {
				case objectAGlobal:
					assert.Equal(t, expectedResult, res.ReturnArguments.GetBytes())
				default:
					assert.Fail(t, "unexpected VCallResult")
				}
				return false
			})

			pl := utils.GenerateVCallRequestMethod(server)
			pl.CallFlags = rms.BuildCallFlags(flags.Interference, flags.State)
			pl.Callee.Set(objectAGlobal)
			pl.CallSiteMethod = "Foo"
			pl.CallOutgoing.Set(outgoing)

			server.SendPayload(ctx, pl)
			{
				commontestutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
				commontestutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
			}

			assert.Equal(t, 0, typedChecker.VCallRequest.Count())
			assert.Equal(t, 1, typedChecker.VCallResult.Count())

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

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) interface{} {
		return execution.Request.Callee.GetValue()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	var (
		objectA        = server.RandomGlobalWithPulse()
		objectB1Global = server.RandomGlobalWithPulse()
		objectB2Global = server.RandomGlobalWithPulse()
		objectB3Global = server.RandomGlobalWithPulse()
		prevPulse      = server.GetPulseNumber()
	)

	server.IncrementPulseAndWaitIdle(ctx)

	// create objects
	{
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectA, prevPulse)
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectB1Global, prevPulse)
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectB2Global, prevPulse)
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectB3Global, prevPulse)
	}

	p := server.GetPulseNumber()

	var (
		flags     = contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty}
		callFlags = rms.BuildCallFlags(flags.Interference, flags.State)

		classB1 = server.RandomGlobalWithPulse()
		classB2 = server.RandomGlobalWithPulse()
		classB3 = server.RandomGlobalWithPulse()

		outgoingCallRef = server.BuildRandomOutgoingWithPulse()
	)

	// add ExecutionMocks to runnerMock
	{
		builder := execution.NewRPCBuilder(outgoingCallRef, objectA)
		objectAExecutionMock := runnerMock.AddExecutionMock(objectA)
		objectAExecutionMock.AddStart(func(ctx execution.Context) {
			logger.Debug("ExecutionStart [A.Foo]")
			assert.Equal(t, server.GlobalCaller(), ctx.Request.Caller.GetValue())
			assert.Equal(t, objectA, ctx.Request.Callee.GetValue())
			assert.Equal(t, outgoingCallRef, ctx.Request.CallOutgoing.GetValue())
		}, &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: builder.CallMethod(objectB1Global, classB1, "Bar", []byte("B1")),
		})

		objectAExecutionMock.AddContinue(func(result []byte) {
			logger.Debug("ExecutionContinue [A.Foo]")
			assert.Equal(t, []byte("finish B1.Bar"), result)
		}, &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: builder.CallMethod(objectB2Global, classB2, "Bar", []byte("B2")),
		})

		objectAExecutionMock.AddContinue(func(result []byte) {
			logger.Debug("ExecutionContinue [A.Foo]")
			assert.Equal(t, []byte("finish B2.Bar"), result)
		}, &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: builder.CallMethod(objectB3Global, classB3, "Bar", []byte("B3")),
		})

		objectAExecutionMock.AddContinue(func(result []byte) {
			logger.Debug("ExecutionContinue [A.Foo]")
			assert.Equal(t, []byte("finish B3.Bar"), result)
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectA),
		})

		runnerMock.AddExecutionMock(objectB1Global).AddStart(func(ctx execution.Context) {
			logger.Debug("ExecutionStart [B1.Bar]")
			assert.Equal(t, objectB1Global, ctx.Request.Callee.GetValue())
			assert.Equal(t, objectA, ctx.Request.Caller.GetValue())
			assert.Equal(t, []byte("B1"), ctx.Request.Arguments.GetBytes())
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish B1.Bar"), objectB1Global),
		})

		runnerMock.AddExecutionMock(objectB2Global).AddStart(func(ctx execution.Context) {
			logger.Debug("ExecutionStart [B2.Bar]")
			assert.Equal(t, objectB2Global, ctx.Request.Callee.GetValue())
			assert.Equal(t, objectA, ctx.Request.Caller.GetValue())
			assert.Equal(t, []byte("B2"), ctx.Request.Arguments.GetBytes())
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish B2.Bar"), objectB2Global),
		})

		runnerMock.AddExecutionMock(objectB3Global).AddStart(func(ctx execution.Context) {
			logger.Debug("ExecutionStart [B3.Bar]")
			assert.Equal(t, objectB3Global, ctx.Request.Callee.GetValue())
			assert.Equal(t, objectA, ctx.Request.Caller.GetValue())
			assert.Equal(t, []byte("B3"), ctx.Request.Arguments.GetBytes())
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish B3.Bar"), objectB3Global),
		})

		runnerMock.AddExecutionClassify(objectA, flags, nil)
		runnerMock.AddExecutionClassify(objectB1Global, flags, nil)
		runnerMock.AddExecutionClassify(objectB2Global, flags, nil)
		runnerMock.AddExecutionClassify(objectB3Global, flags, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
			assert.Equal(t, objectA, request.Caller.GetValue())
			assert.Equal(t, rms.CallTypeMethod, request.CallType)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, p, request.CallOutgoing.GetPulseOfLocal())

			switch request.Callee.GetValue() {
			case objectB1Global:
				assert.Equal(t, []byte("B1"), request.Arguments.GetBytes())
				assert.Equal(t, uint32(1), request.CallSequence)
			case objectB2Global:
				assert.Equal(t, []byte("B2"), request.Arguments.GetBytes())
				assert.Equal(t, uint32(2), request.CallSequence)
			case objectB3Global:
				assert.Equal(t, []byte("B3"), request.Arguments.GetBytes())
				assert.Equal(t, uint32(3), request.CallSequence)
			default:
				t.Fatal("wrong Callee")
			}
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, rms.CallTypeMethod, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee.GetValue() {
			case objectA:
				assert.Equal(t, []byte("finish A.Foo"), res.ReturnArguments.GetBytes())
				assert.Equal(t, server.GlobalCaller(), res.Caller.GetValue())
				assert.Equal(t, outgoingCallRef, res.CallOutgoing.GetValue())
			case objectB1Global:
				assert.Equal(t, []byte("finish B1.Bar"), res.ReturnArguments.GetBytes())
				assert.Equal(t, objectA, res.Caller.GetValue())
				assert.Equal(t, p, res.CallOutgoing.GetPulseOfLocal())
			case objectB2Global:
				assert.Equal(t, []byte("finish B2.Bar"), res.ReturnArguments.GetBytes())
				assert.Equal(t, objectA, res.Caller.GetValue())
				assert.Equal(t, p, res.CallOutgoing.GetPulseOfLocal())
			case objectB3Global:
				assert.Equal(t, []byte("finish B3.Bar"), res.ReturnArguments.GetBytes())
				assert.Equal(t, objectA, res.Caller.GetValue())
				assert.Equal(t, p, res.CallOutgoing.GetPulseOfLocal())
			default:
				t.Fatal("wrong Callee")
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller.GetValue() == objectA
		})
	}

	pl := utils.GenerateVCallRequestMethod(server)
	pl.CallFlags = callFlags
	pl.Callee.Set(objectA)
	pl.CallOutgoing.Set(outgoingCallRef)

	server.SendPayload(ctx, pl)

	// wait for all calls and SMs
	commontestutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 3, typedChecker.VCallRequest.Count())
	assert.Equal(t, 4, typedChecker.VCallResult.Count())

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

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) interface{} {
		return execution.Request.CallOutgoing.GetValue()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	var (
		flags     = contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty}
		callFlags = rms.BuildCallFlags(flags.Interference, flags.State)

		classB = server.RandomGlobalWithPulse()

		objectAGlobal = server.RandomGlobalWithPulse()
		objectBGlobal = server.RandomGlobalWithPulse()

		prevPulse = server.GetPulse().PulseNumber
	)

	server.IncrementPulseAndWaitIdle(ctx)

	// create objects
	{
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectAGlobal, prevPulse)
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectBGlobal, prevPulse)
	}

	var (
		outgoingFirstCall  = server.BuildRandomOutgoingWithPulse()
		outgoingSecondCall = server.BuildRandomOutgoingWithPulse()
	)

	// add ExecutionMocks to runnerMock
	{
		firstBuilder := execution.NewRPCBuilder(outgoingFirstCall, objectAGlobal)
		objectAExecutionFirstMock := runnerMock.AddExecutionMock(outgoingFirstCall)
		objectAExecutionFirstMock.AddStart(nil,
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: firstBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("first")),
			},
		)
		objectAExecutionFirstMock.AddContinue(
			func(result []byte) {
				assert.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: firstBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("second")),
			},
		)
		objectAExecutionFirstMock.AddContinue(
			func(result []byte) {
				assert.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		secondBuilder := execution.NewRPCBuilder(outgoingSecondCall, objectAGlobal)
		objectAExecutionSecondMock := runnerMock.AddExecutionMock(outgoingSecondCall)
		objectAExecutionSecondMock.AddStart(nil,
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: secondBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("first")),
			},
		)
		objectAExecutionSecondMock.AddContinue(
			func(result []byte) {
				assert.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: secondBuilder.CallMethod(objectBGlobal, classB, "Bar", []byte("second")),
			},
		)
		objectAExecutionSecondMock.AddContinue(
			func(result []byte) {
				assert.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
			},
		)

		runnerMock.AddExecutionClassify(outgoingFirstCall, flags, nil)
		runnerMock.AddExecutionClassify(outgoingSecondCall, flags, nil)
	}

	// add publish checker
	{
		typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
			switch string(request.Arguments.GetBytes()[0]) {
			case "f":
				assert.Equal(t, []byte("first"), request.Arguments.GetBytes())
				assert.Equal(t, uint32(1), request.CallSequence)
			case "s":
				assert.Equal(t, []byte("second"), request.Arguments.GetBytes())
				assert.Equal(t, uint32(2), request.CallSequence)
			default:
				t.Fatal("wrong call args")
			}

			result := rms.VCallResult{
				CallType:        request.CallType,
				CallFlags:       request.CallFlags,
				Caller:          request.Caller,
				Callee:          request.Callee,
				CallOutgoing:    request.CallOutgoing,
				CallIncoming:    rms.NewReference(server.RandomGlobalWithPulse()),
				ReturnArguments: rms.NewBytes([]byte("finish B.Bar")),
			}
			msg := server.WrapPayload(&result).Finalize()
			server.SendMessage(ctx, msg)

			return false
		})
		typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
			assert.Equal(t, []byte("finish A.Foo"), result.ReturnArguments.GetBytes())
			return false
		})
	}

	{ // send first VCallRequest A.Foo
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = callFlags
		pl.Callee.Set(objectAGlobal)
		pl.CallSiteMethod = "Foo"
		pl.CallOutgoing.Set(outgoingFirstCall)
		pl.Arguments.SetBytes([]byte("call foo"))

		server.SendPayload(ctx, pl)
	}

	server.WaitActiveThenIdleConveyor()

	{ // send second VCallRequest A.Foo
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = callFlags
		pl.Callee.Set(objectAGlobal)
		pl.CallSiteMethod = "Foo"
		pl.CallOutgoing.Set(outgoingSecondCall)
		pl.Arguments.SetBytes([]byte("call foo"))

		server.SendPayload(ctx, pl)
	}

	// wait for all calls and SMs
	{
		commontestutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
		commontestutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	assert.Equal(t, 4, typedChecker.VCallRequest.Count())
	assert.Equal(t, 2, typedChecker.VCallResult.Count())

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

	Method_PrepareObject(ctx, server, rms.StateStatusReady, objectGlobal, prevPulse)

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

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		assert.Equal(t, objectGlobal, res.Callee.GetValue())
		assert.Equal(t, outgoing, res.CallOutgoing.GetValue())

		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
		require.NoError(t, sysErr)
		require.NotNil(t, contractErr)
		assert.Equal(t, expectedError.Error(), contractErr.Error())

		return false // no resend msg
	})

	{
		pl := utils.GenerateVCallRequestMethodImmutable(server)
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "Accept"
		pl.CallOutgoing.Set(outgoing)

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

	// legitimate sender
	jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{server.GlobalCaller()}, nil)
	jetCoordinatorMock.MeMock.Return(server.GlobalCaller())

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		class        = server.RandomGlobalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	const (
		validatedMem = "12345"
		dirtyMem     = "54321"
	)

	server.IncrementPulseAndWaitIdle(ctx)

	Method_PrepareObject(ctx, server, rms.StateStatusReady, objectGlobal, prevPulse)

	p := server.GetPulse().PulseNumber

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool { return false })
	typedChecker.VStateReport.Set(func(res *rms.VStateReport) bool { return false })
	typedChecker.VStateRequest.Set(func(res *rms.VStateRequest) bool {
		report := utils.NewStateReportBuilder().Pulse(p).Object(objectGlobal).Ready().
			ValidatedMemory([]byte(validatedMem)).DirtyMemory([]byte(dirtyMem)).Class(class).Report()
		server.SendPayload(ctx, &report)
		return false
	})

	pl := utils.GenerateVCallRequestMethodImmutable(server)
	pl.Callee.Set(objectGlobal)
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

			runnerMock := logicless.NewServiceMock(ctx, mc, nil)
			{
				server.ReplaceRunner(runnerMock)
				server.Init(ctx)
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			recordChecker := recordchecker.NewChecker(mc)

			var (
				class     = server.RandomGlobalWithPulse()
				objectRef = server.RandomGlobalWithPulse()
				p1        = server.GetPulse().PulseNumber
			)

			server.IncrementPulseAndWaitIdle(ctx)
			outgoing := server.BuildRandomOutgoingWithPulse()

			// setup LMN record checker
			{
				chain := recordChecker.CreateChainFromReference(rms.NewReference(objectRef))
				switch testCase.isolation.Interference {
				case isolation.CallIntolerable:
					inbound := chain.AddMessage(
						rms.LRegisterRequest{
							AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RInboundRequest{}),
						},
						recordchecker.ProduceResponse(ctx, server),
					)
					inbound.AddMessage(
						rms.LRegisterRequest{
							AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RInboundResponse{}),
						},
						recordchecker.ProduceResponse(ctx, server),
					)
				case isolation.CallTolerable:
					inbound := chain.AddMessage(
						rms.LRegisterRequest{
							AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RLineInboundRequest{}),
						},
						recordchecker.ProduceResponse(ctx, server),
					)
					inbound.AddMessage(
						rms.LRegisterRequest{
							AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RInboundResponse{}),
						},
						recordchecker.ProduceResponse(ctx, server),
					)
					inbound.AddMessage(
						rms.LRegisterRequest{
							AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RLineMemory{}),
						},
						recordchecker.ProduceResponse(ctx, server),
					)
				}
			}

			// add ExecutionMock to runnerMock
			{
				runnerMock.AddExecutionClassify(outgoing, testCase.isolation, nil)
				requestResult := requestresult.New([]byte(callResult), objectRef)
				if testCase.canChangeState {
					newObjDescriptor := descriptor.NewObject(
						reference.Global{}, reference.Local{}, class, []byte(""), false,
					)
					requestResult.SetAmend(newObjDescriptor, []byte(changedObjectMem))
				}

				objectExecutionMock := runnerMock.AddExecutionMock(outgoing)
				objectExecutionMock.AddStart(func(ctx execution.Context) {
					assert.Equal(t, objectRef, ctx.Request.Callee.GetValue())
					assert.Equal(t, outgoing, ctx.Outgoing)
					expectedMemory := origValidatedObjectMem
					if testCase.isolation.State == isolation.CallDirty {
						expectedMemory = origDirtyObjectMem
					}
					assert.Equal(t, []byte(expectedMemory), ctx.ObjectDescriptor.Memory())
				}, &execution.Update{
					Type:   execution.Done,
					Result: requestResult,
				})
			}
			// setup type checker
			{
				typedChecker.VStateRequest.Set(func(req *rms.VStateRequest) bool {
					assert.Equal(t, p1, req.AsOf)
					assert.Equal(t, objectRef, req.Object.GetValue())

					flags := rms.RequestLatestDirtyState | rms.RequestLatestValidatedState |
						rms.RequestOrderedQueue | rms.RequestUnorderedQueue
					assert.Equal(t, flags, req.RequestedContent)

					report := utils.NewStateReportBuilder().Pulse(req.AsOf).Object(objectRef).
						Ready().Class(class).DirtyMemory([]byte(origDirtyObjectMem)).
						ValidatedMemory([]byte(origValidatedObjectMem)).
						Report()

					server.SendPayload(ctx, &report)
					return false
				})
				typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
					assert.Equal(t, objectRef, res.Callee.GetValue())
					assert.Equal(t, outgoing, res.CallOutgoing.GetValue())
					assert.Equal(t, []byte(callResult), res.ReturnArguments.GetBytes())
					return false
				})
				typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
					assert.Equal(t, objectRef, report.Object.GetValue())
					assert.Equal(t, rms.StateStatusReady, report.Status)
					assert.True(t, report.DelegationSpec.IsZero())
					assert.Equal(t, int32(0), report.UnorderedPendingCount)
					assert.Equal(t, int32(0), report.OrderedPendingCount)
					require.NotNil(t, report.ProvidedContent)
					switch testCase.isolation.Interference {
					case isolation.CallIntolerable:
						assert.Equal(t, []byte(origDirtyObjectMem), report.ProvidedContent.LatestDirtyState.Memory.GetBytes())
					case isolation.CallTolerable:
						assert.Equal(t, []byte(changedObjectMem), report.ProvidedContent.LatestValidatedState.Memory.GetBytes())
					default:
						t.Fatal("StateStatusInvalid test case isolation interference")
					}
					return false
				})
				typedChecker.LRegisterRequest.Set(func(request *rms.LRegisterRequest) bool {
					var (
						chainChecker recordchecker.ChainValidator
						err          error
					)
					chainChecker = recordChecker.GetChainValidatorList().GetChainValidatorByReference(objectRef)
					require.NotNil(t, chainChecker)
					chainChecker, err = chainChecker.Feed(*request)
					require.NoError(t, err)
					require.NotNil(t, chainChecker)
					chainChecker.GetProduceResponseFunc()(*request)
					return false
				})
			}

			// VCallRequest
			{
				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = rms.BuildCallFlags(testCase.isolation.Interference, testCase.isolation.State)
				pl.Callee.Set(objectRef)
				pl.CallSiteMethod = "SomeMethod"
				pl.CallOutgoing.Set(outgoing)

				server.SendPayload(ctx, pl)
			}

			commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

			server.IncrementPulseAndWaitIdle(ctx)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VStateReport.Count())
			assert.Equal(t, 1, typedChecker.VStateRequest.Count())
			assert.Equal(t, 1, typedChecker.VCallResult.Count())

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

			state := server.StateReportBuilder().Object(objectRef).Missing().Report()

			server.IncrementPulseAndWaitIdle(ctx)

			server.SendPayload(ctx, &state)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, stateHandled)

			outgoing := server.BuildRandomOutgoingWithPulse()
			if testCase.outgoingFromPast {
				outgoing = server.BuildRandomOutgoingWithGivenPulse(prevPulse)
			}

			expectedError := throw.E("object does not exist", struct {
				ObjectReference string
				State           object.State
			}{
				ObjectReference: objectRef.String(),
				State:           object.Missing,
			})

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
				assert.Equal(t, rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty), result.CallFlags)
				assert.Equal(t, objectRef, result.Callee.GetValue())
				assert.Equal(t, outgoing, result.CallOutgoing.GetValue())
				assert.Equal(t, server.GlobalCaller(), result.Caller.GetValue())
				assert.True(t, result.DelegationSpec.IsZero())
				contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments.GetBytes())
				require.NoError(t, sysErr)
				assert.Equal(t, expectedError.Error(), contractErr.Error())
				return false
			})

			pl := utils.GenerateVCallRequestMethod(server)
			pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
			pl.Callee.Set(objectRef)
			pl.CallSiteMethod = "Method"
			pl.CallOutgoing.Set(outgoing)

			server.SendPayload(ctx, pl)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, execDone)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VCallResult.Count())
			typedChecker.MinimockWait(10 * time.Second)

			mc.Finish()
		})
	}
}

func TestVirtual_Method_ForbiddenIsolation(t *testing.T) {
	table := []struct {
		name         string
		testRailCase string

		callFlags                  rms.CallFlags
		dirtyMemory                []byte
		validatedMemory            []byte
		callResult                 []byte
		expectedUnImplementedError bool
	}{
		{
			name:                       "Method tolerable + validated cannot be executed",
			testRailCase:               "C5449",
			callFlags:                  rms.BuildCallFlags(isolation.CallTolerable, isolation.CallValidated),
			dirtyMemory:                []byte("ok case"),
			validatedMemory:            []byte("not ok case"),
			callResult:                 []byte("bad case"),
			expectedUnImplementedError: true,
		},
		{
			name:                       "Method intolerable + validated cannot be executed if no validated state",
			testRailCase:               "C5475",
			callFlags:                  rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated),
			dirtyMemory:                []byte("ok case"),
			validatedMemory:            nil,
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

			var (
				objectRef = server.RandomGlobalWithPulse()
			)

			{ // send object state to server
				builder := server.StateReportBuilder().Object(objectRef).Ready().
					DirtyMemory(test.dirtyMemory)

				if test.validatedMemory != nil {
					builder = builder.ValidatedMemory(test.validatedMemory)
				} else {
					builder = builder.NoValidated()
				}

				server.IncrementPulseAndWaitIdle(ctx)
				server.SendPayload(ctx, builder.ReportPtr())
				server.WaitActiveThenIdleConveyor()
			}

			outgoingRef := server.BuildRandomOutgoingWithPulse()

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

			{
				typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
					assert.Equal(t, objectRef, res.Callee.GetValue())
					assert.Equal(t, test.callResult, res.ReturnArguments.GetBytes())
					return false // no resend msg
				})

				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = test.callFlags
				pl.Callee.Set(objectRef)
				pl.CallOutgoing.Set(outgoingRef)

				key := pl.CallOutgoing.GetValue()

				runnerMock.AddExecutionClassify(key, contract.MethodIsolation{
					Interference: test.callFlags.GetInterference(),
					State:        test.callFlags.GetState(),
				}, nil)

				// if we use forbidden isolation, then execute should stop before Start happen
				if !test.expectedUnImplementedError {
					result := requestresult.New(test.callResult, outgoingRef)
					desc := descriptor.NewObject(
						reference.Global{}, reference.Local{}, server.RandomGlobalWithPulse(), []byte(""), false,
					)
					result.SetAmend(desc, []byte("new stuff"))

					runnerMock.AddExecutionMock(key).AddStart(nil, &execution.Update{
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

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	{
		server.ReplaceRunner(runnerMock)
		server.Init(ctx)
	}

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	var (
		class     = server.RandomGlobalWithPulse()
		objectRef = server.RandomGlobalWithPulse()
		p1        = server.GetPulseNumber()
		isolation = contract.MethodIsolation{
			Interference: isolation.CallIntolerable,
			State:        isolation.CallValidated,
		}
	)

	server.IncrementPulseAndWaitIdle(ctx)
	outgoing := server.BuildRandomOutgoingWithPulse()

	{
		runnerMock.AddExecutionClassify(outgoing, isolation, nil)
		requestResult := requestresult.New([]byte("call result"), server.RandomGlobalWithPulse())
		newObjDescriptor := descriptor.NewObject(
			reference.Global{}, reference.Local{}, class, []byte(""), false,
		)
		requestResult.SetAmend(newObjDescriptor, []byte(changedObjectMem))

		objectExecutionMock := runnerMock.AddExecutionMock(outgoing)
		objectExecutionMock.AddStart(func(ctx execution.Context) {
			assert.Equal(t, objectRef, ctx.Request.Callee.GetValue())
			assert.Equal(t, []byte(origObjectMem), ctx.ObjectDescriptor.Memory())
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}
	{
		typedChecker.VStateRequest.Set(func(req *rms.VStateRequest) bool {
			assert.Equal(t, p1, req.AsOf)
			assert.Equal(t, objectRef, req.Object.GetValue())

			flags := rms.RequestLatestDirtyState | rms.RequestLatestValidatedState |
				rms.RequestOrderedQueue | rms.RequestUnorderedQueue
			assert.Equal(t, flags, req.RequestedContent)

			report := utils.NewStateReportBuilder().Pulse(req.AsOf).Object(objectRef).Ready().
				Memory([]byte(origObjectMem)).Report()
			server.SendPayload(ctx, &report)
			return false // no resend msg
		})

		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, objectRef, res.Callee.GetValue())
			assert.Equal(t, outgoing, res.CallOutgoing.GetValue())
			contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
			require.NoError(t, sysErr)
			assert.Equal(t, "intolerable call trying to change object state", contractErr.Error())
			return false // no resend msg
		})
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, rms.StateStatusReady, report.Status)
			assert.True(t, report.DelegationSpec.IsZero())
			assert.Equal(t, int32(0), report.UnorderedPendingCount)
			assert.Equal(t, int32(0), report.OrderedPendingCount)
			assert.NotNil(t, report.ProvidedContent)
			assert.Equal(t, []byte(origObjectMem), report.ProvidedContent.LatestDirtyState.Memory.GetBytes())
			return false
		})
	}
	{
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = rms.BuildCallFlags(isolation.Interference, isolation.State)
		pl.Callee.Set(objectRef)
		pl.CallSiteMethod = "SomeMethod"
		pl.CallOutgoing.Set(outgoing)

		server.SendPayload(ctx, pl)
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

	server.IncrementPulseAndWaitIdle(ctx)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 1, typedChecker.VStateRequest.Count())
	assert.Equal(t, 1, typedChecker.VCallResult.Count())
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

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)

	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		class        = server.RandomGlobalWithPulse()
		initialState = []byte("initial state")
		newState     = []byte("updated state")
		dStateID     = reference.NewRecordOf(objectGlobal, server.RandomLocalWithPulse())
	)

	// add typedChecker mock
	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	{
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			assert.Equal(t, newState, report.ProvidedContent.LatestDirtyState.Memory.GetBytes())
			assert.Equal(t, newState, report.ProvidedContent.LatestValidatedState.Memory.GetBytes())
			return false
		})
		typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
			switch string(result.ReturnArguments.GetBytes()) {
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
		report := server.StateReportBuilder().Object(objectGlobal).Ready().Memory(initialState).Report()
		// need for correct handle state report (should from prev pulse)
		server.IncrementPulse(ctx)

		waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, &report)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, waitReport)
	}

	outgoingValidated1 := server.BuildRandomOutgoingWithPulse()
	outgoingValidated2 := server.BuildRandomOutgoingWithPulse()
	outgoingDirty1 := server.BuildRandomOutgoingWithPulse()
	outgoingDirty2 := server.BuildRandomOutgoingWithPulse()
	outgoingChange := server.BuildRandomOutgoingWithPulse()

	// add ExecutionMock to runnerMock
	{
		objectDescriptor := descriptor.NewObject(
			objectGlobal,
			dStateID.GetLocal(),
			class,
			newState,
			false,
		)
		requestResult := requestresult.New([]byte("done"), server.RandomGlobalWithPulse())
		requestResult.SetAmend(objectDescriptor, newState)

		runnerMock.AddExecutionMock(outgoingChange).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, initialState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
		runnerMock.AddExecutionClassify(outgoingChange, tolerableFlags(), nil)

		runnerMock.AddExecutionMock(outgoingValidated1).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, isolation.CallValidated, ctx.Isolation.State)
				assert.Equal(t, initialState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("get validated info 1"), server.RandomGlobalWithPulse()),
			},
		)
		runnerMock.AddExecutionMock(outgoingDirty1).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, isolation.CallDirty, ctx.Isolation.State)
				assert.Equal(t, initialState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("get dirty info 1"), server.RandomGlobalWithPulse()),
			},
		)
		runnerMock.AddExecutionMock(outgoingValidated2).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, isolation.CallValidated, ctx.Isolation.State)
				assert.Equal(t, initialState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("get validated info 2"), server.RandomGlobalWithPulse()),
			},
		)
		runnerMock.AddExecutionMock(outgoingDirty2).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, isolation.CallDirty, ctx.Isolation.State)
				assert.Equal(t, newState, ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("get dirty info 2"), server.RandomGlobalWithPulse()),
			},
		)
		runnerMock.AddExecutionClassify(outgoingValidated1, intolerableFlags(), nil)
		runnerMock.AddExecutionClassify(outgoingValidated2, intolerableFlags(), nil)
		runnerMock.AddExecutionClassify(outgoingDirty1, contract.MethodIsolation{Interference: isolation.CallIntolerable, State: isolation.CallDirty}, nil)
		runnerMock.AddExecutionClassify(outgoingDirty2, contract.MethodIsolation{Interference: isolation.CallIntolerable, State: isolation.CallDirty}, nil)
	}

	// get object state
	{
		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated)
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "GetValidatedMethod1"
		pl.CallOutgoing.Set(outgoingValidated1)

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

		executeDone = server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		pl = utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "GetDirtyMethod1"
		pl.CallOutgoing.Set(outgoingDirty1)

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	}

	// change object state
	{
		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "ChangeMethod"
		pl.CallOutgoing.Set(outgoingChange)

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	}

	// check object state
	{
		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

		pl := utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated)
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "GetValidatedMethod2"
		pl.CallOutgoing.Set(outgoingValidated2)

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)

		executeDone = server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

		pl = utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "GetDirtyMethod2"
		pl.CallOutgoing.Set(outgoingDirty2)

		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	}

	// increment pulse and check VStateReport
	{
		server.IncrementPulse(ctx)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
		// wait for all VCallResults
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

		assert.Equal(t, 1, typedChecker.VStateReport.Count())
		assert.Equal(t, 5, typedChecker.VCallResult.Count())
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
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectGlobal, prevPulse)
	}

	var (
		firstOutgoing  = server.BuildRandomOutgoingWithPulse()
		secondOutgoing = server.BuildRandomOutgoingWithPulse()
	)

	synchronizeExecution := synchronization.NewPoint(1)
	defer synchronizeExecution.Done()

	// add ExecutionMock to runnerMock
	{
		runnerMock.AddExecutionMock(firstOutgoing).AddStart(
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

		runnerMock.AddExecutionMock(secondOutgoing).AddStart(
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

		runnerMock.AddExecutionClassify(firstOutgoing, intolerableFlags(), nil)
		runnerMock.AddExecutionClassify(secondOutgoing, intolerableFlags(), nil)
	}

	// add typedChecker mock
	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	{
		typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
			switch result.CallOutgoing.GetValue() {
			case firstOutgoing:
				assert.Equal(t, []byte("first result"), result.ReturnArguments.GetBytes())
			case secondOutgoing:
				assert.Equal(t, []byte("second result"), result.ReturnArguments.GetBytes())
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
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "GetMethod"
		pl.CallOutgoing.Set(firstOutgoing)

		server.SendPayload(ctx, pl)
		// wait for the first
		commontestutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
		// send second
		pl = utils.GenerateVCallRequestMethod(server)
		pl.Callee.Set(objectGlobal)
		pl.CallSiteMethod = "GetMethod"
		pl.CallOutgoing.Set(secondOutgoing)

		server.SendPayload(ctx, pl)
	}

	// wait for all VCallResults
	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}
