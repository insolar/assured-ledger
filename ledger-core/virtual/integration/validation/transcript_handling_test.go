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

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
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

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	callRequest := utils.GenerateVCallRequestConstructor(server)
	outgoing := callRequest.CallOutgoing
	objectRef := reference.NewSelf(outgoing.GetValue().GetLocal())
	p := server.GetPulse().PulseNumber

	stateHash := append([]byte("init state"), objectRef.AsBytes()...)
	stateID := execute.NewStateID(p, stateHash)
	stateRef := reference.NewRecordOf(objectRef, stateID)

	// add runnerMock
	{
		requestResult := requestresult.New([]byte("call result"), objectRef)
		requestResult.SetActivate(server.RandomGlobalWithPulse(), []byte("init state"))
		runnerMock.AddExecutionMock(outgoing.GetValue()).AddStart(
			nil,
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
	}

	// add typedChecker
	{
		typedChecker.VObjectValidationReport.Set(func(report *rms.VObjectValidationReport) bool {
			require.Equal(t, objectRef, report.Object.GetValue())
			require.Equal(t, p, report.In)
			require.Equal(t, stateRef, report.Validated.GetValue())

			return false
		})
	}

	// send VObjectTranscriptReport
	{
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
				Request: *callRequest,
			},
		)
		pl.ObjectTranscript.Entries[1].Set(
			&rms.VObjectTranscriptReport_TranscriptEntryIncomingResult{
				ObjectState: rms.NewReference(stateRef),
			},
		)

		done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
		server.SendPayload(ctx, &pl)

		commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectValidationReport.Wait(ctx, 1))

		assert.Equal(t, 1, typedChecker.VObjectValidationReport.Count())
	}

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
	objectRef := callRequest.Callee.GetValue()
	classRef := server.RandomGlobalWithPulse()
	outgoing := callRequest.CallOutgoing
	p := server.GetPulse().PulseNumber

	stateId := server.RandomLocalWithPulse()
	stateRef := reference.NewRecordOf(objectRef, stateId)
	objDescriptor := descriptor.NewObject(objectRef, stateId, classRef, []byte("init state"), false)

	newStateHash := append([]byte("new state"), objectRef.AsBytes()...)
	newStateHash = append(newStateHash, objDescriptor.StateID().AsBytes()...)
	newStateID := execute.NewStateID(p, newStateHash)
	newStateRef := reference.NewRecordOf(objectRef, newStateID)

	// add typedChecker
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCachedMemoryRequest.Set(func(report *rms.VCachedMemoryRequest) bool {
			require.Equal(t, objectRef, report.Object.GetValue())
			require.Equal(t, objDescriptor.StateID(), report.StateID.GetValueWithoutBase())

			pl := &rms.VCachedMemoryResponse{
				Object:     report.Object,
				StateID:    report.StateID,
				CallStatus: rms.CachedMemoryStateFound,
				Memory:     rms.NewBytes(objDescriptor.Memory()),
			}
			server.SendPayload(ctx, pl)
			return false
		})
		typedChecker.VObjectValidationReport.Set(func(report *rms.VObjectValidationReport) bool {
			require.Equal(t, objectRef, report.Object.GetValue())
			require.Equal(t, p, report.In)
			require.Equal(t, newStateRef, report.Validated.GetValue())

			return false
		})
	}

	// add runnerMock
	{
		requestResult := requestresult.New([]byte("call result"), objectRef)
		requestResult.SetAmend(objDescriptor, []byte("new state"))
		// runnerMock.AddExecutionClassify(outgoing.GetValue(), contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty}, nil)
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
			Object: rms.NewReference(objectRef),
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
				ObjectState: rms.NewReference(newStateRef),
			},
		)

		done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
		server.SendPayload(ctx, &pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

		assert.Equal(t, 1, typedChecker.VCachedMemoryRequest.Count())
		assert.Equal(t, 1, typedChecker.VObjectValidationReport.Count())
	}

	mc.Finish()
}

func TestValidation_ObjectTranscriptReport_AfterConstructorWithOutgoing(t *testing.T) {
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

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	callRequest := utils.GenerateVCallRequestConstructor(server)
	outgoing := callRequest.CallOutgoing
	objectRef := reference.NewSelf(outgoing.GetValue().GetLocal())
	p := server.GetPulse().PulseNumber

	stateHash := append([]byte("init state"), objectRef.AsBytes()...)
	stateID := execute.NewStateID(p, stateHash)
	stateRef := reference.NewRecordOf(objectRef, stateID)

	outgoingRefFromConstructor := reference.NewRecordOf(objectRef, server.RandomLocalWithPulse())

	// add runnerMock
	{
		requestResult := requestresult.New([]byte("call result"), objectRef)
		requestResult.SetActivate(server.RandomGlobalWithPulse(), []byte("init state"))
		runnerMock.AddExecutionMock(outgoing.GetValue()).AddStart(
			nil,
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
	}

	// add typedChecker
	{
		typedChecker.VObjectValidationReport.Set(func(report *rms.VObjectValidationReport) bool {
			require.Equal(t, objectRef, report.Object.GetValue())
			require.Equal(t, p, report.In)
			require.Equal(t, stateRef, report.Validated.GetValue())

			return false
		})
	}

	// send VObjectTranscriptReport
	{
		pl := rms.VObjectTranscriptReport{
			AsOf:   p,
			Object: rms.NewReference(objectRef),
			ObjectTranscript: rms.VObjectTranscriptReport_Transcript{
				Entries: []rms.Any{{}, {}, {}, {}},
			},
		}
		pl.ObjectTranscript.Entries[0].Set(
			&rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest{
				Request: *callRequest,
			},
		)
		pl.ObjectTranscript.Entries[1].Set(
			&rms.VObjectTranscriptReport_TranscriptEntryOutgoingRequest{
				Outgoing: rms.NewReference(outgoingRefFromConstructor),
			},
		)
		pl.ObjectTranscript.Entries[2].Set(
			&rms.VObjectTranscriptReport_TranscriptEntryOutgoingResult{
				// todo, fixme: here should be complete reply VCallResult
			},
		)
		pl.ObjectTranscript.Entries[3].Set(
			&rms.VObjectTranscriptReport_TranscriptEntryIncomingResult{
				ObjectState: rms.NewReference(stateRef),
			},
		)

		done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
		server.SendPayload(ctx, &pl)

		commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectValidationReport.Wait(ctx, 1))

		assert.Equal(t, 1, typedChecker.VObjectValidationReport.Count())
	}

	mc.Finish()
}
