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
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

var byteArguments = []byte("123")

func tolerableFlags() contract.MethodIsolation {
	return contract.MethodIsolation{
		Interference: contract.CallTolerable,
		State:        contract.CallDirty,
	}
}

func intolerableFlags() contract.MethodIsolation {
	return contract.MethodIsolation{
		Interference: contract.CallIntolerable,
		State:        contract.CallValidated,
	}
}

// for happy path tests
func assertVCallResult(t *testing.T,
	res *payload.VCallResult,
	objectCaller reference.Global,
	objectCallee reference.Global,
	flagsCaller contract.MethodIsolation,
	outgoing reference.Global) {

	require.Equal(t, payload.CTMethod, res.CallType)
	require.Equal(t, objectCaller, res.Caller)
	require.Equal(t, objectCallee, res.Callee)
	require.Equal(t, outgoing.GetLocal().Pulse(), res.CallOutgoing.GetLocal().Pulse())
	assert.Equal(t, flagsCaller.Interference, res.CallFlags.GetInterference()) // copy from VCallRequest
	assert.Equal(t, flagsCaller.State, res.CallFlags.GetState())
}

// for happy path tests
func assertVCallRequest(t *testing.T,
	objectCaller reference.Global,
	objectCallee reference.Global,
	request *payload.VCallRequest,
	flagsCaller contract.MethodIsolation) {

	assert.Equal(t, objectCallee, request.Callee)
	assert.Equal(t, objectCaller, request.Caller)
	assert.Equal(t, payload.CTMethod, request.CallType)
	assert.Equal(t, flagsCaller.Interference, request.CallFlags.GetInterference())
	assert.Equal(t, flagsCaller.State, request.CallFlags.GetState())
	assert.Equal(t, uint32(1), request.CallSequence)

}

