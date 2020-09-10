// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

var byteArguments = []byte("123")

func tolerableFlags() contract.MethodIsolation {
	return contract.MethodIsolation{
		Interference: isolation.CallTolerable,
		State:        isolation.CallDirty,
	}
}

func intolerableFlags() contract.MethodIsolation {
	return contract.MethodIsolation{
		Interference: isolation.CallIntolerable,
		State:        isolation.CallValidated,
	}
}

// for happy path tests
func assertVCallResult(t *testing.T,
	res *rms.VCallResult,
	objectCaller reference.Global,
	objectCallee reference.Global,
	flagsCaller contract.MethodIsolation,
	outgoing reference.Global) {

	require.Equal(t, rms.CallTypeMethod, res.CallType)
	require.Equal(t, objectCaller, res.Caller)
	require.Equal(t, objectCallee, res.Callee)
	require.Equal(t, outgoing.GetLocal().Pulse(), res.CallOutgoing.GetPulseOfLocal())
	assert.Equal(t, flagsCaller.Interference, res.CallFlags.GetInterference()) // copy from VCallRequest
	assert.Equal(t, flagsCaller.State, res.CallFlags.GetState())
}

// for happy path tests
func assertVCallRequest(t *testing.T,
	objectCaller reference.Global,
	objectCallee reference.Global,
	request *rms.VCallRequest,
	flagsCaller contract.MethodIsolation) {

	assert.Equal(t, objectCallee, request.Callee)
	assert.Equal(t, objectCaller, request.Caller)
	assert.Equal(t, rms.CallTypeMethod, request.CallType)
	assert.Equal(t, flagsCaller.Interference, request.CallFlags.GetInterference())
	assert.Equal(t, flagsCaller.State, request.CallFlags.GetState())
	assert.Equal(t, uint32(1), request.CallSequence)

}

func Test_NoDeadLock_WhenOutgoingComeToSameNode(t *testing.T) {
	table := []struct {
		name       string
		flagsA     contract.MethodIsolation
		flagsB     contract.MethodIsolation
		testCaseID string
	}{
		{
			name:       "ordered A.Foo calls ordered B.Bar",
			flagsA:     tolerableFlags(),
			flagsB:     tolerableFlags(),
			testCaseID: "C4959",
		}, {
			name:       "unordered A.Foo calls unordered B.Bar",
			flagsA:     intolerableFlags(),
			flagsB:     intolerableFlags(),
			testCaseID: "C4960",
		},
	}

	for _, test := range table {
		test := test
		t.Run(test.name, func(t *testing.T) {
			insrail.LogCase(t, test.testCaseID)
			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			server.SetMaxParallelism(1)
			defer server.Stop()
			logger := inslogger.FromContext(ctx)

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) interface{} {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				class         = server.RandomGlobalWithPulse()
				objectAGlobal = server.RandomGlobalWithPulse()
				objectBGlobal = server.RandomGlobalWithPulse()
				pulse         = server.GetPulse().PulseNumber
			)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			server.IncrementPulseAndWaitIdle(ctx)

			Method_PrepareObject(ctx, server, rms.StateStatusReady, objectAGlobal, pulse)
			Method_PrepareObject(ctx, server, rms.StateStatusReady, objectBGlobal, pulse)

			outgoingCallRef := server.RandomGlobalWithPulse()
			outgoingA := server.BuildRandomOutgoingWithPulse()

			// add mock
			{
				outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectBGlobal, class, "Bar", byteArguments)
				objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
				objectAExecutionMock.AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [A.Foo]")
						require.Equal(t, objectAGlobal, ctx.Request.Callee)
						require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
					},
					&execution.Update{
						Type:     execution.OutgoingCall,
						Error:    nil,
						Outgoing: outgoingCall,
					},
				)
				bBarResult := []byte("finish B.Bar")
				objectAExecutionMock.AddContinue(
					func(result []byte) {
						logger.Debug("ExecutionContinue [A.Foo]")
						require.Equal(t, bBarResult, result)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
					},
				)

				runnerMock.AddExecutionMock("Bar").AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [B.Bar]")
						require.Equal(t, objectBGlobal, ctx.Request.Callee)
						require.Equal(t, objectAGlobal, ctx.Request.Caller)
						require.Equal(t, byteArguments, ctx.Request.Arguments)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New(bBarResult, objectBGlobal),
					},
				)

				runnerMock.AddExecutionClassify("Foo", test.flagsA, nil)
				runnerMock.AddExecutionClassify("Bar", test.flagsB, nil)
			}

			// checks
			{
				typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
					assertVCallRequest(t, objectAGlobal, objectBGlobal, request, test.flagsA)
					return true // resend
				})

				typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
					switch res.Callee.GetValue() {
					case objectAGlobal:
						require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
						assertVCallResult(t, res, server.GlobalCaller(), objectAGlobal, test.flagsA, outgoingA)
					case objectBGlobal:
						require.Equal(t, []byte("finish B.Bar"), res.ReturnArguments)
						assertVCallResult(t, res, objectAGlobal, objectBGlobal, test.flagsB, outgoingA)

					default:
						t.Fatalf("wrong Callee")
					}
					// we should resend that message only if it's CallResult from B to A
					return res.Caller.GetValue() == objectAGlobal
				})
			}

			pl := utils.GenerateVCallRequestMethod(server)
			pl.CallFlags = rms.BuildCallFlags(test.flagsA.Interference, test.flagsA.State)
			pl.Callee.Set(objectAGlobal)
			pl.CallSiteMethod = "Foo"
			pl.CallOutgoing.Set(outgoingA)
			server.SendPayload(ctx, pl)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			mc.Finish()
		})
	}

}

