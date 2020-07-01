// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
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
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

// A.Foo calls B.Bar, pulse changes, outgoing has proper token
/*
VCallRequest [A.Foo]
ExecutionStart
change pulse -> secondPulse
VStateReport [A]
VDelegatedCallRequest [A]
VCallRequest [B.Bar] + first token
change pulse -> third pulse
NO VStateReport
VDelegatedCallRequest [A] + first token
VCallRequest [B.Bar] + second token
VCallResult [A.Foo] + second token
VDelegatedRequestFinished [A] + second token
*/
func TestVirtual_CallMethodOutgoing_WithTwicePulseChange(t *testing.T) {
	t.Log("C5141")

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
		barIsolation = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}
		fooIsolation = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}

		callOutgoing    = server.RandomLocalWithPulse()
		objectAGlobal   = reference.NewSelf(server.RandomLocalWithPulse())
		outgoingCallRef = reference.NewRecordOf(objectAGlobal, callOutgoing)

		classB        = gen.UniqueGlobalRef()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		firstPulse  = server.GetPulse().PulseNumber
		secondPulse pulse.Number
		thirdPulse  pulse.Number

		firstApprover  = gen.UniqueGlobalRef()
		secondApprover = gen.UniqueGlobalRef()

		firstVCallRequest, firstVDelegatedCallRequest = true, true

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

	Method_PrepareObject(ctx, server, payload.Ready, objectAGlobal)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).
			CallMethod(objectBGlobal, classB, "Bar", []byte("123")).
			SetInterference(barIsolation.Interference).
			SetIsolation(barIsolation.State)
		objectAExecutionMock := runnerMock.AddExecutionMock("Foo")
		objectAExecutionMock.AddStart(
			func(_ execution.Context) {
				logger.Debug("ExecutionStart [A.Foo]")
				server.IncrementPulseAndWaitIdle(ctx)
				secondPulse = server.GetPulse().PulseNumber
				firstExpectedToken.PulseNumber = secondPulse
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

			msg := payload.VDelegatedCallResponse{
				Callee: request.Callee,
				ResponseDelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       server.GetPulse().PulseNumber,
					Callee:            request.Callee,
					Outgoing:          request.CallOutgoing,
					DelegateTo:        server.JetCoordinatorMock.Me(),
				},
			}

			if firstVDelegatedCallRequest {
				assert.Zero(t, request.DelegationSpec)
				msg.ResponseDelegationSpec.Approver = firstApprover
				firstVDelegatedCallRequest = false
			} else {
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				msg.ResponseDelegationSpec.Approver = secondApprover
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

			if firstVCallRequest {
				assert.Equal(t, secondPulse, request.CallOutgoing.Pulse()) // new pulse
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				assert.NotEqual(t, payload.RepeatedCall, request.CallRequestFlags.GetRepeatedCall())

				server.IncrementPulseAndWaitIdle(ctx)
				thirdPulse = server.GetPulse().PulseNumber
				secondExpectedToken.PulseNumber = thirdPulse
				expectedVCallRequest.DelegationSpec = secondExpectedToken
				firstVCallRequest = false
				// request will be sent in previous pulse
				// omit sending
				return false
			}

			assert.Equal(t, secondPulse, request.CallOutgoing.Pulse()) // the same pulse

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

			return false
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, objectAGlobal, res.Callee)
			assert.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
			assert.Equal(t, int(firstPulse), int(res.CallOutgoing.Pulse()))
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
		CallOutgoing:   callOutgoing,
	}

	msg := server.WrapPayload(&pl).SetSender(statemachine.APICaller).Finalize()
	server.SendMessage(ctx, msg)

	// wait for all calls and SMs
	{
		testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
		testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	assert.Equal(t, 2, typedChecker.VCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 2, typedChecker.VDelegatedCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

	mc.Finish()
}

// A.New calls B.Bar, pulse changes, outgoing has proper token
/*
VCallRequest [A.New]
ExecutionStart
change pulse -> secondPulse
VStateReport [A]
VDelegatedCallRequest [A]
VCallRequest [B.Bar] + first token
change pulse -> third pulse
NO VStateReport
VDelegatedCallRequest [A] + first token
VCallRequest [B.New] + second token
VCallResult [A.New] + second token
VDelegatedRequestFinished [A] + second token
*/
func TestVirtual_CallConstructorOutgoing_WithTwicePulseChange(t *testing.T) {
	t.Log("C5142")

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

		classA          = gen.UniqueGlobalRef()
		callOutgoing    = server.RandomLocalWithPulse()
		objectAGlobal   = reference.NewSelf(callOutgoing)
		outgoingCallRef = reference.NewRecordOf(classA, callOutgoing)

		classB        = gen.UniqueGlobalRef()
		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		firstPulse  = server.GetPulse().PulseNumber
		secondPulse pulse.Number
		thirdPulse  pulse.Number

		firstApprover  = gen.UniqueGlobalRef()
		secondApprover = gen.UniqueGlobalRef()

		firstVCallRequest, firstVDelegatedCallRequest = true, true

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
		outgoingCall := execution.NewRPCBuilder(outgoingCallRef, objectAGlobal).CallMethod(objectBGlobal, classB, "Bar", []byte("123"))
		objectAExecutionMock := runnerMock.AddExecutionMock("New")
		objectAExecutionMock.AddStart(
			func(_ execution.Context) {
				server.IncrementPulseAndWaitIdle(ctx)
				secondPulse = server.GetPulse().PulseNumber
				firstExpectedToken.PulseNumber = secondPulse
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Error:    nil,
				Outgoing: outgoingCall,
			},
		)
		objectAExecutionMock.AddContinue(
			func(result []byte) {
				require.Equal(t, []byte("finish B.Bar"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish A.New"), objectAGlobal),
			},
		)
	}

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			// check for pending counts must be in tests: call constructor/call terminal method
			assert.Equal(t, objectAGlobal, report.Object)
			assert.Equal(t, payload.Empty, report.Status)
			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
			assert.Equal(t, objectAGlobal, request.Callee)

			msg := payload.VDelegatedCallResponse{
				Callee: request.Callee,
				ResponseDelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       server.GetPulse().PulseNumber,
					Callee:            request.Callee,
					Outgoing:          request.CallOutgoing,
					DelegateTo:        server.JetCoordinatorMock.Me(),
				},
			}

			if firstVDelegatedCallRequest {
				assert.Zero(t, request.DelegationSpec)
				msg.ResponseDelegationSpec.Approver = firstApprover
				firstVDelegatedCallRequest = false
			} else {
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				msg.ResponseDelegationSpec.Approver = secondApprover
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

			if firstVCallRequest {
				assert.Equal(t, secondPulse, request.CallOutgoing.Pulse()) // new pulse
				assert.Equal(t, firstExpectedToken, request.DelegationSpec)
				assert.NotEqual(t, payload.RepeatedCall, request.CallRequestFlags.GetRepeatedCall())

				server.IncrementPulseAndWaitIdle(ctx)
				thirdPulse = server.GetPulse().PulseNumber
				secondExpectedToken.PulseNumber = thirdPulse
				expectedVCallRequest.DelegationSpec = secondExpectedToken
				firstVCallRequest = false
				// request will be sent in previous pulse
				// omit sending
				return false
			}

			assert.Equal(t, secondPulse, request.CallOutgoing.Pulse()) // the same pulse

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

			return false
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, objectAGlobal, res.Callee)
			assert.Equal(t, []byte("finish A.New"), res.ReturnArguments)
			assert.Equal(t, int(firstPulse), int(res.CallOutgoing.Pulse()))
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
		CallOutgoing:   callOutgoing,
	}

	msg := server.WrapPayload(&pl).SetSender(statemachine.APICaller).Finalize()
	server.SendMessage(ctx, msg)

	// wait for all calls and SMs
	{
		testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
		testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	assert.Equal(t, 2, typedChecker.VCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 2, typedChecker.VDelegatedCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

	mc.Finish()
}