func TestVirtual_CallContractFromContract(t *testing.T) {
	t.Log("C5086")
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

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()
			logger := inslogger.FromContext(ctx)

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)
			server.IncrementPulseAndWaitIdle(ctx)

			var (
				class = gen.UniqueGlobalRef()

				outgoingA       = server.BuildRandomOutgoingWithPulse()
				objectAGlobal   = gen.UniqueGlobalRef()
				outgoingCallRef = gen.UniqueGlobalRef()
				objectBGlobal   = reference.NewSelf(server.RandomLocalWithPulse())
			)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)
			Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

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
				objectAExecutionMock.AddContinue(
					func(result []byte) {
						logger.Debug("ExecutionContinue [A.Foo]")
						require.Equal(t, []byte("finish B.Bar"), result)
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
						Result: requestresult.New([]byte("finish B.Bar"), objectBGlobal),
					},
				)

				runnerMock.AddExecutionClassify("Foo", test.flagsA, nil)
				runnerMock.AddExecutionClassify("Bar", test.flagsB, nil)
			}

			// checks
			{
				typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
					assertVCallRequest(t, objectAGlobal, objectBGlobal, request, test.flagsA)
					assert.Equal(t, byteArguments, request.Arguments)
					assert.Equal(t, outgoingCallRef, request.CallReason)
					assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetLocal().Pulse())
					return true // resend
				})

				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					switch res.Callee {
					case objectAGlobal:
						require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
						assertVCallResult(t, res, server.GlobalCaller(), objectAGlobal, test.flagsA, outgoingA)
					case objectBGlobal:
						require.Equal(t, []byte("finish B.Bar"), res.ReturnArguments)
						assertVCallResult(t, res, objectAGlobal, objectBGlobal, test.flagsA, outgoingA)

					default:
						t.Fatalf("wrong Callee")
					}
					// we should resend that message only if it's CallResult from B to A
					return res.Caller == objectAGlobal
				})
			}

			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(test.flagsA.Interference, test.flagsA.State),
				Caller:              server.GlobalCaller(),
				Callee:              objectAGlobal,
				CallSiteDeclaration: class,
				CallSiteMethod:      "Foo",
				CallOutgoing:        outgoingA,
			}

			server.SendPayload(ctx, &pl)

			testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallRequest.Count())
			require.Equal(t, 2, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_CallOtherMethodInObject(t *testing.T) {
	t.Log("C5116")
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

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()
			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

			logger := inslogger.FromContext(ctx)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)
			server.IncrementPulseAndWaitIdle(ctx)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			var (
				class           = gen.UniqueGlobalRef()
				outgoingA       = server.BuildRandomOutgoingWithPulse()
				objectAGlobal   = gen.UniqueGlobalRef()
				outgoingCallRef = gen.UniqueGlobalRef()
			)

			Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

			// add mok
			{
				outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectAGlobal, class, "Bar", byteArguments)
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
				typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
					assertVCallRequest(t, objectAGlobal, objectAGlobal, request, test.stateSender)
					assert.Equal(t, byteArguments, request.Arguments)
					assert.Equal(t, outgoingCallRef, request.CallReason)
					assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetLocal().Pulse())
					return true // resend
				})

				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					switch res.Caller {
					case objectAGlobal:
						require.Equal(t, []byte("finish A.Bar"), res.ReturnArguments)
						assertVCallResult(t, res, objectAGlobal, objectAGlobal, test.stateSender, outgoingA)
					default:
						require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
						assertVCallResult(t, res, server.GlobalCaller(), objectAGlobal, test.stateSender, outgoingA)
					}
					// we should resend that message only if it's CallResult from A to A
					return res.Caller == objectAGlobal
				})
			}

			pl := payload.VCallRequest{
				CallType:            payload.CTMethod,
				CallFlags:           payload.BuildCallFlags(test.stateSender.Interference, test.stateSender.State),
				Caller:              server.GlobalCaller(),
				Callee:              objectAGlobal,
				CallSiteDeclaration: class,
				CallSiteMethod:      "Foo",
				CallOutgoing:        outgoingA,
			}

			server.SendPayload(ctx, &pl)

			testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallRequest.Count())
			require.Equal(t, 2, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_CallMethodFromConstructor(t *testing.T) {
	t.Log("C5091")
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

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			logger := inslogger.FromContext(ctx)

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)
			server.IncrementPulseAndWaitIdle(ctx)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			var (
				callFlags = tolerableFlags()

				classA        = gen.UniqueGlobalRef()
				outgoingA     = server.BuildRandomOutgoingWithPulse()
				objectAGlobal = reference.NewSelf(outgoingA.GetLocal())

				classB        = gen.UniqueGlobalRef()
				objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

				outgoingCallRef = gen.UniqueGlobalRef()
			)

			Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal)

			// add ExecutionMocks to runnerMock
			{
				outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectBGlobal, classB, "Foo", byteArguments)
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
				typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
					assertVCallRequest(t, objectAGlobal, objectBGlobal, request, callFlags)
					assert.Equal(t, outgoingCallRef, request.CallReason)
					assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetLocal().Pulse())
					return true // resend
				})
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					assert.Equal(t, callFlags.State, res.CallFlags.GetState())
					assert.Equal(t, callFlags.Interference, res.CallFlags.GetInterference())

					switch res.Callee {
					case objectAGlobal:
						require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
						require.Equal(t, payload.CTConstructor, res.CallType)
						require.Equal(t, server.GlobalCaller(), res.Caller)
						require.Equal(t, outgoingA, res.CallOutgoing)
					case objectBGlobal:
						require.Equal(t, []byte("finish B.Foo"), res.ReturnArguments)
						require.Equal(t, payload.CTMethod, res.CallType)
						require.Equal(t, objectAGlobal, res.Caller)
						require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.GetLocal().Pulse())

					default:
						t.Fatalf("wrong Callee")
					}
					// we should resend that message only if it's CallResult from B to A
					return res.Caller == objectAGlobal
				})
			}

			pl := payload.VCallRequest{
				CallType:       payload.CTConstructor,
				CallFlags:      payload.BuildCallFlags(callFlags.Interference, callFlags.State),
				Caller:         server.GlobalCaller(),
				Callee:         classA,
				CallSiteMethod: "New",
				CallOutgoing:   outgoingA,
			}
			msg := server.WrapPayload(&pl).Finalize()
			server.SendMessage(ctx, msg)

			// wait for all calls and SMs
			testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallRequest.Count())
			require.Equal(t, 2, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_CallContractFromContract_RetryLimit(t *testing.T) {
	t.Log("C5320")

	countChangePulse := execute.MaxOutgoingSendCount

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
		// Pass all errors, except for (*SMExecute).stepSendOutgoing
		return !strings.Contains(s, "outgoing retries limit")
	})

	defer server.Stop()

	logger := inslogger.FromContext(ctx)

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
		outgoing = server.BuildRandomOutgoingWithPulse()
		object   = reference.NewSelf(server.RandomLocalWithPulse())

		tokenValue payload.CallDelegationToken
	)

	executeStopped := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	foundError := server.Journal.Wait(func(event debuglogger.UpdateEvent) bool {
		if event.Data.Error != nil {
			return strings.Contains(event.Data.Error.Error(), "outgoing retries limit")
		}
		return false
	})

	Method_PrepareObject(ctx, server, payload.Ready, object)

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      payload.BuildCallFlags(tolerableFlags().Interference, tolerableFlags().State),
		Callee:         object,
		CallSiteMethod: "SomeMethod",
		CallOutgoing:   outgoing,
	}

	// add ExecutionMocks to runnerMock
	{
		runnerMock.AddExecutionClassify("SomeMethod", tolerableFlags(), nil)

		objectExecutionMock := runnerMock.AddExecutionMock("SomeMethod")
		objectExecutionMock.AddStart(
			func(_ execution.Context) {
				logger.Debug("ExecutionStart [SomeMethod]")
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: execution.CallMethod{},
			},
		)
	}

	point := synchronization.NewPoint(1)

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool { return false })

		typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
			require.Equal(t, object, request.Callee)
			newPulse := server.GetPulse().PulseNumber
			approver := gen.UniqueGlobalRef()

			tokenValue = payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				PulseNumber:       newPulse,
				Callee:            request.Callee,
				Outgoing:          request.CallOutgoing,
				DelegateTo:        server.JetCoordinatorMock.Me(),
				Approver:          approver,
			}
			msg := payload.VDelegatedCallResponse{
				Callee:                 request.Callee,
				ResponseDelegationSpec: tokenValue,
			}

			server.SendPayload(ctx, &msg)
			return false
		})

		typedChecker.VCallRequest.Set(func(finished *payload.VCallRequest) bool {
			point.Synchronize()
			return false
		})

	}

	server.WaitIdleConveyor()
	server.SendPayload(ctx, &pl)

	for i := 0; i < countChangePulse; i++ {
		testutils.WaitSignalsTimed(t, 10*time.Second, point.Wait())
		server.IncrementPulseAndWaitIdle(ctx)
		point.WakeUp()
	}

	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone(), executeStopped, foundError)

	require.Equal(t, countChangePulse, typedChecker.VCallRequest.Count())

	mc.Finish()

}