func TestVirtual_CallContractFromContract(t *testing.T) {
	insrail.LogCase(t, "C5086")

	table := []struct {
		name   string
		flagsA contract.MethodIsolation
		flagsB contract.MethodIsolation
	}{
		{
			name:   "ordered A.Foo calls ordered B.Bar",
			flagsA: tolerableFlags(),
			flagsB: tolerableFlags(),
		}, {
			name:   "ordered A.Foo calls unordered B.Bar",
			flagsA: tolerableFlags(),
			flagsB: intolerableFlags(),
		},
		{
			name:   "unordered A.Foo calls unordered B.Bar",
			flagsA: intolerableFlags(),
			flagsB: intolerableFlags(),
		},
	}
	for _, test := range table {
		test := test
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)
			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()
			logger := inslogger.FromContext(ctx)

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) interface{} {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				pn      = server.GetPulse().PulseNumber
				objectA = server.RandomGlobalWithPulse()
				objectB = server.RandomGlobalWithPulse()
			)

			server.IncrementPulseAndWaitIdle(ctx)

			Method_PrepareObject(ctx, server, rms.StateStatusReady, objectA, pn)
			Method_PrepareObject(ctx, server, rms.StateStatusReady, objectB, pn)

			var (
				class     = server.RandomGlobalWithPulse()
				outgoingA = server.BuildRandomOutgoingWithPulse()
				incomingA = reference.NewRecordOf(objectA, outgoingA.GetLocal())
			)
			// add mock
			{
				outgoingCall := execution.NewRPCBuilder(incomingA, objectA).
					CallMethod(objectB, class, "Bar", byteArguments)
				objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
				objectAExecutionMock.AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [A.Foo]")
						require.Equal(t, objectA, ctx.Request.Callee)
						require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
					},
					&execution.Update{
						Type:     execution.OutgoingCall,
						Error:    nil,
						Outgoing: outgoingCall,
					},
				)
				objectAExecutionMock.AddContinue(
					func(result []byte) {
						logger.Debug("ExecutionContinue [A.Foo]")
						require.Equal(t, []byte("finish B.Bar"), result)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("finish A.Foo"), objectA),
					},
				)

				runnerMock.AddExecutionMock("Bar").AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [B.Bar]")
						require.Equal(t, objectB, ctx.Request.Callee)
						require.Equal(t, objectA, ctx.Request.Caller)
						require.Equal(t, byteArguments, ctx.Request.Arguments)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("finish B.Bar"), objectB),
					},
				)

				runnerMock.AddExecutionClassify("Foo", test.flagsA, nil)
				runnerMock.AddExecutionClassify("Bar", test.flagsB, nil)
			}

			// checks
			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			{
				typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
					assertVCallRequest(t, objectA, objectB, request, test.flagsA)
					assert.Equal(t, byteArguments, request.Arguments)
					assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetPulseOfLocal())
					return true // resend
				})

				typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
					switch res.Callee.GetValue() {
					case objectA:
						require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
						assertVCallResult(t, res, server.GlobalCaller(), objectA, test.flagsA, outgoingA)
					case objectB:
						require.Equal(t, []byte("finish B.Bar"), res.ReturnArguments)
						assertVCallResult(t, res, objectA, objectB, test.flagsA, outgoingA)

					default:
						t.Fatalf("wrong Callee")
					}
					// we should resend that message only if it's CallResult from B to A
					return res.Caller.GetValue() == objectA
				})
			}

			pl := utils.GenerateVCallRequestMethod(server)
			pl.CallFlags = rms.BuildCallFlags(test.flagsA.Interference, test.flagsA.State)
			pl.CallSiteMethod = "Foo"
			pl.Callee.Set(objectA)
			pl.CallOutgoing.Set(outgoingA)
			server.SendPayload(ctx, pl)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallRequest.Count())
			require.Equal(t, 2, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_CallOtherMethodInObject(t *testing.T) {
	insrail.LogCase(t, "C5116")

	table := []struct {
		name        string
		stateSender contract.MethodIsolation
	}{
		{
			name:        "ordered A.Foo calls unordered A.Bar",
			stateSender: tolerableFlags(),
		}, {
			name:        "unordered A.Foo calls unordered A.Bar",
			stateSender: intolerableFlags(),
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()
			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

			logger := inslogger.FromContext(ctx)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) interface{} {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			var (
				prevPulse     = server.GetPulse().PulseNumber
				objectAGlobal = server.RandomGlobalWithPulse()
			)

			server.IncrementPulseAndWaitIdle(ctx)

			Method_PrepareObject(ctx, server, rms.StateStatusReady, objectAGlobal, prevPulse)

			var (
				class     = server.RandomGlobalWithPulse()
				outgoingA = server.BuildRandomOutgoingWithPulse()
				incomingA = reference.NewRecordOf(objectAGlobal, outgoingA.GetLocal())
			)

			// add mok
			{
				outgoingCall := execution.NewRPCBuilder(incomingA, objectAGlobal).CallMethod(objectAGlobal, class, "Bar", byteArguments)
				objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
				objectAExecutionMock.AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [A.Foo]")
						require.Equal(t, objectAGlobal, ctx.Request.Callee)
						require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
					},
					&execution.Update{
						Type:     execution.OutgoingCall,
						Error:    nil,
						Outgoing: outgoingCall,
					},
				)
				objectAExecutionMock.AddContinue(
					func(result []byte) {
						logger.Debug("ExecutionContinue [A.Foo]")
						require.Equal(t, []byte("finish A.Bar"), result)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
					},
				)

				runnerMock.AddExecutionMock("Bar").AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [A.Bar]")
						require.Equal(t, objectAGlobal, ctx.Request.Callee)
						require.Equal(t, objectAGlobal, ctx.Request.Caller)
						require.Equal(t, byteArguments, ctx.Request.Arguments)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("finish A.Bar"), objectAGlobal),
					},
				)

				runnerMock.AddExecutionClassify("Foo", test.stateSender, nil)
				runnerMock.AddExecutionClassify("Bar", intolerableFlags(), nil)
			}

			// add typedChecker
			{
				typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
					assertVCallRequest(t, objectAGlobal, objectAGlobal, request, test.stateSender)
					assert.Equal(t, byteArguments, request.Arguments)
					assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetPulseOfLocal())
					return true // resend
				})

				typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
					switch res.Caller.GetValue() {
					case objectAGlobal:
						require.Equal(t, []byte("finish A.Bar"), res.ReturnArguments)
						assertVCallResult(t, res, objectAGlobal, objectAGlobal, test.stateSender, outgoingA)
					default:
						require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
						assertVCallResult(t, res, server.GlobalCaller(), objectAGlobal, test.stateSender, outgoingA)
					}
					// we should resend that message only if it's CallResult from A to A
					return res.Caller.GetValue() == objectAGlobal
				})
			}

			pl := utils.GenerateVCallRequestMethod(server)
			pl.CallFlags = rms.BuildCallFlags(test.stateSender.Interference, test.stateSender.State)
			pl.Callee.Set(objectAGlobal)
			pl.CallSiteMethod = "Foo"
			pl.CallOutgoing.Set(outgoingA)
			server.SendPayload(ctx, pl)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallRequest.Count())
			require.Equal(t, 2, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_CallMethodFromConstructor(t *testing.T) {
	insrail.LogCase(t, "C5091")

	table := []struct {
		name   string
		stateB contract.MethodIsolation
	}{
		{
			name:   "A.New calls ordered B.Foo",
			stateB: tolerableFlags(),
		}, {
			name:   "A.New calls unordered B.Foo",
			stateB: intolerableFlags(),
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			logger := inslogger.FromContext(ctx)

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) interface{} {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			var (
				prevPulse     = server.GetPulse().PulseNumber
				objectBGlobal = server.RandomGlobalWithPulse()
			)

			server.IncrementPulseAndWaitIdle(ctx)

			Method_PrepareObject(ctx, server, rms.StateStatusReady, objectBGlobal, prevPulse)

			var (
				callFlags = tolerableFlags()

				classA        = server.RandomGlobalWithPulse()
				outgoingA     = server.BuildRandomOutgoingWithPulse()
				objectAGlobal = reference.NewSelf(outgoingA.GetLocal())
				incomingA     = reference.NewRecordOf(objectAGlobal, outgoingA.GetLocal())

				classB = server.RandomGlobalWithPulse()
			)
			// add ExecutionMocks to runnerMock
			{
				outgoingCall := execution.NewRPCBuilder(incomingA, objectAGlobal).CallMethod(objectBGlobal, classB, "Foo", byteArguments)
				objectAResult := requestresult.New([]byte("finish A.New"), objectAGlobal)
				objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))
				objectAExecutionMock := runnerMock.AddExecutionMock("New")
				objectAExecutionMock.AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [A.New]")
						require.Equal(t, classA, ctx.Request.Callee)
						require.Equal(t, outgoingA, ctx.Request.CallOutgoing)
					},
					&execution.Update{
						Type:     execution.OutgoingCall,
						Error:    nil,
						Outgoing: outgoingCall,
					},
				)
				objectAExecutionMock.AddContinue(
					func(result []byte) {
						logger.Debug("ExecutionContinue [A.New]")
						require.Equal(t, []byte("finish B.Foo"), result)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: objectAResult,
					},
				)

				runnerMock.AddExecutionMock("Foo").AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [B.Foo]")
						require.Equal(t, objectBGlobal, ctx.Request.Callee)
						require.Equal(t, objectAGlobal, ctx.Request.Caller)
						require.Equal(t, byteArguments, ctx.Request.Arguments)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("finish B.Foo"), objectBGlobal),
					},
				)

				runnerMock.AddExecutionClassify("Foo", test.stateB, nil)
			}

			// add checks to typedChecker
			{
				typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
					assertVCallRequest(t, objectAGlobal, objectBGlobal, request, callFlags)
					assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetPulseOfLocal())
					return true // resend
				})
				typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
					assert.Equal(t, callFlags.State, res.CallFlags.GetState())
					assert.Equal(t, callFlags.Interference, res.CallFlags.GetInterference())

					switch res.Callee.GetValue() {
					case objectAGlobal:
						require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
						require.Equal(t, rms.CallTypeConstructor, res.CallType)
						require.Equal(t, server.GlobalCaller(), res.Caller)
						require.Equal(t, outgoingA, res.CallOutgoing)
					case objectBGlobal:
						require.Equal(t, []byte("finish B.Foo"), res.ReturnArguments)
						require.Equal(t, rms.CallTypeMethod, res.CallType)
						require.Equal(t, objectAGlobal, res.Caller)
						require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.GetPulseOfLocal())

					default:
						t.Fatalf("wrong Callee")
					}
					// we should resend that message only if it's CallResult from B to A
					return res.Caller.GetValue() == objectAGlobal
				})
			}

			pl := utils.GenerateVCallRequestConstructor(server)
			pl.Callee.Set(classA)
			pl.CallOutgoing.Set(outgoingA)
			server.SendPayload(ctx, pl)

			// wait for all calls and SMs
			commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallRequest.Count())
			require.Equal(t, 2, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_CallContractFromContract_RetryLimit(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5320")

	countChangePulse := execute.MaxOutgoingSendCount

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
		// Pass all errors, except for (*SMExecute).stepSendOutgoing
		return !strings.Contains(s, "outgoing retries limit")
	})

	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		object     = server.RandomGlobalWithPulse()
		pulse      = server.GetPulse().PulseNumber
		tokenValue rms.CallDelegationToken
	)

	executeStopped := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	foundError := server.Journal.Wait(func(event debuglogger.UpdateEvent) bool {
		if event.Data.Error != nil {
			return strings.Contains(event.Data.Error.Error(), "outgoing retries limit")
		}
		return false
	})

	server.IncrementPulseAndWaitIdle(ctx)

	Method_PrepareObject(ctx, server, rms.StateStatusReady, object, pulse)

	pl := utils.GenerateVCallRequestMethod(server)
	pl.CallFlags = rms.BuildCallFlags(tolerableFlags().Interference, tolerableFlags().State)
	pl.Callee.Set(object)

	// add ExecutionMocks to runnerMock
	{
		key := pl.CallOutgoing.GetValue()
		runnerMock.AddExecutionClassify(key, tolerableFlags(), nil)

		builder := execution.NewRPCBuilder(pl.CallOutgoing.GetValue(), pl.Callee.GetValue())
		callMethod := builder.CallMethod(server.RandomGlobalWithPulse(), reference.Global{}, "Method", pl.Arguments.GetBytes())

		objectExecutionMock := runnerMock.AddExecutionMock(key)
		objectExecutionMock.AddStart(nil, &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: callMethod,
		})
	}

	point := synchronization.NewPoint(1)
	defer point.Done()

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool { return false })

		typedChecker.VDelegatedCallRequest.Set(func(request *rms.VDelegatedCallRequest) bool {
			require.Equal(t, object, request.Callee)
			newPulse := server.GetPulse().PulseNumber
			approver := server.RandomGlobalWithPulse()

			tokenValue = rms.CallDelegationToken{
				TokenTypeAndFlags: rms.DelegationTokenTypeCall,
				PulseNumber:       newPulse,
				Callee:            request.Callee,
				Outgoing:          request.CallOutgoing,
				DelegateTo:        rms.NewReference(server.JetCoordinatorMock.Me()),
				Approver:          rms.NewReference(approver),
			}
			msg := rms.VDelegatedCallResponse{
				Callee:                 request.Callee,
				CallIncoming:           request.CallIncoming,
				ResponseDelegationSpec: tokenValue,
			}

			server.SendPayload(ctx, &msg)
			return false
		})

		typedChecker.VCallRequest.Set(func(finished *rms.VCallRequest) bool {
			point.Synchronize()
			return false
		})

		typedChecker.VDelegatedRequestFinished.Set(func(finished *rms.VDelegatedRequestFinished) bool { return false })
	}

	server.SendPayload(ctx, pl)

	for i := 0; i < countChangePulse; i++ {
		commontestutils.WaitSignalsTimed(t, 10*time.Second, point.Wait())
		server.IncrementPulseAndWaitIdle(ctx)
		point.WakeUp()
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeStopped, foundError)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, countChangePulse, typedChecker.VCallRequest.Count())
	require.Equal(t, countChangePulse, typedChecker.VDelegatedCallRequest.Count())
	require.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

	mc.Finish()

}
