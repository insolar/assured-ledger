// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
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

	vFindCallRequest := payload.VFindCallRequest{
		LookAt:   pd.GetPulseNumber(),
		Callee:   objectRef,
		Outgoing: outgoing,
	}

	sender := gen.UniqueGlobalRef()
	smVFindCallRequest := SMVFindCallRequest{
		Meta: &payload.Meta{
			Sender: sender,
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

	vCallResult := payload.VCallResult{
		Callee:       objectRef,
		CallOutgoing: outgoing,
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
			vFindCallRequest := payload.VFindCallRequest{
				LookAt:   pd.GetPulseNumber(),
				Callee:   gen.UniqueGlobalRef(),
				Outgoing: outgoing,
			}

			smVFindCallRequest := SMVFindCallRequest{
				Meta: &payload.Meta{
					Sender: sender,
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
			vFindCallRequest := payload.VFindCallRequest{
				LookAt:   pd.GetPulseNumber(),
				Callee:   objectRef,
				Outgoing: gen.UniqueGlobalRef(),
			}

			smVFindCallRequest := SMVFindCallRequest{
				Meta: &payload.Meta{
					Sender: sender,
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

			checkMessage := func(msg payload.Marshaler) {
				switch msg0 := msg.(type) {
				case *payload.VFindCallResponse:
					require.Equal(t, pd.GetPulseNumber(), msg0.LookedAt)
					require.Equal(t, objectRef, msg0.Callee)
					require.Equal(t, outgoing, msg0.Outgoing)
					require.Equal(t, payload.CallStateMissing, msg0.Status)
					require.Nil(t, msg0.CallResult)
				default:
					panic("Unexpected message type")
				}
			}
			checkTarget := func(target reference.Global) {
				require.Equal(t, sender, target)
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

			require.Equal(t, payload.CallStateFound, smVFindCallRequest.status)
			require.Equal(t, &vCallResult, smVFindCallRequest.callResult)
		}

		{
			messageSender := messagesender.NewServiceMockWrapper(mc)
			messageSenderAdapter := messageSender.NewAdapterMock()
			messageSenderAdapter.SetDefaultPrepareAsyncCall(ctx)

			smVFindCallRequest.messageSender = messageSenderAdapter.Mock()

			checkMessage := func(msg payload.Marshaler) {
				switch msg0 := msg.(type) {
				case *payload.VFindCallResponse:
					require.Equal(t, pd.GetPulseNumber(), msg0.LookedAt)
					require.Equal(t, objectRef, msg0.Callee)
					require.Equal(t, outgoing, msg0.Outgoing)
					require.Equal(t, payload.CallStateFound, msg0.Status)
					require.Equal(t, &vCallResult, msg0.CallResult)
				default:
					panic("Unexpected message type")
				}
			}
			checkTarget := func(target reference.Global) {
				require.Equal(t, sender, target)
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
