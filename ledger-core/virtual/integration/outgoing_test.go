// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

// A.Foo calls B.Bar, pulse changes, outgoing has proper token
//
// VCallRequest [A.Foo]
// -> ExecutionStart
// -> -> change pulse to secondPulse
// -> VStateReport [A]
// -> VDelegatedCallRequest [A]
// -> VCallRequest [B.Bar] + first token
// -> -> change pulse to thirdPulse
// -> NO VStateReport
// -> VDelegatedCallRequest [A] + first token
// -> VCallRequest [B.Bar] + second token
// -> VCallResult [A.Foo] + second token
// -> VDelegatedRequestFinished [A] + second token
func TestVirtual_CallMethodOutgoing_WithTwicePulseChange(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5141")

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

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		prevPulse     = server.GetPulse().PulseNumber
		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())
	)

	server.IncrementPulseAndWaitIdle(ctx)

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal, prevPulse)

	var (
		barIsolation = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}
		fooIsolation = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}

		outgoingCallRef = reference.NewRecordOf(objectAGlobal, server.RandomLocalWithPulse())

		classB        = gen.UniqueGlobalRef()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		firstPulse  = server.GetPulse().PulseNumber
		secondPulse pulse.Number
		thirdPulse  pulse.Number

		firstApprover  = gen.UniqueGlobalRef()
		secondApprover = gen.UniqueGlobalRef()

		firstExpectedToken = payload.CallDelegationToken{
			TokenTypeAndFlags: payload.DelegationTokenTypeCall,
			PulseNumber:       0, // fill later
			Callee:            objectAGlobal,
			Outgoing:          outgoingCallRef,
			DelegateTo:        server.JetCoordinatorMock.Me(),
			Approver:          firstApprover,
		}
		secondExpectedToken = payload.CallDelegationToken{
			TokenTypeAndFlags: payload.DelegationTokenTypeCall,
			PulseNumber:       0, // fill later
			Callee:            objectAGlobal,
			Outgoing:          outgoingCallRef,
			DelegateTo:        server.JetCoordinatorMock.Me(),
			Approver:          secondApprover,
		}

		expectedVCallRequest = payload.VCallRequest{
			CallType:         payload.CTMethod,
			CallFlags:        payload.BuildCallFlags(barIsolation.Interference, barIsolation.State),
			Callee:           objectBGlobal,
			Caller:           objectAGlobal,
			CallSequence:     1,
			CallSiteMethod:   "Bar",
			CallReason:       outgoingCallRef,
			CallRequestFlags: payload.BuildCallRequestFlags(payload.SendResultDefault, payload.RepeatedCall),
			Arguments:        []byte("123"),
		}
	)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).
			CallMethod(objectBGlobal, classB, "Bar", []byte("123")).
			SetInterference(barIsolation.Interference).
			SetIsolation(barIsolation.State)

		runnerMock.AddExecutionMock("Foo").AddStart(func(_ execution.Context) {
			logger.Debug("ExecutionStart [A.Foo]")

			server.IncrementPulse(ctx)
			secondPulse = server.GetPulse().PulseNumber
			firstExpectedToken.PulseNumber = secondPulse
			server.WaitActiveThenIdleConveyor()

		}, &execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		}).AddContinue(func(result []byte) {
			logger.Debug("ExecutionContinue [A.Foo]")
			require.Equal(t, []byte("finish B.Bar"), result)
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		})

		runnerMock.AddExecutionClassify("Foo", fooIsolation, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			// check for pending counts must be in tests: call constructor/call terminal method
			assert.Equal(t, objectAGlobal, report.Object)
			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
			assert.Equal(t, objectAGlobal, request.Callee)

			token := payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				PulseNumber:       server.GetPulse().PulseNumber,
				Callee:            request.Callee,
				Outgoing:          request.CallOutgoing,
				DelegateTo:        server.JetCoordinatorMock.Me(),
			}

			msg := payload.VDelegatedCallResponse{
				Callee:                 request.Callee,
				CallIncoming:           request.CallIncoming,
				ResponseDelegationSpec: token,
			}

			switch typedChecker.VDelegatedCallRequest.CountBefore() {
			case 1:
				assert.Zero(t, request.DelegationSpec)
				msg.ResponseDelegationSpec.Approver = firstApprover
			case 2:
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				msg.ResponseDelegationSpec.Approver = secondApprover
			default:
				t.Fatal("unexpected")
			}

			server.SendPayload(ctx, &msg)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			assert.Equal(t, objectAGlobal, finished.Callee)
			assert.Equal(t, secondExpectedToken, finished.DelegationSpec)
			return false
		})
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, objectBGlobal, request.Callee)

			switch typedChecker.VCallRequest.CountBefore() {
			case 1:
				assert.Equal(t, secondPulse, request.CallOutgoing.GetLocal().Pulse()) // new pulse
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				assert.NotEqual(t, payload.RepeatedCall, request.CallRequestFlags.GetRepeatedCall())

				server.IncrementPulse(ctx)
				thirdPulse = server.GetPulse().PulseNumber
				secondExpectedToken.PulseNumber = thirdPulse
				expectedVCallRequest.DelegationSpec = secondExpectedToken
				server.WaitActiveThenIdleConveyor()

				// request will be sent in previous pulse
				// omit sending
			case 2:
				assert.Equal(t, secondPulse, request.CallOutgoing.GetLocal().Pulse()) // the same pulse

				// reRequest -> check all fields
				expectedVCallRequest.CallOutgoing = request.CallOutgoing
				assert.Equal(t, &expectedVCallRequest, request)

				server.SendPayload(ctx, &payload.VCallResult{
					CallType:        request.CallType,
					CallFlags:       request.CallFlags,
					CallAsOf:        request.CallAsOf,
					Caller:          request.Caller,
					Callee:          request.Callee,
					CallOutgoing:    request.CallOutgoing,
					ReturnArguments: []byte("finish B.Bar"),
				})
			default:
				t.Fatal("unexpected")
			}

			return false
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, objectAGlobal, res.Callee)
			assert.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
			assert.Equal(t, int(firstPulse), int(res.CallOutgoing.GetLocal().Pulse()))
			assert.Equal(t, secondExpectedToken, res.DelegationSpec)
			return false
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      payload.BuildCallFlags(fooIsolation.Interference, fooIsolation.State),
		Caller:         statemachine.APICaller,
		Callee:         objectAGlobal,
		CallSiteMethod: "Foo",
		CallOutgoing:   outgoingCallRef,
	}

	msg := server.WrapPayload(&pl).SetSender(statemachine.APICaller).Finalize()
	server.SendMessage(ctx, msg)

	// wait for all calls and SMs
	{
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	assert.Equal(t, 2, typedChecker.VCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 2, typedChecker.VDelegatedCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

	mc.Finish()
}

// A.New calls B.Bar, pulse changes, outgoing has proper token
//
// VCallRequest [A.New]
// -> ExecutionStart
// -> -> change pulse to secondPulse
// -> VStateReport [A]
// -> VDelegatedCallRequest [A]
// -> VCallRequest [B.Bar] + first token
// -> -> change pulse to thirdPulse
// -> NO VStateReport
// -> VDelegatedCallRequest [A] + first token
// -> VCallRequest [B.Bar] + second token
// -> VCallResult [A.New] + second token
// -> VDelegatedRequestFinished [A] + second token
func TestVirtual_CallConstructorOutgoing_WithTwicePulseChange(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5142")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		constructorIsolation = contract.ConstructorIsolation()

		barIsolation = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}

		classA    = server.RandomGlobalWithPulse()
		outgoing  = server.BuildRandomOutgoingWithPulse()
		objectRef = reference.NewSelf(outgoing.GetLocal())

		classB        = server.RandomGlobalWithPulse()
		objectBGlobal = server.RandomGlobalWithPulse()

		firstPulse  = server.GetPulse().PulseNumber
		secondPulse pulse.Number

		firstApprover  = gen.UniqueGlobalRef()
		secondApprover = gen.UniqueGlobalRef()

		firstExpectedToken, secondExpectedToken payload.CallDelegationToken

		expectedVCallRequest = payload.VCallRequest{
			CallType:         payload.CTMethod,
			CallFlags:        payload.BuildCallFlags(barIsolation.Interference, barIsolation.State),
			Callee:           objectBGlobal,
			Caller:           objectRef,
			CallSequence:     1,
			CallSiteMethod:   "Bar",
			CallReason:       outgoing,
			CallRequestFlags: payload.BuildCallRequestFlags(payload.SendResultDefault, payload.RepeatedCall),
			Arguments:        []byte("123"),
		}
	)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoing, objectRef).CallMethod(objectBGlobal, classB, "Bar", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), outgoing)
		objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))

		runnerMock.AddExecutionMock("New").AddStart(func(_ execution.Context) {
			server.IncrementPulse(ctx)
			secondPulse = server.GetPulse().PulseNumber
			server.WaitActiveThenIdleConveyor()
		}, &execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		}).AddContinue(func(result []byte) {
			require.Equal(t, []byte("finish B.Bar"), result)
		}, &execution.Update{
			Type:   execution.Done,
			Result: objectAResult,
		})
	}

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			// check for pending counts must be in tests: call constructor/call terminal method
			assert.Equal(t, objectRef, report.Object)
			assert.Equal(t, payload.Empty, report.Status)
			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
			assert.Equal(t, objectRef, request.Callee)
			assert.Equal(t, outgoing, request.CallOutgoing)

			msg := payload.VDelegatedCallResponse{
				Callee:       request.Callee,
				CallIncoming: request.CallIncoming,
			}

			switch typedChecker.VDelegatedCallRequest.CountBefore() {
			case 1:
				assert.Zero(t, request.DelegationSpec)

				firstExpectedToken = payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       server.GetPulse().PulseNumber,
					Callee:            request.Callee,
					Outgoing:          request.CallOutgoing,
					DelegateTo:        server.JetCoordinatorMock.Me(),
					Approver:          firstApprover,
				}
				msg.ResponseDelegationSpec = firstExpectedToken
			case 2:
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)

				secondExpectedToken = payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       server.GetPulse().PulseNumber,
					Callee:            request.Callee,
					Outgoing:          request.CallOutgoing,
					DelegateTo:        server.JetCoordinatorMock.Me(),
					Approver:          secondApprover,
				}
				msg.ResponseDelegationSpec = secondExpectedToken
				expectedVCallRequest.DelegationSpec = secondExpectedToken
			default:
				t.Fatal("unexpected")
			}

			server.SendPayload(ctx, &msg)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			assert.Equal(t, objectRef, finished.Callee)
			assert.Equal(t, secondExpectedToken, finished.DelegationSpec)
			assert.Equal(t, payload.CTConstructor, finished.CallType)
			assert.NotNil(t, finished.LatestState)
			return false
		})
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, objectBGlobal, request.Callee)

			switch typedChecker.VCallRequest.CountBefore() {
			case 1:
				assert.Equal(t, secondPulse, request.CallOutgoing.GetLocal().Pulse()) // new pulse
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				assert.NotEqual(t, payload.RepeatedCall, request.CallRequestFlags.GetRepeatedCall())

				server.IncrementPulseAndWaitIdle(ctx)
				// request will be sent in previous pulse
				// omit sending
			case 2:
				assert.Equal(t, secondPulse, request.CallOutgoing.GetLocal().Pulse()) // the same pulse

				// reRequest -> check all fields
				expectedVCallRequest.CallOutgoing = request.CallOutgoing
				assert.Equal(t, &expectedVCallRequest, request)

				server.SendPayload(ctx, &payload.VCallResult{
					CallType:        request.CallType,
					CallFlags:       request.CallFlags,
					CallAsOf:        request.CallAsOf,
					Caller:          request.Caller,
					Callee:          request.Callee,
					CallOutgoing:    request.CallOutgoing,
					ReturnArguments: []byte("finish B.Bar"),
				})
			default:
				t.Fatal("unexpected")
			}

			return false
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, objectRef, res.Callee)
			assert.Equal(t, []byte("finish A.New"), res.ReturnArguments)
			assert.Equal(t, int(firstPulse), int(res.CallOutgoing.GetLocal().Pulse()))
			assert.Equal(t, secondExpectedToken, res.DelegationSpec)
			return false
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(constructorIsolation.Interference, constructorIsolation.State),
		Caller:         statemachine.APICaller,
		Callee:         classA,
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
	}

	msg := server.WrapPayload(&pl).SetSender(statemachine.APICaller).Finalize()
	server.SendMessage(ctx, msg)

	// wait for all calls and SMs
	{
		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	assert.Equal(t, 2, typedChecker.VCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 2, typedChecker.VDelegatedCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

	mc.Finish()
}

func TestVirtual_CallContractOutgoingReturnsError(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4971")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	utils.AssertNotJumpToStep(t, server.Journal, "stepTakeLock")

	logger := inslogger.FromContext(ctx)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.Callee.String()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		prevPulse     = server.GetPulse().PulseNumber
		outgoingA     = gen.UniqueGlobalRefWithPulse(prevPulse)
		objectBGlobal = gen.UniqueGlobalRefWithPulse(prevPulse)
	)

	server.IncrementPulseAndWaitIdle(ctx)

	// create objects
	{
		Method_PrepareObject(ctx, server, payload.Ready, outgoingA, prevPulse)
		Method_PrepareObject(ctx, server, payload.Ready, objectBGlobal, prevPulse)
	}

	var (
		flags     = contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		callFlags = payload.BuildCallFlags(flags.Interference, flags.State)

		classB          = gen.UniqueGlobalRef()
		outgoingCallRef = reference.NewRecordOf(
			server.GlobalCaller(), server.RandomLocalWithPulse(),
		)
	)

	p := server.GetPulse().PulseNumber

	// add ExecutionMocks to runnerMock
	{
		builder := execution.NewRPCBuilder(outgoingCallRef, outgoingA)
		objectAExecutionMock := runnerMock.AddExecutionMock(outgoingA.String())
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [A.Foo]")
				require.Equal(t, server.GlobalCaller(), ctx.Request.Caller)
				require.Equal(t, outgoingA, ctx.Request.Callee)
				require.Equal(t, outgoingCallRef, ctx.Request.CallOutgoing)
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: builder.CallMethod(objectBGlobal, classB, "Bar", []byte("B")),
			},
		)

		objectAExecutionMock.AddContinue(
			func(result []byte) {
				logger.Debug("ExecutionContinue [A.Foo]")

				expectedError := throw.W(errors.New("some error"), "failed to execute request")
				serializedErr, err := foundation.MarshalMethodErrorResult(expectedError)
				require.NoError(t, err)
				require.NotNil(t, serializedErr)
				require.Equal(t, serializedErr, result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), outgoingA),
			},
		)

		someError := errors.New("some error")
		resultWithErr, err := foundation.MarshalMethodErrorResult(someError)
		if err != nil {
			panic(throw.W(err, "can't create error result"))
		}

		objectBExecutionMock := runnerMock.AddExecutionMock(objectBGlobal.String())

		objectBExecutionMock.AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [B.Bar]")
				require.Equal(t, objectBGlobal, ctx.Request.Callee)
				require.Equal(t, outgoingA, ctx.Request.Caller)
				require.Equal(t, []byte("B"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Error,
				Error:  someError,
				Result: requestresult.New(resultWithErr, objectBGlobal),
			},
		)

		objectBExecutionMock.AddAbort(func() {
			logger.Debug("aborted [B.Foo]")
		})

		runnerMock.AddExecutionClassify(outgoingA.String(), flags, nil)
		runnerMock.AddExecutionClassify(objectBGlobal.String(), flags, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, outgoingA, request.Caller)
			assert.Equal(t, payload.CTMethod, request.CallType)
			assert.Equal(t, outgoingCallRef, request.CallReason)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, p, request.CallOutgoing.GetLocal().Pulse())

			switch request.Callee {
			case objectBGlobal:
				require.Equal(t, []byte("B"), request.Arguments)
				require.Equal(t, uint32(1), request.CallSequence)
			default:
				t.Fatal("wrong Callee")
			}
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, payload.CTMethod, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee {
			case outgoingA:
				require.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
				require.Equal(t, server.GlobalCaller(), res.Caller)
				require.Equal(t, outgoingCallRef, res.CallOutgoing)
			case objectBGlobal:
				expectedError := throw.W(errors.New("some error"), "failed to execute request")
				contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
				require.NoError(t, sysErr)
				require.NotNil(t, contractErr)
				require.Equal(t, expectedError.Error(), contractErr.Error())

				require.Equal(t, outgoingA, res.Caller)
				require.Equal(t, p, res.CallOutgoing.GetLocal().Pulse())
			default:
				t.Fatal("wrong Callee")
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == outgoingA
		})
	}

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      callFlags,
		Caller:         server.GlobalCaller(),
		Callee:         outgoingA,
		CallSiteMethod: "Foo",
		CallOutgoing:   outgoingCallRef,
	}
	server.SendPayload(ctx, &pl)

	// wait for all calls and SMs
	commontestutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}
