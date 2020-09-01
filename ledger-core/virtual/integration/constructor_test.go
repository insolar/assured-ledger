// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_Constructor_BadClassRef(t *testing.T) {
	insrail.LogCase(t, "C5030")

	table := []struct {
		name         string
		randomCallee bool
	}{
		// {"Callee is empty", false},
		{"Callee is randomRef", true},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			var (
				mc     = minimock.NewController(t)
				callee payload.Reference
			)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			utils.AssertNotJumpToStep(t, server.Journal, "stepTakeLock")

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			expectedError, err := foundation.MarshalMethodErrorResult(errors.New("bad class reference"))
			require.NoError(t, err)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
				require.Equal(t, expectedError, res.ReturnArguments)

				return false // no resend msg
			})

			if test.randomCallee {
				callee = server.RandomGlobalWithPulse()
			}

			{
				pl := utils.GenerateVCallRequestConstructor(server)
				pl.Callee = callee

				server.SendPayload(ctx, pl)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
			}

			assert.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_Constructor_CurrentPulseWithoutObject(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4995")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	pl := utils.GenerateVCallRequestConstructor(server)

	var (
		p            = server.GetPulse().PulseNumber
		outgoing     = pl.CallOutgoing
		objectRef    = reference.NewSelf(outgoing.GetLocal())
		runnerResult = []byte("123")
		class        = pl.Callee
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, runnerResult, res.ReturnArguments)
		require.Equal(t, objectRef, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)
		require.Equal(t, payload.CallTypeConstructor, res.CallType)
		require.Equal(t, pl.CallFlags, res.CallFlags)

		return false // no resend msg
	})
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		objectState := payload.ObjectState{
			State: []byte("some memory"),
			Class: class,
		}
		expected := &payload.VStateReport{
			Status:           payload.StateStatusReady,
			AsOf:             p,
			Object:           objectRef,
			LatestDirtyState: objectRef,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState:     &objectState,
				LatestValidatedState: &objectState,
			},
		}
		expected.ProvidedContent.LatestDirtyState.Reference =
			report.ProvidedContent.LatestDirtyState.Reference
		assert.Equal(t, expected, report)

		return false
	})

	{
		requestResult := requestresult.New(runnerResult, outgoing)
		requestResult.SetActivate(reference.Global{}, class, []byte("some memory"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(nil, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	server.SendPayload(ctx, pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	server.IncrementPulseAndWaitIdle(ctx)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))

	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())

	mc.Finish()
}

func TestVirtual_Constructor_HasStateWithMissingStatus(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4996")

	// VE has object's state record with Status==StateStatusMissing
	// Constructor call should work on top of such entry
	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Arguments = []byte("arguments")

	var (
		prevPulse = server.GetPulse().PulseNumber
		class     = pl.Callee
		outgoing  = pl.CallOutgoing
		objectRef = reference.NewSelf(outgoing.GetLocal())
	)

	server.IncrementPulseAndWaitIdle(ctx)

	currPulse := server.GetPulse().PulseNumber

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte("some memory"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(func(execution execution.Context) {
				require.Equal(t, "New", execution.Request.CallSiteMethod)
				require.Equal(t, []byte("arguments"), execution.Request.Arguments)
			}, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, []byte("123"), res.ReturnArguments)
		require.Equal(t, objectRef, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)

		return false // no resend msg
	})
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		objectState := payload.ObjectState{
			State: []byte("some memory"),
			Class: class,
		}
		expected := &payload.VStateReport{
			Status:           payload.StateStatusReady,
			AsOf:             currPulse,
			Object:           objectRef,
			LatestDirtyState: objectRef,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState:     &objectState,
				LatestValidatedState: &objectState,
			},
		}
		expected.ProvidedContent.LatestDirtyState.Reference =
			report.ProvidedContent.LatestDirtyState.Reference
		assert.Equal(t, expected, report)

		return false
	})

	{
		done := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		pl := makeVStateReportWithState(objectRef, payload.StateStatusMissing, nil, prevPulse)
		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
	}

	server.SendPayload(ctx, pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	server.IncrementPulseAndWaitIdle(ctx)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))

	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())

	mc.Finish()
}

