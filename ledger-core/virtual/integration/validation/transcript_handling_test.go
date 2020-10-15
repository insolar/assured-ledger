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

	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/referencebuilder/checker"
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

	recordService := checker.NewTypedReferenceBuilder(ctx, t)
	server.ReplaceRecordReferenceBuilderService(recordService)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	prevPulse := server.GetPulse().PulseNumber

	// VObjectTranscriptReport is always from previous pulse
	server.IncrementPulse(ctx)
	curPulse := server.GetPulse().PulseNumber

	// initialize values
	var (
		inboundRequest  = server.RandomGlobalWithPulse()
		inboundResponse = server.RandomGlobalWithPulse()
		rLineMemory     = server.RandomGlobalWithPulse()
		rLineActivate   = server.RandomGlobalWithPulse()
	)
	// add RecordReferenceBuilderMock
	{
		// todo: add more checks
		recordService.ROutboundRequest.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.ROutboundRequest) reference.Global {
			return inboundRequest
		})
		recordService.RInboundResponse.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.RInboundResponse) reference.Global {
			return inboundResponse
		})
		recordService.RLineActivate.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.RLineActivate) reference.Global {
			return rLineActivate
		})
		recordService.RLineMemory.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.RLineMemory) reference.Global {
			return rLineMemory
		})
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	plWrapper := utils.GenerateVCallRequestConstructor(server)
	callRequest := plWrapper.Get()
	outgoing := callRequest.CallOutgoing
	objectRef := plWrapper.GetObject()

	// add runnerMock
	{
		requestResult := requestresult.New([]byte("call result"), objectRef)
		requestResult.SetActivate(server.RandomGlobalWithPulse(), []byte("init state"))
		runnerMock.AddExecutionMock(outgoing.GetValue()).AddStart(
			func(ctx execution.Context) {
				assert.Empty(t, ctx.ObjectDescriptor)
				assertExecutionContext(t, ctx, &callRequest, objectRef, curPulse)
			},
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
			require.Equal(t, curPulse, report.In)
			require.Equal(t, rLineActivate, report.Validated.GetValue())

			return false
		})
	}

	// send VObjectTranscriptReport
	{
		pl := rms.VObjectTranscriptReport{
			AsOf:   prevPulse,
			Object: rms.NewReference(objectRef),
			ObjectTranscript: rms.Transcript{
				Entries: []rms.Any{
					rms.NewAny(
						&rms.Transcript_TranscriptEntryIncomingRequest{
							Request:  callRequest,
							Incoming: rms.NewReference(inboundRequest),
						},
					),
					rms.NewAny(
						&rms.Transcript_TranscriptEntryIncomingResult{
							IncomingResult: rms.NewReference(inboundResponse),
							ObjectState:    rms.NewReference(rLineActivate),
							Reason:         callRequest.CallOutgoing,
						},
					),
				},
			},
		}

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

	recordService := checker.NewTypedReferenceBuilder(ctx, t)
	server.ReplaceRecordReferenceBuilderService(recordService)

	server.Init(ctx)

	prevPulse := server.GetPulse().PulseNumber

	// VObjectTranscriptReport is always from previous pulse
	server.IncrementPulse(ctx)
	curPulse := server.GetPulse().PulseNumber

	// initialize values
	var (
		inboundRequest  = server.RandomGlobalWithPulse()
		inboundResponse = server.RandomGlobalWithPulse()
		rLineMemory     = server.RandomGlobalWithPulse()
	)
	// add RecordReferenceBuilderMock
	{
		// todo: add more checks
		recordService.ROutboundRequest.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.ROutboundRequest) reference.Global {
			return inboundRequest
		})
		recordService.RInboundResponse.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.RInboundResponse) reference.Global {
			return inboundResponse
		})
		recordService.RLineMemory.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.RLineMemory) reference.Global {
			return rLineMemory
		})
	}

	callRequest := utils.GenerateVCallRequestMethod(server)
	objectRef := callRequest.Callee.GetValue()
	classRef := server.RandomGlobalWithPulse()
	outgoing := callRequest.CallOutgoing

	objDescriptor := descriptor.NewObject(objectRef, server.RandomLocalWithPulse(), classRef, []byte("init state"), false)

	// add typedChecker
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCachedMemoryRequest.Set(func(req *rms.VCachedMemoryRequest) bool {
			require.Equal(t, objectRef, req.Object.GetValue())
			require.Equal(t, objDescriptor.State().GetLocal(), req.State.GetValue().GetLocal())

			pl := &rms.VCachedMemoryResponse{
				CallStatus: rms.CachedMemoryStateFound,
				State: rms.ObjectState{
					Reference: req.State,
					Class:     rms.NewReference(classRef),
					Memory:    rms.NewBytes([]byte("init state")),
				},
			}
			server.SendPayload(ctx, pl)
			return false
		})
		typedChecker.VObjectValidationReport.Set(func(report *rms.VObjectValidationReport) bool {
			require.Equal(t, objectRef, report.Object.GetValue())
			require.Equal(t, curPulse, report.In)
			require.Equal(t, rLineMemory, report.Validated.GetValue())

			return false
		})
	}

	// add runnerMock
	{
		requestResult := requestresult.New([]byte("call result"), objectRef)
		requestResult.SetAmend(objDescriptor, []byte("new state"))
		runnerMock.AddExecutionMock(outgoing.GetValue()).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, objDescriptor.State().GetLocal(), ctx.ObjectDescriptor.State().GetLocal())
				assertDescriptor(t, ctx, curPulse)
				assertExecutionContext(t, ctx, callRequest, objectRef, curPulse)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
	}

	// send VObjectTranscriptReport
	{
		pl := rms.VObjectTranscriptReport{
			AsOf:   prevPulse,
			Object: rms.NewReference(objectRef),
			ObjectTranscript: rms.Transcript{
				Entries: []rms.Any{
					rms.NewAny(
						&rms.Transcript_TranscriptEntryIncomingRequest{
							Incoming:     rms.NewReference(inboundRequest),
							ObjectMemory: rms.NewReference(objDescriptor.State()),
							Request:      *callRequest,
						},
					),
					rms.NewAny(
						&rms.Transcript_TranscriptEntryIncomingResult{
							IncomingResult: rms.NewReference(inboundResponse),
							ObjectState:    rms.NewReference(rLineMemory),
							Reason:         callRequest.CallOutgoing,
						},
					),
				},
			},
		}

		done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
		server.SendPayload(ctx, &pl)
		commontestutils.WaitSignalsTimed(t, 60*time.Second, done)
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

	recordService := checker.NewTypedReferenceBuilder(ctx, t)
	server.ReplaceRecordReferenceBuilderService(recordService)

	server.Init(ctx)

	prevPulse := server.GetPulse().PulseNumber

	// VObjectTranscriptReport is always from previous pulse
	server.IncrementPulse(ctx)
	curPulse := server.GetPulse().PulseNumber

	// initialize values
	var (
		rLineStart       = server.RandomGlobalWithPulse()
		inboundResponse  = server.RandomGlobalWithPulse()
		outboundRequest  = server.RandomGlobalWithPulse()
		outboundResponse = server.RandomGlobalWithPulse()
		rLineMemory      = server.RandomGlobalWithPulse()
		rLineActivate    = server.RandomGlobalWithPulse()
	)
	// add RecordReferenceBuilderMock
	{
		counter := 0
		// todo: add more checks
		recordService.ROutboundRequest.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.ROutboundRequest) reference.Global {
			switch counter {
			// todo: so many times? wtf?
			case 0, 1, 2, 3:
				assert.Equal(t, rms.CallTypeConstructor, record.CallType)

				counter++
				return rLineStart
			case 4:
				assert.Equal(t, rms.CallTypeMethod, record.CallType)

				counter++
				return outboundRequest
			default:
				t.Fatal("unexpected case")
				return reference.Global{}
			}
			// fixme: different refs
			// switch record.Polymorph {
			// case rms.TypeRInboundRequestPolymorphID:
			// 	return inboundRequest
			// case rms.TypeROutboundRequestPolymorphID:
			// 	return outboundRequest
			// default:
			// 	t.Fatalf("unexpected type: %v", record.Polymorph)
			// 	return reference.Global{}
			// }
		})
		recordService.RInboundResponse.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.RInboundResponse) reference.Global {
			return inboundResponse
		})
		recordService.ROutboundResponse.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.ROutboundResponse) reference.Global {
			return outboundResponse
		})
		recordService.RLineActivate.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.RLineActivate) reference.Global {
			return rLineActivate
		})
		recordService.RLineMemory.AnticipatedRefFromBytesMock(func(object reference.Global, pn pulse.Number, record *rms.RLineMemory) reference.Global {
			return rLineMemory
		})
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	plWrapper := utils.GenerateVCallRequestConstructor(server)
	callRequest := plWrapper.Get()
	outgoing := callRequest.CallOutgoing.GetValue()
	objectRef := plWrapper.GetObject()
	class := callRequest.Caller.GetValue()

	calledObjectRef := server.RandomGlobalWithPulse()

	// add runnerMock
	{
		outgoingCall := execution.NewRPCBuilder(outgoing, objectRef).
			CallMethod(calledObjectRef, class, "Bar", []byte("123"))

		requestResult := requestresult.New([]byte("call result"), objectRef)
		requestResult.SetActivate(server.RandomGlobalWithPulse(), []byte("init state"))
		runnerMock.AddExecutionMock(outgoing).AddStart(func(ctx execution.Context) {
			assert.Empty(t, ctx.ObjectDescriptor)
			assertExecutionContext(t, ctx, &callRequest, objectRef, curPulse)
		}, &execution.Update{
			Type:     execution.OutgoingCall,
			Error:    nil,
			Outgoing: outgoingCall,
		}).AddContinue(func(result []byte) {
			assert.Equal(t, []byte("finish B.Bar"), result)
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})

	}

	// add typedChecker
	{
		typedChecker.VObjectValidationReport.Set(func(report *rms.VObjectValidationReport) bool {
			require.Equal(t, objectRef, report.Object.GetValue())
			require.Equal(t, curPulse, report.In)
			require.Equal(t, rLineActivate, report.Validated.GetValue())

			return false
		})
	}

	// send VObjectTranscriptReport
	{
		pl := rms.VObjectTranscriptReport{
			AsOf:   prevPulse,
			Object: rms.NewReference(objectRef),
			ObjectTranscript: rms.Transcript{
				Entries: []rms.Any{{}, {}, {}, {}},
			},
		}
		pl.ObjectTranscript.Entries[0].Set(
			&rms.Transcript_TranscriptEntryIncomingRequest{
				Request:  callRequest,
				Incoming: rms.NewReference(rLineStart),
			},
		)
		pl.ObjectTranscript.Entries[1].Set(
			&rms.Transcript_TranscriptEntryOutgoingRequest{
				Request: rms.NewReference(outboundRequest),
				Reason:  callRequest.CallOutgoing,
			},
		)
		pl.ObjectTranscript.Entries[2].Set(
			&rms.Transcript_TranscriptEntryOutgoingResult{
				OutgoingResult: rms.NewReference(outboundResponse),
				CallResult: rms.VCallResult{
					CallType:        rms.CallTypeMethod,
					CallFlags:       rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
					Caller:          rms.NewReference(objectRef),
					Callee:          rms.NewReference(class),
					CallOutgoing:    rms.NewReference(outboundRequest),
					ReturnArguments: rms.NewBytes([]byte("finish B.Bar")),
				},
				Reason: callRequest.CallOutgoing,
			},
		)
		pl.ObjectTranscript.Entries[3].Set(
			&rms.Transcript_TranscriptEntryIncomingResult{
				IncomingResult: rms.NewReference(inboundResponse),
				ObjectState:    rms.NewReference(rLineActivate),
				Reason:         callRequest.CallOutgoing,
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

func TestValidation_ObjectTranscriptReport_AfterTwoInterleaving(t *testing.T) {
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

	callRequest1 := utils.GenerateVCallRequestMethod(server)
	outgoing1 := callRequest1.CallOutgoing.GetValue()
	objectRef := callRequest1.Callee.GetValue()
	classRef := server.RandomGlobalWithPulse()
	p := server.GetPulse().PulseNumber

	callRequest2 := &rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		CallFlags:      rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
		Caller:         rms.NewReference(server.GlobalCaller()),
		Callee:         rms.NewReference(objectRef),
		CallSiteMethod: "GetMethod",
		CallSequence:   1,
		CallOutgoing:   rms.NewReference(server.BuildRandomOutgoingWithPulse()),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}
	outgoing2 := callRequest2.CallOutgoing.GetValue()

	stateId := server.RandomLocalWithPulse()
	stateRef := reference.NewRecordOf(objectRef, stateId)
	objDescriptor := descriptor.NewObject(objectRef, stateId, classRef, []byte("init state"), false)

	newStateRef := reference.NewRecordOf(objectRef, server.RandomLocalWithPulse())

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCachedMemoryRequest.Set(func(req *rms.VCachedMemoryRequest) bool {
			require.Equal(t, objectRef, req.Object.GetValue())
			require.Equal(t, objDescriptor.State(), req.State.GetValue())

			pl := &rms.VCachedMemoryResponse{
				CallStatus: rms.CachedMemoryStateFound,
				State: rms.ObjectState{
					Reference: req.State,
					Class:     rms.NewReference(classRef),
					Memory:    rms.NewBytes(objDescriptor.Memory()),
				},
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
		requestResult := requestresult.New([]byte("call1 result"), objectRef)
		requestResult.SetAmend(objDescriptor, []byte("new state"))
		runnerMock.AddExecutionMock(outgoing1).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, stateRef, ctx.ObjectDescriptor.State())
				assertDescriptor(t, ctx, p)
				assertExecutionContext(t, ctx, callRequest1, objectRef, p)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
		runnerMock.AddExecutionMock(outgoing2).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, stateRef, ctx.ObjectDescriptor.State())
				assertDescriptor(t, ctx, p)
				assertExecutionContext(t, ctx, callRequest2, objectRef, p)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("call2 result"), objectRef),
			},
		)
	}

	// send VObjectTranscriptReport
	{
		pl := rms.VObjectTranscriptReport{
			AsOf:   p,
			Object: rms.NewReference(objectRef),
			ObjectTranscript: rms.Transcript{
				Entries: []rms.Any{{}, {}, {}, {}},
			},
		}
		pl.ObjectTranscript.Entries[0].Set(
			&rms.Transcript_TranscriptEntryIncomingRequest{
				ObjectMemory: rms.NewReference(stateRef),
				Request:      *callRequest1,
			},
		)
		pl.ObjectTranscript.Entries[1].Set(
			&rms.Transcript_TranscriptEntryIncomingRequest{
				ObjectMemory: rms.NewReference(stateRef),
				Request:      *callRequest2,
			},
		)
		pl.ObjectTranscript.Entries[2].Set(
			&rms.Transcript_TranscriptEntryIncomingResult{
				ObjectState: rms.NewReference(newStateRef),
				Reason:      callRequest1.CallOutgoing,
			},
		)
		pl.ObjectTranscript.Entries[3].Set(
			&rms.Transcript_TranscriptEntryIncomingResult{
				ObjectState: rms.NewReference(stateRef),
				Reason:      callRequest2.CallOutgoing,
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

func TestValidation_ObjectTranscriptReport_AfterTwoSequential(t *testing.T) {
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

	callRequest1 := utils.GenerateVCallRequestMethod(server)
	outgoing1 := callRequest1.CallOutgoing.GetValue()
	objectRef := callRequest1.Callee.GetValue()
	classRef := server.RandomGlobalWithPulse()
	p := server.GetPulse().PulseNumber

	callRequest2 := &rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		CallFlags:      rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
		Caller:         rms.NewReference(server.GlobalCaller()),
		Callee:         rms.NewReference(objectRef),
		CallSiteMethod: "GetMethod",
		CallSequence:   1,
		CallOutgoing:   rms.NewReference(server.BuildRandomOutgoingWithPulse()),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}
	outgoing2 := callRequest2.CallOutgoing.GetValue()

	stateId := server.RandomLocalWithPulse()
	stateRef := reference.NewRecordOf(objectRef, stateId)
	objDescriptor := descriptor.NewObject(objectRef, stateId, classRef, []byte("init state"), false)

	newStateRef := reference.NewRecordOf(objectRef, server.RandomLocalWithPulse())

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCachedMemoryRequest.Set(func(req *rms.VCachedMemoryRequest) bool {
			require.Equal(t, objectRef, req.Object.GetValue())
			require.Equal(t, objDescriptor.State(), req.State.GetValue())

			pl := &rms.VCachedMemoryResponse{
				CallStatus: rms.CachedMemoryStateFound,
				State: rms.ObjectState{
					Reference: req.State,
					Class:     rms.NewReference(classRef),
					Memory:    rms.NewBytes(objDescriptor.Memory()),
				},
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
		requestResult := requestresult.New([]byte("call1 result"), objectRef)
		requestResult.SetAmend(objDescriptor, []byte("new state"))
		runnerMock.AddExecutionMock(outgoing1).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, stateRef, ctx.ObjectDescriptor.State())
				assertDescriptor(t, ctx, p)
				assertExecutionContext(t, ctx, callRequest1, objectRef, p)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
		runnerMock.AddExecutionMock(outgoing2).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, stateRef, ctx.ObjectDescriptor.State())
				assertDescriptor(t, ctx, p)
				assertExecutionContext(t, ctx, callRequest2, objectRef, p)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("call2 result"), objectRef),
			},
		)
	}

	// send VObjectTranscriptReport
	{
		pl := rms.VObjectTranscriptReport{
			AsOf:   p,
			Object: rms.NewReference(objectRef),
			ObjectTranscript: rms.Transcript{
				Entries: []rms.Any{{}, {}, {}, {}},
			},
		}
		pl.ObjectTranscript.Entries[0].Set(
			&rms.Transcript_TranscriptEntryIncomingRequest{
				ObjectMemory: rms.NewReference(stateRef),
				Request:      *callRequest1,
			},
		)
		pl.ObjectTranscript.Entries[1].Set(
			&rms.Transcript_TranscriptEntryIncomingResult{
				ObjectState: rms.NewReference(newStateRef),
				Reason:      callRequest1.CallOutgoing,
			},
		)
		pl.ObjectTranscript.Entries[2].Set(
			&rms.Transcript_TranscriptEntryIncomingRequest{
				ObjectMemory: rms.NewReference(stateRef),
				Request:      *callRequest2,
			},
		)
		pl.ObjectTranscript.Entries[3].Set(
			&rms.Transcript_TranscriptEntryIncomingResult{
				ObjectState: rms.NewReference(stateRef),
				Reason:      callRequest2.CallOutgoing,
			},
		)

		done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
		server.SendPayload(ctx, &pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	assert.Equal(t, 1, typedChecker.VCachedMemoryRequest.Count())
	assert.Equal(t, 1, typedChecker.VObjectValidationReport.Count())

	mc.Finish()
}

func TestValidation_ObjectTranscriptReport_WithPending(t *testing.T) {
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

	var (
		callRequest     *rms.VCallRequest
		outgoing        rms.Reference
		pendingRequest  *rms.VCallRequest
		pendingOutgoing rms.Reference

		objectRef                                                      reference.Global
		classRef                                                       reference.Global
		objDescriptor                                                  descriptor.Object
		initStateRef, pendingFinishedStateRef, requestFinishedStateRef reference.Global
	)

	// prepare requests and memory
	{
		pendingRequest = utils.GenerateVCallRequestMethod(server)
		objectRef = pendingRequest.Callee.GetValue()
		classRef = server.RandomGlobalWithPulse()
		pendingOutgoing = pendingRequest.CallOutgoing

		stateId := server.RandomLocalWithPulse()
		initStateRef = reference.NewRecordOf(objectRef, stateId)
		objDescriptor = descriptor.NewObject(objectRef, stateId, classRef, []byte("init state"), false)

		pendingFinishedStateRef = reference.NewRecordOf(objectRef, server.RandomLocalWithPulse())

		server.IncrementPulse(ctx)

		callRequest = utils.GenerateVCallRequestMethod(server)
		callRequest.Callee.Set(objectRef)
		outgoing = callRequest.CallOutgoing

		requestFinishedStateRef = reference.NewRecordOf(objectRef, server.RandomLocalWithPulse())
	}

	currentPulse := server.GetPulse().PulseNumber

	// add typedChecker
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCachedMemoryRequest.Set(func(req *rms.VCachedMemoryRequest) bool {
			assert.Equal(t, objectRef, req.Object.GetValue())

			var memory []byte
			switch req.State.GetValue() {
			case initStateRef:
				memory = []byte("init state")
			case pendingFinishedStateRef:
				memory = []byte("after pending state")
			default:
				t.Fatalf("unexpected stateRef")
			}
			pl := &rms.VCachedMemoryResponse{
				CallStatus: rms.CachedMemoryStateFound,
				State: rms.ObjectState{
					Reference: req.State,
					Class:     rms.NewReference(classRef),
					Memory:    rms.NewBytes(memory),
				},
			}
			server.SendPayload(ctx, pl)
			return false
		})
		typedChecker.VObjectValidationReport.Set(func(report *rms.VObjectValidationReport) bool {
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, currentPulse, report.In)
			assert.Equal(t, requestFinishedStateRef, report.Validated.GetValue())

			return false
		})
	}

	// add runnerMock
	{
		requestResult := requestresult.New([]byte("call result"), objectRef)
		requestResult.SetAmend(objDescriptor, []byte("latest state"))
		runnerMock.AddExecutionMock(outgoing.GetValue()).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, []byte("after pending state"), ctx.ObjectDescriptor.Memory())
				assert.Equal(t, pendingFinishedStateRef, ctx.ObjectDescriptor.State())
				pulseOfState := pendingFinishedStateRef.GetLocal().Pulse()
				assertDescriptor(t, ctx, pulseOfState)
				assertExecutionContext(t, ctx, callRequest, objectRef, currentPulse)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
		pendingResult := requestresult.New([]byte("call result"), objectRef)
		pendingResult.SetAmend(objDescriptor, []byte("after pending state"))
		runnerMock.AddExecutionMock(pendingOutgoing.GetValue()).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, []byte("init state"), ctx.ObjectDescriptor.Memory())
				assert.Equal(t, initStateRef, ctx.ObjectDescriptor.State())
				pulseOfState := initStateRef.GetLocal().Pulse()

				assertDescriptor(t, ctx, pulseOfState)
				assertExecutionContext(t, ctx, pendingRequest, objectRef, currentPulse)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: pendingResult,
			},
		)
	}

	// send VObjectTranscriptReport
	{
		pl := rms.VObjectTranscriptReport{
			AsOf:   currentPulse,
			Object: rms.NewReference(objectRef),
			PendingTranscripts: []rms.Transcript{
				{
					Entries: []rms.Any{{}, {}},
				},
			},
			ObjectTranscript: rms.Transcript{
				Entries: []rms.Any{{}, {}},
			},
		}
		pl.PendingTranscripts[0].Entries[0].Set(
			&rms.Transcript_TranscriptEntryIncomingRequest{
				ObjectMemory: rms.NewReference(initStateRef),
				Request:      *pendingRequest,
			},
		)
		pl.PendingTranscripts[0].Entries[1].Set(
			&rms.Transcript_TranscriptEntryIncomingResult{
				ObjectState: rms.NewReference(pendingFinishedStateRef),
				Reason:      pendingRequest.CallOutgoing,
			},
		)
		pl.ObjectTranscript.Entries[0].Set(
			&rms.Transcript_TranscriptEntryIncomingRequest{
				ObjectMemory: rms.NewReference(pendingFinishedStateRef),
				Request:      *callRequest,
			},
		)
		pl.ObjectTranscript.Entries[1].Set(
			&rms.Transcript_TranscriptEntryIncomingResult{
				ObjectState: rms.NewReference(requestFinishedStateRef),
				Reason:      callRequest.CallOutgoing,
			},
		)

		done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
		server.SendPayload(ctx, &pl)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

		assert.Equal(t, 2, typedChecker.VCachedMemoryRequest.Count())
		assert.Equal(t, 1, typedChecker.VObjectValidationReport.Count())
	}

	mc.Finish()
}

