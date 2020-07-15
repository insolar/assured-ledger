// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/shareddata"
	"github.com/stretchr/testify/require"
)

func TestVFindCallRequest(t *testing.T) {
	var (
		ctx       = instestlogger.TestContext(t)
		mc        = minimock.NewController(t)
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot = conveyor.NewPastPulseSlot(nil, pd.AsRange())
		objectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		objectRef = reference.NewSelf(objectID)
		outgoing  = gen.UniqueGlobalRef()
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

	// not found call summary
	{
		execCtx := smachine.NewExecutionContextMock(mc).
			GetPublishedLinkMock.Expect(callsummary.BuildSummarySharedKey(vFindCallRequest.LookAt)).
			Return(smachine.SharedDataLink{}).
			JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepNotFoundResponse))

		smVFindCallRequest.stepGetRequestData(execCtx)
	}

	sharedCallSummary := callsummary.SharedCallSummary{Requests: callregistry.NewObjectRequestTable()}

	reqs := callregistry.NewWorkingTable()

	vCallResult := payload.VCallResult{
		Callee:       objectRef,
		CallOutgoing: outgoing,
	}

	reqs.Add(contract.CallTolerable, outgoing)
	reqs.SetActive(contract.CallTolerable, outgoing)
	reqs.Finish(contract.CallTolerable, outgoing, &vCallResult)

	sharedCallSummary.Requests.AddObjectCallResults(objectRef,
		callregistry.ObjectCallResults{
			CallResults: reqs.GetResults(),
		})

	callSummarySdl := smachine.NewUnboundSharedData(&sharedCallSummary)

	// not found request in call summary
	{
		{
			// try to find for unknown object
			vFindCallRequest := payload.VFindCallRequest{
				LookAt:   pd.GetPulseNumber(),
				Callee:   gen.UniqueGlobalRef(),
				Outgoing: outgoing,
			}

			smVFindCallRequest.Payload = &vFindCallRequest

			execCtx := smachine.NewExecutionContextMock(mc).
				GetPublishedLinkMock.Expect(callsummary.BuildSummarySharedKey(vFindCallRequest.LookAt)).
				Return(callSummarySdl).
				UseSharedMock.Set(shareddata.CallSharedDataAccessor).
				JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepNotFoundResponse))

			smVFindCallRequest.stepGetRequestData(execCtx)
		}

		{
			// try to find for unknown request
			vFindCallRequest := payload.VFindCallRequest{
				LookAt:   pd.GetPulseNumber(),
				Callee:   objectRef,
				Outgoing: gen.UniqueGlobalRef(),
			}

			smVFindCallRequest.Payload = &vFindCallRequest

			execCtx := smachine.NewExecutionContextMock(mc).
				GetPublishedLinkMock.Expect(callsummary.BuildSummarySharedKey(vFindCallRequest.LookAt)).
				Return(callSummarySdl).
				UseSharedMock.Set(shareddata.CallSharedDataAccessor).
				JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepNotFoundResponse))

			smVFindCallRequest.stepGetRequestData(execCtx)
		}

		// reset to correct object and request ref
		smVFindCallRequest.Payload = &vFindCallRequest

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
					require.Equal(t, payload.MissingCall, msg0.Status)
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
	}

	// passed
	{
		execCtx := smachine.NewExecutionContextMock(mc).
			GetPublishedLinkMock.Expect(callsummary.BuildSummarySharedKey(vFindCallRequest.LookAt)).
			Return(callSummarySdl).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			JumpMock.Set(testutils.AssertJumpStep(t, smVFindCallRequest.stepSendResponse))

		smVFindCallRequest.stepGetRequestData(execCtx)

		require.Equal(t, payload.FoundCall, smVFindCallRequest.status)
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
				require.Equal(t, payload.FoundCall, msg0.Status)
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

	mc.Finish()
}