func TestVirtual_Constructor_PrevPulseStateWithMissingStatus(t *testing.T) {
	defer commontestutils.LeakTester(t)
	// Constructor call with outgoing.Pulse < currentPulse
	// state request, state report{Status: StateStatusMissing}
	insrail.LogCase(t, "C4997")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		p1        = server.GetPulse().PulseNumber
		outgoing  = server.BuildRandomOutgoingWithPulse()
		objectRef = reference.NewSelf(outgoing.GetLocal())
		class     = server.RandomGlobalWithPulse()
	)

	server.IncrementPulseAndWaitIdle(ctx)
	p2 := server.GetPulse().PulseNumber

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateRequest.Set(func(req *payload.VStateRequest) bool {
		require.Equal(t, p1, req.AsOf)
		require.Equal(t, objectRef, req.Object)

		flags := payload.StateRequestContentFlags(0)
		flags.Set(
			payload.RequestLatestDirtyState,
			payload.RequestLatestValidatedState,
			payload.RequestOrderedQueue,
			payload.RequestUnorderedQueue,
		)
		require.Equal(t, flags, req.RequestedContent)

		report := payload.VStateReport{
			Status: payload.StateStatusMissing,
			AsOf:   p1,
			Object: objectRef,
		}

		server.SendMessage(ctx, utils.NewRequestWrapper(p2, &report).SetSender(server.JetCoordinatorMock.Me()).Finalize())

		return false // no resend msg
	})
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, []byte("123"), res.ReturnArguments)
		require.Equal(t, objectRef, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)

		return false // no resend msg
	})
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		objectState := payload.ObjectState{
			State: []byte("some memory"),
			Class: class,
		}
		expected := &payload.VStateReport{
			Status:           payload.StateStatusReady,
			AsOf:             p2,
			Object:           objectRef,
			LatestDirtyState: objectRef,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState:     &objectState,
				LatestValidatedState: &objectState,
			},
		}
		expected.ProvidedContent.LatestDirtyState.Reference =
			report.ProvidedContent.LatestDirtyState.Reference
		assert.Equal(t, expected, report)

		return false
	})

	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Callee = class
	pl.CallOutgoing = outgoing
	pl.Arguments = []byte("arguments")

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueGlobalRef())
		requestResult.SetActivate(gen.UniqueGlobalRef(), class, []byte("some memory"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(func(execution execution.Context) {
				require.Equal(t, "New", execution.Request.CallSiteMethod)
				require.Equal(t, []byte("arguments"), execution.Request.Arguments)
			}, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	server.SendPayload(ctx, pl)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	server.IncrementPulseAndWaitIdle(ctx)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateRequest.Wait(ctx, 1))

	require.Equal(t, 1, typedChecker.VStateRequest.Count())
	require.Equal(t, 1, typedChecker.VCallResult.Count())
	require.Equal(t, 1, typedChecker.VStateReport.Count())

	mc.Finish()
}

// A.New calls B.New
func TestVirtual_CallConstructorFromConstructor(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5090")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.Callee.String()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		isolation = contract.ConstructorIsolation()
		callFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)

		classA    = server.RandomGlobalWithPulse()
		outgoingA = server.BuildRandomOutgoingWithPulse()
		objectA   = reference.NewSelf(outgoingA.GetLocal())
		incomingA = reference.NewRecordOf(classA, outgoingA.GetLocal())

		classB        = server.RandomGlobalWithPulse()
		objectBGlobal = server.RandomGlobalWithPulse()
	)

	// add ExecutionMocks to runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(incomingA, objectA).CallConstructor(classB, "New", []byte("123"))
		objectAResult := requestresult.New([]byte("finish A.New"), outgoingA)
		objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))
		objectAExecutionMock := runnerMock.AddExecutionMock(classA.String())
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.New]")
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
				t.Log("ExecutionContinue [A.New]")
				require.Equal(t, []byte("finish B.New"), result)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: objectAResult,
			},
		)

		objectBResult := requestresult.New([]byte("finish B.New"), objectBGlobal)
		objectBResult.SetActivate(reference.Global{}, classB, []byte("state B"))
		runnerMock.AddExecutionMock(classB.String()).AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B.New]")
				require.Equal(t, classB, ctx.Request.Callee)
				require.Equal(t, objectA, ctx.Request.Caller)
				require.Equal(t, []byte("123"), ctx.Request.Arguments)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: objectBResult,
			},
		)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *payload.VCallRequest) bool {
			assert.Equal(t, classB, request.Callee)
			assert.Equal(t, objectA, request.Caller)
			assert.Equal(t, []byte("123"), request.Arguments)
			assert.Equal(t, payload.CallTypeConstructor, request.CallType)
			assert.Equal(t, uint32(1), request.CallSequence)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetLocal().Pulse())
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, payload.CallTypeConstructor, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee {
			case objectA:
				require.Equal(t, []byte("finish A.New"), res.ReturnArguments)
				require.Equal(t, server.GlobalCaller(), res.Caller)
				require.Equal(t, outgoingA, res.CallOutgoing)
			default:
				require.Equal(t, []byte("finish B.New"), res.ReturnArguments)
				require.Equal(t, objectA, res.Caller)
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.GetLocal().Pulse())
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller == objectA
		})
	}

	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Callee = classA
	pl.CallOutgoing = outgoingA
	server.SendPayload(ctx, pl)

	// wait for all calls and SMs
	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Constructor_WrongConstructorName(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4977")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	utils.AssertNotJumpToStep(t, server.Journal, "stepTakeLock")

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Callee = testwallet.GetClass()
	pl.CallSiteMethod = "NotExistingConstructorName"

	var (
		outgoing  = pl.CallOutgoing
		objectRef = reference.NewSelf(outgoing.GetLocal())
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		require.Equal(t, objectRef, res.Callee)
		require.Equal(t, outgoing, res.CallOutgoing)

		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
		require.Equal(t, &foundation.Error{"failed to execute request;\texecution error;\tfailed to find contracts constructor"}, contractErr)
		require.NoError(t, sysErr)

		return false // no resend msg
	})

	server.SendPayload(ctx, pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Constructor_PulseChangedWhileOutgoing(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5085")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)

	authService := authentication.NewServiceMock(t)
	server.ReplaceAuthenticationService(authService)

	server.Init(ctx)

	pl := utils.GenerateVCallRequestConstructor(server)

	var (
		isolation = contract.ConstructorIsolation()
		callFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)

		class     = pl.Callee
		outgoing  = pl.CallOutgoing
		objectRef = reference.NewSelf(outgoing.GetLocal())

		constructorPulse = server.GetPulse().PulseNumber

		delegationToken payload.CallDelegationToken
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	// add type checks
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			assert.Equal(t, objectRef, report.Object)
			assert.Equal(t, payload.StateStatusEmpty, report.Status)
			assert.Equal(t, int32(1), report.OrderedPendingCount)
			assert.Equal(t, constructorPulse, report.OrderedPendingEarliestPulse)
			assert.Equal(t, int32(0), report.UnorderedPendingCount)
			assert.Empty(t, report.UnorderedPendingEarliestPulse)
			assert.Empty(t, report.LatestDirtyState)
			assert.Empty(t, report.LatestValidatedState)

			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(msg *payload.VDelegatedCallRequest) bool {
			assert.Zero(t, msg.DelegationSpec)
			assert.Equal(t, objectRef, msg.Callee)
			assert.Equal(t, outgoing, msg.CallOutgoing)

			delegationToken = server.DelegationToken(msg.CallOutgoing, server.GlobalCaller(), msg.Callee)
			server.SendPayload(ctx, &payload.VDelegatedCallResponse{
				Callee:                 msg.Callee,
				CallIncoming:           msg.CallIncoming,
				ResponseDelegationSpec: delegationToken,
			})
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			assert.Equal(t, payload.CallTypeConstructor, finished.CallType)
			assert.Equal(t, callFlags, finished.CallFlags)
			assert.Equal(t, outgoing, finished.CallOutgoing)
			assert.Equal(t, objectRef, finished.Callee)
			require.NotNil(t, finished.LatestState)
			assert.Equal(t, []byte("234"), finished.LatestState.State)
			assert.Equal(t, delegationToken, finished.DelegationSpec)
			return false
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, []byte("123"), res.ReturnArguments)
			assert.Equal(t, outgoing, res.CallOutgoing)
			assert.Equal(t, objectRef, res.Callee)
			assert.Equal(t, payload.CallTypeConstructor, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)
			assert.Equal(t, delegationToken, res.DelegationSpec)
			return false
		})
	}

	synchronizeExecution := synchronization.NewPoint(1)

	// add executionMock
	{
		requestResult := requestresult.New([]byte("123"), outgoing)
		requestResult.SetActivate(reference.Global{}, class, []byte("234"))

		runnerMock.AddExecutionMock(outgoing.String()).
			AddStart(func(ctx execution.Context) {
				synchronizeExecution.Synchronize()
			}, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	// add authService mock functions
	{
		authService.HasToSendTokenMock.Set(func(token payload.CallDelegationToken) (b1 bool) {
			assert.Equal(t, objectRef, token.Callee)
			return true
		})
		authService.CheckMessageFromAuthorizedVirtualMock.Set(func(ctx context.Context, payloadObj interface{}, sender reference.Global, pr pulse.Range) (mustReject bool, err error) {
			assert.Equal(t, server.GlobalCaller(), sender)
			return false, nil
		})
		authService.GetCallDelegationTokenMock.Set(func(outgoingRef reference.Global, to reference.Global, pn pulse.Number, object reference.Global) (c1 payload.CallDelegationToken) {
			assert.Equal(t, objectRef, object)
			return payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				PulseNumber:       server.GetPulse().PulseNumber,
				Callee:            object,
				Outgoing:          outgoingRef,
				DelegateTo:        to,
				Approver:          server.GlobalCaller(),
			}
		})
	}

	server.SendPayload(ctx, pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
	server.IncrementPulseAndWaitIdle(ctx)

	synchronizeExecution.WakeUp()

	synchronizeExecution.Done()
	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	msgVStateRequest := payload.VStateRequest{
		AsOf:   constructorPulse,
		Object: objectRef,
	}

	server.SendPayload(ctx, &msgVStateRequest)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 2))

	{
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedCallRequest.Count())
		assert.Equal(t, 0, typedChecker.VDelegatedCallResponse.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())
		assert.Equal(t, 2, typedChecker.VStateReport.Count())
	}

	mc.Finish()
}