func assertExecutionContext(t *testing.T, ctx execution.Context, req *rms.VCallRequest, objectRef reference.Global, p pulse.Number) {
	requestPulse := req.CallOutgoing.GetPulseOfLocal()

	utils.AssertVCallRequestEqual(t, req, ctx.Request)
	assert.NotEmpty(t, ctx.Context)
	assert.Empty(t, ctx.Result)
	assert.Equal(t, uint32(0), ctx.Sequence)

	assert.Equal(t, p, ctx.Pulse.PulseNumber)
	assert.NotEmpty(t, ctx.Pulse.DataExt)
	assert.Equal(t, p.AsEpoch(), ctx.Pulse.DataExt.PulseEpoch)

	assert.Equal(t, objectRef, ctx.Object)
	assert.Equal(t, requestPulse, ctx.Incoming.GetLocal().Pulse())

	assert.Equal(t, req.CallOutgoing.GetValue(), ctx.Outgoing)

	assert.Equal(t, req.CallFlags.GetInterference(), ctx.Isolation.Interference)
	assert.Equal(t, req.CallFlags.GetState(), ctx.Isolation.State)
	assert.Empty(t, ctx.LogicContext)
}

func assertDescriptor(t *testing.T, ctx execution.Context, p pulse.Number) {
	assert.Equal(t, p, ctx.ObjectDescriptor.HeadRef().GetLocal().Pulse())
	assert.Equal(t, p, ctx.ObjectDescriptor.State().GetLocal().Pulse())
	assert.False(t, ctx.ObjectDescriptor.Deactivated())
	assert.NotEmpty(t, ctx.ObjectDescriptor.Memory())
	// class, err := ctx.ObjectDescriptor.Class()
	// assert.NoError(t, err)
	// assert.Equal(t, objectRef, class)
}
