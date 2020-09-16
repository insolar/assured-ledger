// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package validation

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestValidation_ObjectTranscriptReport_AfterConstructor(t *testing.T) {
	defer commontestutils.LeakTester(t)
	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	authService := authentication.NewServiceMock(t)
	authService.CheckMessageFromAuthorizedVirtualMock.Return(false, nil)
	server.ReplaceAuthenticationService(authService)

	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	callRequest := utils.GenerateVCallRequestConstructor(server)
	outgoing := callRequest.CallOutgoing
	objectRef := reference.NewSelf(outgoing.GetValue().GetLocal())
	p := server.GetPulse().PulseNumber

	pl := rms.VObjectTranscriptReport{
		AsOf:   p,
		Object: rms.NewReference(objectRef),
		ObjectTranscript: rms.VObjectTranscriptReport_Transcript{
			Entries: []rms.Any{
				{},
				{},
			},
		},
	}
	pl.ObjectTranscript.Entries[0].Set(
		&rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest{
			Request:      *callRequest,
		},
	)
	pl.ObjectTranscript.Entries[1].Set(
		&rms.VObjectTranscriptReport_TranscriptEntryIncomingResult{
		},
	)

	done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
	server.SendPayload(ctx, &pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())

	mc.Finish()
}

func TestValidation_ObjectTranscriptReport_AfterMethod(t *testing.T) {
	defer commontestutils.LeakTester(t)
	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	authService := authentication.NewServiceMock(t)
	authService.CheckMessageFromAuthorizedVirtualMock.Return(false, nil)
	server.ReplaceAuthenticationService(authService)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)

	callRequest := utils.GenerateVCallRequestMethod(server)
	objectRef := callRequest.Callee
	outgoing := callRequest.CallOutgoing
	p := server.GetPulse().PulseNumber

	objDescriptor := descriptor.NewObject(objectRef.GetValue(), server.RandomLocalWithPulse(), server.RandomGlobalWithPulse(), []byte("init state"), false)
	stateRef := reference.NewRecordOf(objDescriptor.HeadRef(), objDescriptor.StateID())
	var newStateRef reference.Global

	// add typedChecker
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCachedMemoryRequest.Set(func(report *rms.VCachedMemoryRequest) bool {
			require.Equal(t, stateRef.GetLocal(), report.StateID.GetValue().GetLocal())

			pl := &rms.VCachedMemoryResponse{
				Object:     report.Object,
				StateID:    report.StateID,
				CallStatus: rms.CachedMemoryStateFound,
				Memory:     rms.NewBytes(objDescriptor.Memory()),
			}
			server.SendPayload(ctx, pl)
			return false
		})
	}

	// add runnerMock
	{
		newObjDescr := descriptor.NewObject(objectRef.GetValue(), server.RandomLocalWithPulse(), server.RandomGlobalWithPulse(), []byte("new state"), false)
		newStateRef = reference.NewRecordOf(newObjDescr.HeadRef(), newObjDescr.StateID())
		requestResult := requestresult.New([]byte("call result"), server.RandomGlobalWithPulse())
		requestResult.SetAmend(objDescriptor, []byte("new state"))
		runnerMock.AddExecutionClassify(outgoing.GetValue(), contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty}, nil)
		runnerMock.AddExecutionMock(outgoing.GetValue()).AddStart(
			nil,
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
	}

	// send VObjectTranscriptReport
	{
		pl := rms.VObjectTranscriptReport{
			AsOf:   p,
			Object: objectRef,
			ObjectTranscript: rms.VObjectTranscriptReport_Transcript{
				Entries: []rms.Any{{}, {}},
			},
		}
		pl.ObjectTranscript.Entries[0].Set(
			&rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest{
				ObjectMemory: rms.NewReference(stateRef),
				Request:      *callRequest,
			},
		)
		pl.ObjectTranscript.Entries[1].Set(
			&rms.VObjectTranscriptReport_TranscriptEntryIncomingResult{
				IncomingResult: rms.NewReference(reference.Global{}),
				ObjectState:    rms.NewReference(newStateRef),
			},
		)

		done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
		server.SendPayload(ctx, &pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

		assert.Equal(t, 1, typedChecker.VCachedMemoryRequest.Count())
	}

	mc.Finish()
}