// -> VCallRequest [A.New]
// -> ExecutionStart
// -> change pulse -> secondPulse
// -> VStateReport [A]
// -> VDelegatedCallRequest [A]
// -> change pulse -> thirdPulse
// -> NO VStateReport
// -> VDelegatedCallRequest [A] + first token
// -> VCallResult [A.New] + second token
// -> VDelegatedRequestFinished [A] + second token
func TestVirtual_CallConstructor_WithTwicePulseChange(t *testing.T) {
	insrail.LogCase(t, "C5208")

	defer commontestutils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		classA    = server.RandomGlobalWithPulse()
		outgoing  = server.BuildRandomOutgoingWithPulse()
		objectRef = reference.NewSelf(outgoing.GetLocal())

		firstPulse = server.GetPulse().PulseNumber

		firstApprover  = server.RandomGlobalWithPulse()
		secondApprover = server.RandomGlobalWithPulse()

		firstExpectedToken, secondExpectedToken payload.CallDelegationToken
	)

	synchronizeExecution := synchronization.NewPoint(1)

	// add ExecutionMocks to runnerMock
	{
		objectAResult := requestresult.New([]byte("finish A.New"), outgoing)
		objectAResult.SetActivate(reference.Global{}, classA, []byte("state A"))
		runnerMock.AddExecutionMock("New").AddStart(
			func(_ execution.Context) {
				synchronizeExecution.Synchronize()
			},
			&execution.Update{
				Type:   execution.Done,
				Result: objectAResult,
			},
		)
	}

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			// check for pending counts must be in tests: call constructor/call terminal method
			assert.Equal(t, objectRef, report.Object)
			assert.Equal(t, payload.StateStatusEmpty, report.Status)
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
			default:
				t.Fatal("unexpected")
			}

			server.SendPayload(ctx, &msg)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			assert.Equal(t, objectRef, finished.Callee)
			assert.Equal(t, payload.CallTypeConstructor, finished.CallType)
			assert.NotNil(t, finished.LatestState)
			assert.Equal(t, secondExpectedToken, finished.DelegationSpec)
			assert.Equal(t, []byte("state A"), finished.LatestState.State)
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

	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Callee = classA
	pl.CallOutgoing = outgoing
	execDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	server.SendPayload(ctx, pl)

	// wait for results
	{
		commontestutils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())

		// wait for pulse change
		for i := 0; i < 2; i++ {
			tokenRequestDone := server.Journal.Wait(
				predicate.ChainOf(
					predicate.NewSMTypeFilter(&execute.SMDelegatedTokenRequest{}, predicate.AfterAnyStopOrError),
					predicate.NewSMTypeFilter(&execute.SMExecute{}, predicate.BeforeStep((&execute.SMExecute{}).StepWaitExecutionResult)),
				),
			)
			server.IncrementPulseAndWaitIdle(ctx)
			commontestutils.WaitSignalsTimed(t, 20*time.Second, tokenRequestDone)
		}

		synchronizeExecution.Done()
		// wait for SMExecutcute finish
		commontestutils.WaitSignalsTimed(t, 10*time.Second, execDone)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 2, typedChecker.VDelegatedCallRequest.Count())
	assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

	mc.Finish()
}

