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
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
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
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C5030")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	utils.AssertNotJumpToStep(t, server.Journal, "stepTakeLock")

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	expectedError, err := foundation.MarshalMethodErrorResult(errors.New("bad class reference"))
	require.NoError(t, err)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		assert.Equal(t, expectedError, res.ReturnArguments.GetBytes())

		return false // no resend msg
	})

	{
		pl := utils.GenerateVCallRequestConstructor(server)

		server.SendPayload(ctx, pl)
	}

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
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
		outgoing     = pl.CallOutgoing.GetValue()
		objectRef    = reference.NewSelf(outgoing.GetLocal())
		runnerResult = []byte("123")
		class        = pl.Callee.GetValue()
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		assert.Equal(t, runnerResult, res.ReturnArguments.GetBytes())
		assert.Equal(t, objectRef, res.Callee.GetValue())
		assert.Equal(t, outgoing, res.CallOutgoing.GetValue())
		assert.Equal(t, rms.CallTypeConstructor, res.CallType)
		assert.Equal(t, pl.CallFlags, res.CallFlags)

		return false // no resend msg
	})
	typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
		objectState := rms.ObjectState{
			State: rms.NewBytes([]byte("some memory")),
			Class: rms.NewReference(class),
		}
		expected := &rms.VStateReport{
			Status:           rms.StateStatusReady,
			AsOf:             p,
			Object:           rms.NewReference(objectRef),
			LatestDirtyState: rms.NewReference(objectRef),
			ProvidedContent: &rms.VStateReport_ProvidedContentBody{
				LatestDirtyState:     &objectState,
				LatestValidatedState: &objectState,
			},
		}
		report.ProvidedContent.LatestDirtyState.Reference = rms.Reference{}
		report.ProvidedContent.LatestValidatedState.Reference = rms.Reference{}
		utils.AssertVStateReportsEqual(t, expected, report)

		return false
	})

	typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
		assert.Equal(t, objectRef, report.Object.GetValue())
		assert.Equal(t, pl.CallOutgoing.GetValue().GetLocal().Pulse(), report.AsOf)

		assert.Len(t, report.ObjectTranscript.Entries, 2)

		request, ok := report.ObjectTranscript.Entries[0].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest)
		require.True(t, ok)
		result, ok := report.ObjectTranscript.Entries[1].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingResult)
		require.True(t, ok)

		assert.Empty(t, request.Incoming)
		assert.Empty(t, request.ObjectMemory)
		utils.AssertVCallRequestEqual(t, pl, &request.Request)

		assert.Empty(t, result.IncomingResult)
		assert.Equal(t, pl.CallOutgoing.GetValue().GetLocal().Pulse(), result.ObjectState.Get().GetLocal().Pulse())
		assert.Equal(t, objectRef.GetBase(), result.ObjectState.Get().GetBase())

		return false
	})

	{
		requestResult := requestresult.New(runnerResult, outgoing)
		requestResult.SetActivate(class, []byte("some memory"))

		runnerMock.AddExecutionMock(outgoing).
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
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectTranscriptReport.Wait(ctx, 1))

	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())
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

	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Arguments = rms.NewBytes([]byte("arguments"))

	var (
		prevPulse = server.GetPulse().PulseNumber
		class     = pl.Callee.GetValue()
		outgoing  = pl.CallOutgoing.GetValue()
		objectRef = reference.NewSelf(outgoing.GetLocal())
	)

	server.IncrementPulseAndWaitIdle(ctx)

	currPulse := server.GetPulse().PulseNumber

	{
		requestResult := requestresult.New([]byte("123"), server.RandomGlobalWithPulse())
		requestResult.SetActivate(class, []byte("some memory"))

		runnerMock.AddExecutionMock(outgoing).AddStart(func(execution execution.Context) {
			require.Equal(t, "New", execution.Request.CallSiteMethod)
			require.Equal(t, []byte("arguments"), execution.Request.Arguments.GetBytes())
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		require.Equal(t, []byte("123"), res.ReturnArguments.GetBytes())
		require.Equal(t, objectRef, res.Callee.GetValue())
		require.Equal(t, outgoing, res.CallOutgoing.GetValue())

		return false // no resend msg
	})
	typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
		objectState := rms.ObjectState{
			State: rms.NewBytes([]byte("some memory")),
			Class: rms.NewReference(class),
		}
		expected := &rms.VStateReport{
			Status:           rms.StateStatusReady,
			AsOf:             currPulse,
			Object:           rms.NewReference(objectRef),
			LatestDirtyState: rms.NewReference(objectRef),
			ProvidedContent: &rms.VStateReport_ProvidedContentBody{
				LatestDirtyState:     &objectState,
				LatestValidatedState: &objectState,
			},
		}
		report.ProvidedContent.LatestDirtyState.Reference = rms.Reference{}
		report.ProvidedContent.LatestValidatedState.Reference = rms.Reference{}
		utils.AssertVStateReportsEqual(t, expected, report)

		return false
	})
	typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
		assert.Equal(t, objectRef, report.Object.GetValue())
		assert.Equal(t, currPulse, report.AsOf)

		assert.Len(t, report.ObjectTranscript.Entries, 2)

		request, ok := report.ObjectTranscript.Entries[0].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest)
		require.True(t, ok)
		result, ok := report.ObjectTranscript.Entries[1].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingResult)
		require.True(t, ok)

		assert.Empty(t, request.Incoming)
		assert.Empty(t, request.ObjectMemory)
		utils.AssertVCallRequestEqual(t, pl, &request.Request)

		assert.Empty(t, result.IncomingResult)
		assert.Equal(t, currPulse, result.ObjectState.Get().GetLocal().Pulse())
		assert.Equal(t, objectRef.GetBase(), result.ObjectState.Get().GetBase())

		return false
	})

	{
		done := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		pl := makeVStateReportWithState(objectRef, rms.StateStatusMissing, nil, prevPulse)
		server.SendPayload(ctx, pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
	}

	server.SendPayload(ctx, pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	server.IncrementPulseAndWaitIdle(ctx)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectTranscriptReport.Wait(ctx, 1))

	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())
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

	// generate VCallRequest
	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Callee = rms.NewReference(class)
	pl.CallOutgoing = rms.NewReference(outgoing)
	pl.Arguments = rms.NewBytes([]byte("arguments"))

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	// typedChecks
	{
		typedChecker.VStateRequest.Set(func(req *rms.VStateRequest) bool {
			require.Equal(t, p1, req.AsOf)
			require.Equal(t, objectRef, req.Object.GetValue())

			flags := rms.StateRequestContentFlags(0)
			flags.Set(
				rms.RequestLatestDirtyState,
				rms.RequestLatestValidatedState,
				rms.RequestOrderedQueue,
				rms.RequestUnorderedQueue,
			)
			require.Equal(t, flags, req.RequestedContent)

			report := rms.VStateReport{
				Status: rms.StateStatusMissing,
				AsOf:   p1,
				Object: rms.NewReference(objectRef),
			}

			server.SendMessage(ctx, utils.NewRequestWrapper(p2, &report).SetSender(server.JetCoordinatorMock.Me()).Finalize())

			return false // no resend msg
		})
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, []byte("123"), res.ReturnArguments.GetBytes())
			assert.Equal(t, objectRef, res.Callee.GetValue())
			assert.Equal(t, outgoing, res.CallOutgoing.GetValue())

			return false // no resend msg
		})
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			objectState := rms.ObjectState{
				State: rms.NewBytes([]byte("some memory")),
				Class: rms.NewReference(class),
			}
			expected := &rms.VStateReport{
				Status:           rms.StateStatusReady,
				AsOf:             p2,
				Object:           rms.NewReference(objectRef),
				LatestDirtyState: rms.NewReference(objectRef),
				ProvidedContent: &rms.VStateReport_ProvidedContentBody{
					LatestDirtyState:     &objectState,
					LatestValidatedState: &objectState,
				},
			}
			report.ProvidedContent.LatestDirtyState.Reference = rms.Reference{}
			report.ProvidedContent.LatestValidatedState.Reference = rms.Reference{}
			utils.AssertVStateReportsEqual(t, expected, report)

			return false
		})
		typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, p2, report.AsOf)

			assert.Len(t, report.ObjectTranscript.Entries, 2)

			request, ok := report.ObjectTranscript.Entries[0].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest)
			require.True(t, ok)
			result, ok := report.ObjectTranscript.Entries[1].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingResult)
			require.True(t, ok)

			assert.Empty(t, request.Incoming)
			assert.Empty(t, request.ObjectMemory)
			utils.AssertVCallRequestEqual(t, pl, &request.Request)

			assert.Empty(t, result.IncomingResult)
			assert.Equal(t, p2, result.ObjectState.Get().GetLocal().Pulse())
			assert.Equal(t, objectRef.GetBase(), result.ObjectState.Get().GetBase())

			return false
		})
	}

	{
		requestResult := requestresult.New([]byte("123"), server.RandomGlobalWithPulse())
		requestResult.SetActivate(class, []byte("some memory"))

		runnerMock.AddExecutionMock(outgoing).
			AddStart(func(execution execution.Context) {
				require.Equal(t, "New", execution.Request.CallSiteMethod)
				require.Equal(t, []byte("arguments"), execution.Request.Arguments.GetBytes())
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
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectTranscriptReport.Wait(ctx, 1))

	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())
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

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) interface{} {
		return execution.Request.Callee.GetValue()
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		isolation = contract.ConstructorIsolation()
		callFlags = rms.BuildCallFlags(isolation.Interference, isolation.State)

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
		objectAResult.SetActivate(classA, []byte("state A"))
		objectAExecutionMock := runnerMock.AddExecutionMock(classA)
		objectAExecutionMock.AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [A.New]")
				require.Equal(t, classA, ctx.Request.Callee.GetValue())
				require.Equal(t, outgoingA, ctx.Request.CallOutgoing.GetValue())
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
		objectBResult.SetActivate(classB, []byte("state B"))
		runnerMock.AddExecutionMock(classB).AddStart(
			func(ctx execution.Context) {
				t.Log("ExecutionStart [B.New]")
				assert.Equal(t, classB, ctx.Request.Callee.GetValue())
				assert.Equal(t, objectA, ctx.Request.Caller.GetValue())
				assert.Equal(t, []byte("123"), ctx.Request.Arguments.GetBytes())
			},
			&execution.Update{
				Type:   execution.Done,
				Result: objectBResult,
			},
		)
	}

	// add checks to typedChecker
	{
		typedChecker.VCallRequest.Set(func(request *rms.VCallRequest) bool {
			assert.Equal(t, classB, request.Callee.GetValue())
			assert.Equal(t, objectA, request.Caller.GetValue())
			assert.Equal(t, []byte("123"), request.Arguments.GetBytes())
			assert.Equal(t, rms.CallTypeConstructor, request.CallType)
			assert.Equal(t, uint32(1), request.CallSequence)
			assert.Equal(t, callFlags, request.CallFlags)
			assert.Equal(t, server.GetPulse().PulseNumber, request.CallOutgoing.GetPulseOfLocal())
			return true // resend
		})
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, rms.CallTypeConstructor, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)

			switch res.Callee.GetValue() {
			case objectA:
				require.Equal(t, []byte("finish A.New"), res.ReturnArguments.GetBytes())
				require.Equal(t, server.GlobalCaller(), res.Caller.GetValue())
				require.Equal(t, outgoingA, res.CallOutgoing.GetValue())
			default:
				require.Equal(t, []byte("finish B.New"), res.ReturnArguments.GetBytes())
				require.Equal(t, objectA, res.Caller.GetValue())
				require.Equal(t, server.GetPulse().PulseNumber, res.CallOutgoing.GetPulseOfLocal())
			}
			// we should resend that message only if it's CallResult from B to A
			return res.Caller.GetValue() == objectA
		})
	}

	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Callee = rms.NewReference(classA)
	pl.CallOutgoing = rms.NewReference(outgoingA)
	server.SendPayload(ctx, pl)

	// wait for all calls and SMs
	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VCallRequest.Count())
	require.Equal(t, 2, typedChecker.VCallResult.Count())

	// check transcripts
	typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
		return false
	})

	typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
		if report.Object.GetValue() == objectA {
			assert.Len(t, report.ObjectTranscript.Entries, 4)

			_, ok := report.ObjectTranscript.Entries[0].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest)
			require.True(t, ok)

			_, ok = report.ObjectTranscript.Entries[1].Get().(*rms.VObjectTranscriptReport_TranscriptEntryOutgoingRequest)
			require.True(t, ok)

			_, ok = report.ObjectTranscript.Entries[2].Get().(*rms.VObjectTranscriptReport_TranscriptEntryOutgoingResult)
			require.True(t, ok)

			_, ok = report.ObjectTranscript.Entries[3].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingResult)
			require.True(t, ok)
		} else {
			assert.Len(t, report.ObjectTranscript.Entries, 2)

			_, ok := report.ObjectTranscript.Entries[0].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest)
			require.True(t, ok)

			_, ok = report.ObjectTranscript.Entries[1].Get().(*rms.VObjectTranscriptReport_TranscriptEntryIncomingResult)
			require.True(t, ok)
		}

		return false
	})

	server.IncrementPulseAndWaitIdle(ctx)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectTranscriptReport.Wait(ctx, 2))

	// 2 objects, 2 transcripts
	assert.Equal(t, 2, typedChecker.VObjectTranscriptReport.Count())

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
	pl.Callee.Set(testwallet.GetClass())
	pl.CallSiteMethod = "NotExistingConstructorName"

	var (
		outgoing  = pl.CallOutgoing.GetValue()
		objectRef = reference.NewSelf(outgoing.GetLocal())
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		require.Equal(t, objectRef, res.Callee.GetValue())
		require.Equal(t, outgoing, res.CallOutgoing.GetValue())

		contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
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
		callFlags = rms.BuildCallFlags(isolation.Interference, isolation.State)

		class     = pl.Callee.GetValue()
		outgoing  = pl.CallOutgoing.GetValue()
		objectRef = reference.NewSelf(outgoing.GetLocal())

		constructorPulse = server.GetPulse().PulseNumber

		delegationToken rms.CallDelegationToken
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	// add type checks
	{
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, rms.StateStatusEmpty, report.Status)
			assert.Equal(t, int32(1), report.OrderedPendingCount)
			assert.Equal(t, constructorPulse, report.OrderedPendingEarliestPulse)
			assert.Equal(t, int32(0), report.UnorderedPendingCount)
			assert.Empty(t, report.UnorderedPendingEarliestPulse)
			assert.Empty(t, report.LatestDirtyState)
			assert.Empty(t, report.LatestValidatedState)

			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, constructorPulse, report.AsOf)
			require.Len(t, report.ObjectTranscript.Entries, 0)

			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(msg *rms.VDelegatedCallRequest) bool {
			assert.Zero(t, msg.DelegationSpec)
			assert.Equal(t, objectRef, msg.Callee.GetValue())
			assert.Equal(t, outgoing, msg.CallOutgoing.GetValue())

			delegationToken = server.DelegationToken(msg.CallOutgoing.GetValue(), server.GlobalCaller(), msg.Callee.GetValue())
			server.SendPayload(ctx, &rms.VDelegatedCallResponse{
				Callee:                 msg.Callee,
				CallIncoming:           msg.CallIncoming,
				ResponseDelegationSpec: delegationToken,
			})
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *rms.VDelegatedRequestFinished) bool {
			assert.Equal(t, rms.CallTypeConstructor, finished.CallType)
			assert.Equal(t, callFlags, finished.CallFlags)
			assert.Equal(t, outgoing, finished.CallOutgoing.GetValue())
			assert.Equal(t, objectRef, finished.Callee.GetValue())
			require.NotNil(t, finished.LatestState)
			assert.Equal(t, []byte("234"), finished.LatestState.State.GetBytes())
			assert.Equal(t, delegationToken, finished.DelegationSpec)
			return false
		})
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, []byte("123"), res.ReturnArguments.GetBytes())
			assert.Equal(t, outgoing, res.CallOutgoing.GetValue())
			assert.Equal(t, objectRef, res.Callee.GetValue())
			assert.Equal(t, rms.CallTypeConstructor, res.CallType)
			assert.Equal(t, callFlags, res.CallFlags)
			assert.Equal(t, delegationToken, res.DelegationSpec)
			return false
		})
	}

	synchronizeExecution := synchronization.NewPoint(1)

	// add executionMock
	{
		requestResult := requestresult.New([]byte("123"), outgoing)
		requestResult.SetActivate(class, []byte("234"))

		runnerMock.AddExecutionMock(outgoing).
			AddStart(func(ctx execution.Context) {
				synchronizeExecution.Synchronize()
			}, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	// add authService mock functions
	{
		authService.HasToSendTokenMock.Set(func(token rms.CallDelegationToken) (b1 bool) {
			assert.Equal(t, objectRef, token.Callee.GetValue())
			return true
		})
		authService.CheckMessageFromAuthorizedVirtualMock.Set(func(ctx context.Context, payloadObj interface{}, sender reference.Global, pr pulse.Range) (mustReject bool, err error) {
			assert.Equal(t, server.GlobalCaller(), sender)
			return false, nil
		})
		authService.GetCallDelegationTokenMock.Set(func(outgoingRef reference.Global, to reference.Global, pn pulse.Number, object reference.Global) (c1 rms.CallDelegationToken) {
			assert.Equal(t, objectRef, object)
			return rms.CallDelegationToken{
				TokenTypeAndFlags: rms.DelegationTokenTypeCall,
				PulseNumber:       server.GetPulse().PulseNumber,
				Callee:            rms.NewReference(object),
				Outgoing:          rms.NewReference(outgoingRef),
				DelegateTo:        rms.NewReference(to),
				Approver:          rms.NewReference(server.GlobalCaller()),
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

	msgVStateRequest := rms.VStateRequest{
		AsOf:   constructorPulse,
		Object: rms.NewReference(objectRef),
	}

	server.SendPayload(ctx, &msgVStateRequest)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 2))
	commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectTranscriptReport.Wait(ctx, 1))

	{
		assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())
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

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		classA    = server.RandomGlobalWithPulse()
		outgoing  = server.BuildRandomOutgoingWithPulse()
		objectRef = reference.NewSelf(outgoing.GetLocal())

		firstPulse = server.GetPulse().PulseNumber

		firstApprover  = server.RandomGlobalWithPulse()
		secondApprover = server.RandomGlobalWithPulse()

		firstExpectedToken, secondExpectedToken rms.CallDelegationToken
	)

	synchronizeExecution := synchronization.NewPoint(1)

	// add ExecutionMocks to runnerMock
	{
		objectAResult := requestresult.New([]byte("finish A.New"), outgoing)
		objectAResult.SetActivate(classA, []byte("state A"))
		runnerMock.AddExecutionMock(outgoing).AddStart(func(_ execution.Context) {
			synchronizeExecution.Synchronize()
		}, &execution.Update{
			Type:   execution.Done,
			Result: objectAResult,
		})
	}

	// generate VCallRequest
	pl := utils.GenerateVCallRequestConstructor(server)
	pl.Callee.Set(classA)
	pl.CallOutgoing.Set(outgoing)

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			// check for pending counts must be in tests: call constructor/call terminal method
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, rms.StateStatusEmpty, report.Status)
			assert.Zero(t, report.DelegationSpec)
			return false
		})
		typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, pl.CallOutgoing.GetValue().GetLocal().Pulse(), report.AsOf)
			require.Len(t, report.ObjectTranscript.Entries, 0)

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
			default:
				t.Fatal("unexpected")
			}

			server.SendPayload(ctx, &msg)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *rms.VDelegatedRequestFinished) bool {
			assert.Equal(t, objectRef, finished.Callee.GetValue())
			assert.Equal(t, rms.CallTypeConstructor, finished.CallType)
			assert.NotNil(t, finished.LatestState)
			assert.Equal(t, secondExpectedToken, finished.DelegationSpec)
			assert.Equal(t, []byte("state A"), finished.LatestState.State.GetBytes())
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
	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())

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
			pl.CallFlags = rms.BuildCallFlags(test.isolation.Interference, test.isolation.State)
			pl.Callee.Set(testwallet.GetClass())

			var (
				outgoing  = pl.CallOutgoing.GetValue()
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
			typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
				require.Equal(t, objectRef, result.Callee.GetValue())
				require.Equal(t, outgoing, result.CallOutgoing.GetValue())

				contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments.GetBytes())
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
