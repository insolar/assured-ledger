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

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
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

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	var (
		prevPulse     = server.GetPulse().PulseNumber
		objectAGlobal = server.RandomGlobalWithPulse()
	)

	server.IncrementPulseAndWaitIdle(ctx)

	Method_PrepareObject(ctx, server, rms.StateStatusReady, objectAGlobal, prevPulse)

	var (
		barIsolation = contract.MethodIsolation{
			Interference: isolation.CallTolerable,
			State:        isolation.CallDirty,
		}
		fooIsolation = contract.MethodIsolation{
			Interference: isolation.CallTolerable,
			State:        isolation.CallDirty,
		}

		outgoingCallRef = reference.NewRecordOf(objectAGlobal, server.RandomLocalWithPulse())

		classB        = server.RandomGlobalWithPulse()
		objectBGlobal = server.RandomGlobalWithPulse()

		firstPulse  = server.GetPulse().PulseNumber
		secondPulse pulse.Number

		firstApprover  = server.RandomGlobalWithPulse()
		secondApprover = server.RandomGlobalWithPulse()

		firstExpectedToken = rms.CallDelegationToken{
			TokenTypeAndFlags: rms.DelegationTokenTypeCall,
			PulseNumber:       0, // fill later
			Callee:            rms.NewReference(objectAGlobal),
			Outgoing:          rms.NewReference(outgoingCallRef),
			DelegateTo:        rms.NewReference(server.JetCoordinatorMock.Me()),
			Approver:          rms.NewReference(firstApprover),
		}
		secondExpectedToken = rms.CallDelegationToken{
			TokenTypeAndFlags: rms.DelegationTokenTypeCall,
			PulseNumber:       0, // fill later
			Callee:            rms.NewReference(objectAGlobal),
			Outgoing:          rms.NewReference(outgoingCallRef),
			DelegateTo:        rms.NewReference(server.JetCoordinatorMock.Me()),
			Approver:          rms.NewReference(secondApprover),
		}

		expectedVCallRequest = rms.VCallRequest{
			CallType:         rms.CallTypeMethod,
			CallFlags:        rms.BuildCallFlags(barIsolation.Interference, barIsolation.State),
			Callee:           rms.NewReference(objectBGlobal),
			Caller:           rms.NewReference(objectAGlobal),
			CallSequence:     1,
			CallSiteMethod:   "Bar",
			CallRequestFlags: rms.BuildCallRequestFlags(rms.SendResultDefault, rms.RepeatedCall),
			Arguments:        rms.NewBytes([]byte("123")),
		}
	)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).
			CallMethod(objectBGlobal, classB, "Bar", []byte("123")).
			SetInterference(barIsolation.Interference).
			SetIsolation(barIsolation.State)

		runnerMock.AddExecutionMock(outgoingCallRef).AddStart(func(_ execution.Context) {
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
			assert.Equal(t, []byte("finish B.Bar"), result)
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestresult.New([]byte("finish A.Foo"), objectAGlobal),
		})

		runnerMock.AddExecutionClassify(outgoingCallRef, fooIsolation, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			// check for pending counts must be in tests: call constructor/call terminal method
			assert.Equal(t, objectAGlobal, report.Object.GetValue())
			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *rms.VDelegatedCallRequest) bool {
			assert.Equal(t, objectAGlobal, request.Callee.GetValue())

			token := rms.CallDelegationToken{
				TokenTypeAndFlags: rms.DelegationTokenTypeCall,
				PulseNumber:       server.GetPulse().PulseNumber,
				Callee:            request.Callee,
				Outgoing:          request.CallOutgoing,
				DelegateTo:        rms.NewReference(server.JetCoordinatorMock.Me()),
			}

			msg := rms.VDelegatedCallResponse{
				Callee:                 request.Callee,
				CallIncoming:           request.CallIncoming,
				ResponseDelegationSpec: token,
			}

			switch typedChecker.VDelegatedCallRequest.CountBefore() {
			case 1:
				assert.Zero(t, request.DelegationSpec)
				msg.ResponseDelegationSpec.Approver.Set(firstApprover)
			case 2:
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				secondExpectedToken.PulseNumber = server.GetPulse().PulseNumber
				msg.ResponseDelegationSpec.Approver.Set(secondApprover)
			default:
				t.Fatal("unexpected")
			}

			server.SendPayload(ctx, &msg)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *rms.VDelegatedRequestFinished) bool {
			assert.Equal(t, objectAGlobal, finished.Callee.GetValue())
			assert.Equal(t, secondExpectedToken, finished.DelegationSpec)
			return false
		})
		typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
			assert.Equal(t, objectBGlobal, request.Callee.GetValue())

			switch typedChecker.VCallRequest.CountBefore() {
			case 1:
				assert.Equal(t, secondPulse, request.CallOutgoing.GetPulseOfLocal()) // new pulse
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				assert.NotEqual(t, rms.RepeatedCall, request.CallRequestFlags.GetRepeatedCall())

				server.IncrementPulse(ctx)

			case 2:
				assert.Equal(t, secondPulse, request.CallOutgoing.GetPulseOfLocal()) // the same pulse

				// reRequest -> check all fields
				expectedVCallRequest.DelegationSpec = secondExpectedToken
				expectedVCallRequest.CallOutgoing = request.CallOutgoing
				utils.AssertVCallRequestEqual(t, &expectedVCallRequest, request)

				server.SendPayload(ctx, &rms.VCallResult{
					CallType:        request.CallType,
					CallFlags:       request.CallFlags,
					Caller:          request.Caller,
					Callee:          request.Callee,
					CallOutgoing:    request.CallOutgoing,
					CallIncoming:    rms.NewReference(server.RandomGlobalWithPulse()),
					ReturnArguments: rms.NewBytes([]byte("finish B.Bar")),
				})
			default:
				t.Fatal("unexpected")
			}

			return false
		})
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, objectAGlobal, res.Callee.GetValue())
			assert.Equal(t, []byte("finish A.Foo"), res.ReturnArguments.GetBytes())
			assert.Equal(t, int(firstPulse), int(res.CallOutgoing.GetPulseOfLocal()))
			assert.Equal(t, secondExpectedToken, res.DelegationSpec)
			return false
		})
	}

	pl := utils.GenerateVCallRequestMethod(server)
	pl.CallFlags = rms.BuildCallFlags(fooIsolation.Interference, fooIsolation.State)
	pl.Callee.Set(objectAGlobal)
	pl.CallSiteMethod = "Foo"
	pl.CallOutgoing.Set(outgoingCallRef)

	server.SendPayload(ctx, pl)

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

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	var (
		barIsolation = contract.MethodIsolation{
			Interference: isolation.CallTolerable,
			State:        isolation.CallDirty,
		}

		classB        = server.RandomGlobalWithPulse()
		objectBGlobal = server.RandomGlobalWithPulse()

		firstPulse  = server.GetPulse().PulseNumber
		secondPulse pulse.Number

		firstApprover  = server.RandomGlobalWithPulse()
		secondApprover = server.RandomGlobalWithPulse()

		firstExpectedToken, secondExpectedToken rms.CallDelegationToken

		plWrapper = utils.GenerateVCallRequestConstructor(server)
		pl        = plWrapper.Get()

		classA    = pl.Callee.GetValue()
		outgoing  = plWrapper.GetOutgoing()
		objectRef = plWrapper.GetObject()

		expectedVCallRequest = rms.VCallRequest{
			CallType:         rms.CallTypeMethod,
			CallFlags:        rms.BuildCallFlags(barIsolation.Interference, barIsolation.State),
			Callee:           rms.NewReference(objectBGlobal),
			Caller:           rms.NewReference(objectRef),
			CallSequence:     1,
			CallSiteMethod:   "Bar",
			CallRequestFlags: rms.BuildCallRequestFlags(rms.SendResultDefault, rms.RepeatedCall),
			Arguments:        rms.NewBytes([]byte("123")),
		}
	)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoing, objectRef).CallMethod(objectBGlobal, classB, "Bar", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), outgoing)
		objectAResult.SetActivate(classA, []byte("state A"))

		runnerMock.AddExecutionMock(outgoing).AddStart(func(_ execution.Context) {
			server.IncrementPulse(ctx)
			secondPulse = server.GetPulse().PulseNumber
			server.WaitActiveThenIdleConveyor()
		}, &execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		}).AddContinue(func(result []byte) {
			assert.Equal(t, []byte("finish B.Bar"), result)
		}, &execution.Update{
			Type:   execution.Done,
			Result: objectAResult,
		})
	}

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			// check for pending counts must be in tests: call constructor/call terminal method
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, rms.StateStatusEmpty, report.Status)
			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *rms.VDelegatedCallRequest) bool {
			assert.Equal(t, objectRef, request.Callee.GetValue())
			assert.Equal(t, outgoing, request.CallOutgoing.GetValue())

			msg := rms.VDelegatedCallResponse{
				Callee:       request.Callee,
				CallIncoming: request.CallIncoming,
			}

			switch typedChecker.VDelegatedCallRequest.CountBefore() {
			case 1:
				assert.Zero(t, request.DelegationSpec)

				firstExpectedToken = rms.CallDelegationToken{
					TokenTypeAndFlags: rms.DelegationTokenTypeCall,
					PulseNumber:       server.GetPulse().PulseNumber,
					Callee:            request.Callee,
					Outgoing:          request.CallOutgoing,
					DelegateTo:        rms.NewReference(server.JetCoordinatorMock.Me()),
					Approver:          rms.NewReference(firstApprover),
				}
				msg.ResponseDelegationSpec = firstExpectedToken
			case 2:
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)

				secondExpectedToken = rms.CallDelegationToken{
					TokenTypeAndFlags: rms.DelegationTokenTypeCall,
					PulseNumber:       server.GetPulse().PulseNumber,
					Callee:            request.Callee,
					Outgoing:          request.CallOutgoing,
					DelegateTo:        rms.NewReference(server.JetCoordinatorMock.Me()),
					Approver:          rms.NewReference(secondApprover),
				}
				msg.ResponseDelegationSpec = secondExpectedToken
				expectedVCallRequest.DelegationSpec = secondExpectedToken
			default:
				t.Fatal("unexpected")
			}

			server.SendPayload(ctx, &msg)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *rms.VDelegatedRequestFinished) bool {
			assert.Equal(t, objectRef, finished.Callee.GetValue())
			assert.Equal(t, secondExpectedToken, finished.DelegationSpec)
			assert.Equal(t, rms.CallTypeConstructor, finished.CallType)
			assert.NotNil(t, finished.LatestState)
			return false
		})
		typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
			assert.Equal(t, objectBGlobal, request.Callee.GetValue())

			switch typedChecker.VCallRequest.CountBefore() {
			case 1:
				assert.Equal(t, secondPulse, request.CallOutgoing.GetPulseOfLocal()) // new pulse
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				assert.NotEqual(t, rms.RepeatedCall, request.CallRequestFlags.GetRepeatedCall())

				server.IncrementPulseAndWaitIdle(ctx)
				// request will be sent in previous pulse
				// omit sending
			case 2:
				assert.Equal(t, secondPulse, request.CallOutgoing.GetPulseOfLocal()) // the same pulse

				// reRequest -> check all fields
				expectedVCallRequest.CallOutgoing = request.CallOutgoing
				utils.AssertVCallRequestEqual(t, &expectedVCallRequest, request)

				server.SendPayload(ctx, &rms.VCallResult{
					CallType:        request.CallType,
					CallFlags:       request.CallFlags,
					Caller:          request.Caller,
					Callee:          request.Callee,
					CallOutgoing:    request.CallOutgoing,
					CallIncoming:    rms.NewReference(server.RandomGlobalWithPulse()),
					ReturnArguments: rms.NewBytes([]byte("finish B.Bar")),
				})
			default:
				t.Fatal("unexpected")
			}

			return false
		})
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, objectRef, res.Callee.GetValue())
			assert.Equal(t, []byte("finish A.New"), res.ReturnArguments.GetBytes())
			assert.Equal(t, int(firstPulse), int(res.CallOutgoing.GetPulseOfLocal()))
			assert.Equal(t, secondExpectedToken, res.DelegationSpec)
			return false
		})
	}

	server.SendPayload(ctx, &pl)

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

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) interface{} {
		return execution.Request.Callee.GetValue()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	var (
		prevPulse = server.GetPulse().PulseNumber
		objectA   = server.RandomGlobalWithPulse()
		objectB   = server.RandomGlobalWithPulse()
	)

	server.IncrementPulseAndWaitIdle(ctx)

	// create objects
	{
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectA, prevPulse)
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectB, prevPulse)
	}

	var (
		flags     = contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty}
		callFlags = rms.BuildCallFlags(flags.Interference, flags.State)

		classB          = server.RandomGlobalWithPulse()
		outgoingCallRef = server.BuildRandomOutgoingWithPulse()
	)

	p := server.GetPulse().PulseNumber

	// add ExecutionMocks to runnerMock
	{
		builder := execution.NewRPCBuilder(outgoingCallRef, objectA)
		objectAExecutionMock := runnerMock.AddExecutionMock(objectA)
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [A.Foo]")
				assert.Equal(t, server.GlobalCaller(), ctx.Request.Caller.GetValue())
				assert.Equal(t, objectA, ctx.Request.Callee.GetValue())
				assert.Equal(t, outgoingCallRef, ctx.Request.CallOutgoing.GetValue())
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Outgoing: builder.CallMethod(objectB, classB, "Bar", []byte("B")),
			},
		)

		objectAExecutionMock.AddContinue(
			func(result []byte) {
				logger.Debug("ExecutionContinue [A.Foo]")

				expectedError := throw.W(errors.New("some error"), "failed to execute request")
				serializedErr, err := foundation.MarshalMethodErrorResult(expectedError)
				require.NoError(t, err)
				assert.Equal(t, serializedErr, result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.Foo"), objectA),
			},
		)

		someError := errors.New("some error")
		resultWithErr, err := foundation.MarshalMethodErrorResult(someError)
		if err != nil {
			panic(throw.W(err, "can't create error result"))
		}

		objectBExecutionMock := runnerMock.AddExecutionMock(objectB)

		objectBExecutionMock.AddStart(
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [B.Bar]")
				assert.Equal(t, objectB, ctx.Request.Callee.GetValue())
				assert.Equal(t, objectA, ctx.Request.Caller.GetValue())
				assert.Equal(t, []byte("B"), ctx.Request.Arguments.GetBytes())
			},
			&execution.Update{
				Type:   execution.Error,
				Error:  someError,
				Result: requestresult.New(resultWithErr, objectB),
			},
		)

		objectBExecutionMock.AddAbort(func() {
			logger.Debug("aborted [B.Foo]")
		})

		runnerMock.AddExecutionClassify(objectA, flags, nil)
		runnerMock.AddExecutionClassify(objectB, flags, nil)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
			assert.Equal(t, objectA, request.Caller.GetValue())
			assert.Equal(t, rms.CallTypeMethod, request.CallType)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, p, request.CallOutgoing.GetPulseOfLocal())

			switch request.Callee.GetValue() {
			case objectB:
				assert.Equal(t, []byte("B"), request.Arguments.GetBytes())
				assert.Equal(t, uint32(1), request.CallSequence)
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
			case objectB:
				expectedError := throw.W(errors.New("some error"), "failed to execute request")
				contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
				require.NoError(t, sysErr)
				require.NotNil(t, contractErr)
				assert.Equal(t, expectedError.Error(), contractErr.Error())

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
	pl.CallSiteMethod = "Foo"
	pl.CallOutgoing.Set(outgoingCallRef)

	server.SendPayload(ctx, pl)

	// wait for all calls and SMs
	commontestutils.WaitSignalsTimed(t, 20*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 20*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallRequest.Count())
	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}