func TestVirtual_Constructor_IsolationNegotiation(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5031")

	table := []struct {
		name      string
		isolation contract.MethodIsolation
	}{
		{
			name:      "call constructor with intolerable and dirty flags",
			isolation: contract.MethodIsolation{Interference: isolation.CallIntolerable, State: isolation.CallDirty},
		},
		{
			name:      "call constructor with tolerable and validated flags",
			isolation: contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallValidated},
		},
		{
			name:      "call constructor with intolerable and validated flags",
			isolation: contract.MethodIsolation{Interference: isolation.CallIntolerable, State: isolation.CallValidated},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			pl := utils.GenerateVCallRequestConstructor(server)
			pl.CallFlags = payload.BuildCallFlags(test.isolation.Interference, test.isolation.State)
			pl.Callee = testwallet.GetClass()

			var (
				outgoing  = pl.CallOutgoing
				objectRef = reference.NewSelf(outgoing.GetLocal())
			)

			expectedError := throw.W(throw.IllegalValue(), "failed to negotiate call isolation params", struct {
				methodIsolation contract.MethodIsolation
				callIsolation   contract.MethodIsolation
			}{
				methodIsolation: contract.ConstructorIsolation(),
				callIsolation:   test.isolation,
			})

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
				require.Equal(t, objectRef, result.Callee)
				require.Equal(t, outgoing, result.CallOutgoing)

				contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments)
				require.NoError(t, sysErr)
				require.NotNil(t, contractErr)
				require.Equal(t, expectedError.Error(), contractErr.Error())

				return false // no resend msg
			})

			server.SendPayload(ctx, pl)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}
