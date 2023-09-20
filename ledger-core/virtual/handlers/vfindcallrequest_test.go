package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/testutils/shareddata"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
)

func TestVFindCallRequest(t *testing.T) {
	var (
		ctx       = instestlogger.TestContext(t)
		mc        = minimock.NewController(t)
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot = conveyor.NewPastPulseSlot(nil, pd.AsRange())
		objectRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		outgoing  = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
	)

	vFindCallRequest := rms.VFindCallRequest{
		LookAt:   pd.GetPulseNumber(),
		Callee:   rms.NewReference(objectRef),
		Outgoing: rms.NewReference(outgoing),
	}

	sender := gen.UniqueGlobalRef()
	smVFindCallRequest := SMVFindCallRequest{
		Meta: &rms.Meta{
			Sender: rms.NewReference(sender),
		},
		Payload:   &vFindCallRequest,
		pulseSlot: &pulseSlot,
	}

	{
		initCtx := smachine.NewInitializationContextMock(mc).
			JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepProcessRequest))

		smVFindCallRequest.Init(initCtx)
	}

	sync := smsync.NewConditionalBool(false, "summaryDone").SyncLink()

	t.Run("found_callsummary_sync", func(t *testing.T) {
		{
			callSummarySyncSdl := smachine.NewUnboundSharedData(&sync)

			execCtx := smachine.NewExecutionContextMock(mc).
				GetPublishedLinkMock.Expect(callsummary.BuildSummarySyncKey(objectRef)).
				Return(callSummarySyncSdl).
				JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepWaitCallResult))

			smVFindCallRequest.stepProcessRequest(execCtx)

			require.Equal(t, callsummary.SyncAccessor{SharedDataLink: callSummarySyncSdl}, smVFindCallRequest.syncLinkAccessor)
		}

		{
			execCtx := smachine.NewExecutionContextMock(mc).
				UseSharedMock.Set(shareddata.CallSharedDataAccessor).
				AcquireMock.Expect(sync).Return(true).
				JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepGetRequestData))

			smVFindCallRequest.stepWaitCallResult(execCtx)
		}
	})

	t.Run("not_found_callsummary", func(t *testing.T) {
		execCtx := smachine.NewExecutionContextMock(mc).
			GetPublishedLinkMock.Expect(callsummary.BuildSummarySharedKey(vFindCallRequest.LookAt)).
			Return(smachine.SharedDataLink{}).
			JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepNotFoundResponse))

		smVFindCallRequest.stepGetRequestData(execCtx)
	})

	sharedCallSummary := callsummary.SharedCallSummary{Requests: callregistry.NewObjectRequestTable()}

	reqs := callregistry.NewWorkingTable()

	vCallResult := rms.VCallResult{
		Callee:       rms.NewReference(objectRef),
		CallOutgoing: rms.NewReference(outgoing),
	}

	reqs.Add(isolation.CallTolerable, outgoing)
	reqs.SetActive(isolation.CallTolerable, outgoing)
	reqs.Finish(isolation.CallTolerable, outgoing, &vCallResult)

	sharedCallSummary.Requests.AddObjectCallResults(objectRef,
		callregistry.ObjectCallResults{
			CallResults: reqs.GetResults(),
		})

	callSummarySdl := smachine.NewUnboundSharedData(&sharedCallSummary)

	t.Run("not_found_in_callsummary", func(t *testing.T) {
		t.Run("object", func(t *testing.T) {
			// try to find for unknown object
			vFindCallRequest := rms.VFindCallRequest{
				LookAt:   pd.GetPulseNumber(),
				Callee:   rms.NewReference(gen.UniqueGlobalRef()),
				Outgoing: rms.NewReference(outgoing),
			}

			smVFindCallRequest := SMVFindCallRequest{
				Meta: &rms.Meta{
					Sender: rms.NewReference(sender),
				},
				Payload:   &vFindCallRequest,
				pulseSlot: &pulseSlot,
			}

			execCtx := smachine.NewExecutionContextMock(mc).
				GetPublishedLinkMock.Expect(callsummary.BuildSummarySharedKey(vFindCallRequest.LookAt)).
				Return(callSummarySdl).
				UseSharedMock.Set(shareddata.CallSharedDataAccessor).
				JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepNotFoundResponse))

			smVFindCallRequest.stepGetRequestData(execCtx)
		})

		t.Run("request", func(t *testing.T) {
			// try to find for unknown request
			vFindCallRequest := rms.VFindCallRequest{
				LookAt:   pd.GetPulseNumber(),
				Callee:   rms.NewReference(objectRef),
				Outgoing: rms.NewReference(gen.UniqueGlobalRef()),
			}

			smVFindCallRequest := SMVFindCallRequest{
				Meta: &rms.Meta{
					Sender: rms.NewReference(sender),
				},
				Payload:   &vFindCallRequest,
				pulseSlot: &pulseSlot,
			}

			execCtx := smachine.NewExecutionContextMock(mc).
				GetPublishedLinkMock.Expect(callsummary.BuildSummarySharedKey(vFindCallRequest.LookAt)).
				Return(callSummarySdl).
				UseSharedMock.Set(shareddata.CallSharedDataAccessor).
				JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepNotFoundResponse))

			smVFindCallRequest.stepGetRequestData(execCtx)
		})

		{
			execCtx := smachine.NewExecutionContextMock(mc).
				JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepSendResponse))

			smVFindCallRequest.stepNotFoundResponse(execCtx)
		}

		// check that send nil in result
		{
			messageSender := messagesender.NewServiceMockWrapper(mc)
			messageSenderAdapter := messageSender.NewAdapterMock()
			messageSenderAdapter.SetDefaultPrepareAsyncCall(ctx)

			smVFindCallRequest.messageSender = messageSenderAdapter.Mock()

			checkMessage := func(msg rmsreg.GoGoSerializable) {
				switch msg0 := msg.(type) {
				case *rms.VFindCallResponse:
					assert.Equal(t, pd.GetPulseNumber(), msg0.LookedAt)
					assert.Equal(t, objectRef, msg0.Callee.GetValue())
					assert.Equal(t, outgoing, msg0.Outgoing.GetValue())
					assert.Equal(t, rms.CallStateMissing, msg0.Status)
					assert.Nil(t, msg0.CallResult)
				default:
					panic("Unexpected message type")
				}
			}
			checkTarget := func(target reference.Global) {
				assert.Equal(t, sender, target)
			}
			messageSender.SendTarget.SetCheckMessage(checkMessage)
			messageSender.SendTarget.SetCheckTarget(checkTarget)

			execCtx := smachine.NewExecutionContextMock(mc).
				StopMock.Expect().
				Return(smachine.StateUpdate{})

			smVFindCallRequest.stepSendResponse(execCtx)
		}
	})

	t.Run("found", func(t *testing.T) {
		{
			execCtx := smachine.NewExecutionContextMock(mc).
				GetPublishedLinkMock.Expect(callsummary.BuildSummarySharedKey(vFindCallRequest.LookAt)).
				Return(callSummarySdl).
				UseSharedMock.Set(shareddata.CallSharedDataAccessor).
				JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepSendResponse))

			smVFindCallRequest.stepGetRequestData(execCtx)

			assert.Equal(t, rms.CallStateFound, smVFindCallRequest.status)
			assert.Equal(t, &vCallResult, smVFindCallRequest.callResult)
		}

		{
			messageSender := messagesender.NewServiceMockWrapper(mc)
			messageSenderAdapter := messageSender.NewAdapterMock()
			messageSenderAdapter.SetDefaultPrepareAsyncCall(ctx)

			smVFindCallRequest.messageSender = messageSenderAdapter.Mock()

			checkMessage := func(msg rmsreg.GoGoSerializable) {
				switch msg0 := msg.(type) {
				case *rms.VFindCallResponse:
					assert.Equal(t, pd.GetPulseNumber(), msg0.LookedAt)
					assert.Equal(t, objectRef, msg0.Callee.GetValue())
					assert.Equal(t, outgoing, msg0.Outgoing.GetValue())
					assert.Equal(t, rms.CallStateFound, msg0.Status)
					assert.Equal(t, &vCallResult, msg0.CallResult)
				default:
					panic("Unexpected message type")
				}
			}
			checkTarget := func(target reference.Global) {
				assert.Equal(t, sender, target)
			}

			messageSender.SendTarget.SetCheckMessage(checkMessage)
			messageSender.SendTarget.SetCheckTarget(checkTarget)

			execCtx := smachine.NewExecutionContextMock(mc).
				StopMock.Expect().
				Return(smachine.StateUpdate{})

			smVFindCallRequest.stepSendResponse(execCtx)
		}
	})

	mc.Finish()
}
