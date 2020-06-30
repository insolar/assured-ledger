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
func TestVirtual_CallMethodOutgoing_WithPulseChange(t *testing.T) {
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

		classB        = gen.UniqueGlobalRef()
		objectAGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		objectBGlobal = reference.NewSelf(server.RandomLocalWithPulse())

		callOutgoing = server.RandomLocalWithPulse()

		outgoingCallRef = reference.NewRecordOf(objectAGlobal, callOutgoing)
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
			func(ctx execution.Context) {
				logger.Debug("ExecutionStart [A.Foo]")
				assert.Equal(t, objectAGlobal, ctx.Object)
				assert.Equal(t, callOutgoing, ctx.Request.CallOutgoing)
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

	callCount := 0

	firstPulse := server.GetPulse().PulseNumber

	expectedToken := payload.CallDelegationToken{
		TokenTypeAndFlags: payload.DelegationTokenTypeCall,
		PulseNumber:       server.GetPulse().PulseNumber,
		Callee:            objectAGlobal,
		Outgoing:          outgoingCallRef,
		DelegateTo:        server.JetCoordinatorMock.Me(),
	}

	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			assert.Equal(t, firstPulse, report.OrderedPendingEarliestPulse)
			assert.Equal(t, int32(1), report.OrderedPendingCount)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
			assert.Zero(t, request.DelegationSpec)

			server.SendPayload(ctx, &payload.VDelegatedCallResponse{
				Callee:                 request.Callee,
				ResponseDelegationSpec: expectedToken,
			})

			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			assert.Equal(t, expectedToken, finished.DelegationSpec)
			return false
		})
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, firstPulse, request.CallOutgoing.Pulse())
			if callCount == 0 {
				assert.Zero(t, request.DelegationSpec)
				server.IncrementPulseAndWaitIdle(ctx)
				callCount++
				// request will be sent in previous pulse
				// omit sending
				return false
			}

			assert.Equal(t, payload.RepeatedCall, request.CallRequestFlags.GetRepeatedCall())

			assert.Equal(t, expectedToken, request.DelegationSpec)

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
			assert.Equal(t, []byte("finish A.Foo"), res.ReturnArguments)
			assert.Equal(t, int(firstPulse), int(res.CallOutgoing.Pulse()))
			assert.Equal(t, expectedToken, res.DelegationSpec)
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
	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 2, typedChecker.VCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 1, typedChecker.VDelegatedCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

	mc.Finish()
}
