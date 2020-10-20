package validation

import (
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
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

// constructor pending +  unordered method call
// check transcript
func TestValidation_HappyPathWithPending(t *testing.T) {
	// todo add case
	defer commonTestUtils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	logger := inslogger.FromContext(ctx)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	{
		server.ReplaceRunner(runnerMock)
		server.Init(ctx)
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		plWrapper      = utils.GenerateVCallRequestConstructor(server)
		constructorReq = plWrapper.Get()
		class          = constructorReq.Callee.GetValue()
		outgoingP1     = constructorReq.CallOutgoing.GetValue()
		object         = reference.NewSelf(outgoingP1.GetLocal())

		incomingP1 = reference.NewRecordOf(object, outgoingP1.GetLocal())
		p1         = server.GetPulse().PulseNumber
		isolation  = contract.MethodIsolation{
			Interference: isolation.CallIntolerable,
			State:        isolation.CallDirty, // todo, need fix it. use dirty state because R0 does not copy dirty to validated state
		}
		getDelegated = false
	)

	bytes := []byte("after pending state")
	afterPendingStateID := server.RandomLocalWithPulse()
	dirtyStateRef := reference.NewRecordOf(object, afterPendingStateID)

	pendingTranscript := utils.GenerateIncomingTranscript(constructorReq, nil, dirtyStateRef, server.RandomGlobalWithPulse(), server.RandomGlobalWithPulse())

	server.IncrementPulseAndWaitIdle(ctx)

	var (
		p2         = server.GetPulse().PulseNumber
		outgoingP2 = server.BuildRandomOutgoingWithPulse()
	)

	// method request
	pl := utils.GenerateVCallRequestMethod(server)
	{
		pl.CallFlags = rms.BuildCallFlags(isolation.Interference, isolation.State)
		pl.Callee.Set(object)
		pl.CallSiteMethod = "SomeMethod"
		pl.CallOutgoing.Set(outgoingP2)
	}
	methodTranscript := utils.GenerateIncomingTranscript(*pl, dirtyStateRef, dirtyStateRef, server.RandomGlobalWithPulse(), server.RandomGlobalWithPulse())

	// create object state
	{
		vsrPayload := &rms.VStateReport{
			Status:                      rms.StateStatusEmpty,
			Object:                      rms.NewReference(object),
			AsOf:                        p1,
			OrderedPendingCount:         1,
			OrderedPendingEarliestPulse: p1,
		}

		server.SendPayload(ctx, vsrPayload)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1))
		server.WaitActiveThenIdleConveyor()
	}

	// add ExecutionMock to runnerMock
	{
		runnerMock.AddExecutionClassify(outgoingP2, isolation, nil)
		requestResult := requestresult.New([]byte("call result"), server.RandomGlobalWithPulse())

		objectExecutionMock := runnerMock.AddExecutionMock(outgoingP2)
		objectExecutionMock.AddStart(func(ctx execution.Context) {
			logger.Debug("ExecutionStart [SomeMethod]")
			require.Equal(t, object, ctx.Request.Callee.GetValue())
			require.Equal(t, bytes, ctx.ObjectDescriptor.Memory())
			require.Equal(t, afterPendingStateID, ctx.ObjectDescriptor.State().GetLocal())
			require.True(t, getDelegated)
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	// add checks to typedChecker
	{
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, object, res.Callee.GetValue())
			assert.Equal(t, []byte("call result"), res.ReturnArguments.GetBytes())
			assert.Equal(t, p2, res.CallOutgoing.GetPulseOfLocal()) // will be fail
			assert.Empty(t, res.DelegationSpec)
			return false
		})
		typedChecker.VDelegatedCallResponse.SetResend(false)
		typedChecker.VStateReport.SetResend(false)
		typedChecker.VObjectTranscriptReport.Set(func(res *rms.VObjectTranscriptReport) bool {

			utils.AssertAnyTranscriptEqual(t, &pendingTranscript.Entries[0], &res.PendingTranscripts[0].Entries[0])
			utils.AssertAnyTranscriptEqual(t, &pendingTranscript.Entries[1], &res.PendingTranscripts[0].Entries[1])

			utils.AssertAnyTranscriptEqual(t, &methodTranscript.Entries[0], &res.ObjectTranscript.Entries[0])
			utils.AssertAnyTranscriptEqual(t, &methodTranscript.Entries[1], &res.ObjectTranscript.Entries[1])

			return false
		})
	}

	// VDelegatedCallRequest
	{
		delegatedRequest := rms.VDelegatedCallRequest{
			Callee:       rms.NewReference(object),
			CallFlags:    rms.BuildCallFlags(constructorReq.CallFlags.GetInterference(), constructorReq.CallFlags.GetState()),
			CallOutgoing: rms.NewReference(outgoingP1),
			CallIncoming: rms.NewReference(incomingP1),
		}
		server.SendPayload(ctx, &delegatedRequest)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVDelegatedCallRequest{}, 1))
	}
	// VDelegatedRequestFinished
	{

		finished := rms.VDelegatedRequestFinished{
			CallType:          rms.CallTypeConstructor,
			CallFlags:         rms.BuildCallFlags(constructorReq.CallFlags.GetInterference(), constructorReq.CallFlags.GetState()),
			Callee:            rms.NewReference(object),
			CallOutgoing:      rms.NewReference(outgoingP1),
			CallIncoming:      rms.NewReference(incomingP1),
			PendingTranscript: pendingTranscript,
			LatestState: &rms.ObjectState{
				Reference:   rms.NewReferenceLocal(dirtyStateRef),
				Class:       rms.NewReference(class),
				Memory:      rms.NewBytes(bytes),
				Deactivated: false,
			},
		}
		server.SendPayload(ctx, &finished)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVDelegatedRequestFinished{}, 1))
		getDelegated = true
	}

	// VCallRequest
	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	server.SendPayload(ctx, pl)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	wait := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)

	server.IncrementPulseAndWaitIdle(ctx)

	// commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1))
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)

	assert.Equal(t, 1, typedChecker.VCallResult.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())

	mc.Finish()
}
