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

	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_DeactivateObject(t *testing.T) {
	defer commonTestUtils.LeakTester(t)
	insrail.LogCase(t, "C5134")

	table := []struct {
		name                string
		stateIsEqual        bool
		dirtyIsDeactivated  bool
		entirelyDeactivated bool
	}{
		{name: "ValidatedState==DirtyState and both states are active", stateIsEqual: true, dirtyIsDeactivated: false, entirelyDeactivated: false},
		{name: "ValidatedState!=DirtyState and DirtyState is deactivated", stateIsEqual: false, dirtyIsDeactivated: true, entirelyDeactivated: false},
		{name: "both states are deactivated", stateIsEqual: true, dirtyIsDeactivated: true, entirelyDeactivated: true},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, t, nil)
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				objectGlobal   = server.RandomGlobalWithPulse()
				validatedState = []byte("initial state")
				dirtyState     = validatedState
			)
			if !test.stateIsEqual {
				dirtyState = []byte("dirty state")
			}

			builder := server.StateReportBuilder().Object(objectGlobal)

			server.IncrementPulseAndWaitIdle(ctx)

			// Send VStateReport
			{
				if test.entirelyDeactivated {
					builder = builder.Inactive()
				} else {
					builder = builder.Ready().ValidatedMemory(validatedState).DirtyMemory(dirtyState)
					if test.dirtyIsDeactivated {
						builder = builder.DirtyInactive()
					}
				}

				wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, builder.ReportPtr())
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)
			}

			var (
				outgoingDestroy      = server.BuildRandomOutgoingWithPulse()
				outgoingGetValidated = server.BuildRandomOutgoingWithPulse()
				outgoingGetDirty     = server.BuildRandomOutgoingWithPulse()
			)

			// Execution mock
			{
				// deactivate method
				objectAExecutionMock := runnerMock.AddExecutionMock(outgoingDestroy)
				objectAExecutionMock.AddStart(nil, &execution.Update{
					Type:     execution.OutgoingCall,
					Outgoing: execution.NewRPCBuilder(outgoingDestroy, objectGlobal).Deactivate(),
				},
				).AddContinue(nil, &execution.Update{
					Type:   execution.Done,
					Result: requestresult.New([]byte("deactivate result"), objectGlobal),
				},
				)
				runnerMock.AddExecutionClassify(outgoingDestroy, tolerableFlags(), nil)

				// get methods
				runnerMock.AddExecutionClassify(outgoingGetValidated, intolerableFlags(), nil)
				runnerMock.AddExecutionClassify(outgoingGetDirty, contract.MethodIsolation{Interference: isolation.CallIntolerable, State: isolation.CallDirty}, nil)

				runnerMock.AddExecutionMock(outgoingGetValidated).AddStart(
					func(ctx execution.Context) {
						assert.Equal(t, objectGlobal, ctx.Request.Callee.GetValue())
						assert.Equal(t, validatedState, ctx.ObjectDescriptor.Memory())
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("validated result"), server.RandomGlobalWithPulse()),
					},
				)
			}

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

			// typedChecker mock
			{
				typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
					assert.Equal(t, objectGlobal, report.Object.GetValue())
					assert.Equal(t, rms.StateStatusInactive, report.Status)
					assert.True(t, report.DelegationSpec.IsZero())
					assert.Nil(t, report.ProvidedContent)
					assert.Empty(t, report.LatestValidatedState)
					assert.Empty(t, report.LatestDirtyState)
					assert.Empty(t, report.LatestValidatedCode)
					assert.Empty(t, report.LatestDirtyCode)
					assert.Equal(t, int32(0), report.UnorderedPendingCount)
					assert.Equal(t, int32(0), report.OrderedPendingCount)

					return false
				})
				typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
					assert.Equal(t, objectGlobal, report.Object.GetValue())
					// assert.Equal(t, outgoing.GetLocal().Pulse(), report.AsOf)
					assert.NotEmpty(t, report.ObjectTranscript.Entries) // todo fix assert
					return false
				})
				typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
					assert.Equal(t, objectGlobal, result.Callee.GetValue())

					switch result.CallOutgoing.GetValue() {
					case outgoingDestroy:
						assert.Equal(t, []byte("deactivate result"), result.ReturnArguments.GetBytes())
					case outgoingGetValidated:
						if test.entirelyDeactivated {
							contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments.GetBytes())
							require.NoError(t, sysErr)
							assert.Contains(t, contractErr.Error(), "try to call method on deactivated object")
						} else {
							assert.Equal(t, []byte("validated result"), result.ReturnArguments.GetBytes())
						}
					case outgoingGetDirty:
						contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments.GetBytes())
						require.NoError(t, sysErr)
						assert.Contains(t, contractErr.Error(), "try to call method on deactivated object")
					default:
						t.Fatalf("unexpected outgoing")
					}

					return false
				})
			}

			// Deactivate object if it's active
			{
				if !(test.dirtyIsDeactivated || test.entirelyDeactivated) {
					pl := utils.GenerateVCallRequestMethod(server)
					pl.Callee.Set(objectGlobal)
					pl.CallOutgoing.Set(outgoingDestroy)

					executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
					server.SendPayload(ctx, pl)
					commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
				}
			}

			// VCallRequest get state dirty/validated
			{
				// get validated state
				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated)
				pl.Callee.Set(objectGlobal)
				pl.CallSiteMethod = "GetValidated"
				pl.CallOutgoing.Set(outgoingGetValidated)

				executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)

				// get dirty state
				pl = utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
				pl.Callee.Set(objectGlobal)
				pl.CallSiteMethod = "GetDirty"
				pl.CallOutgoing.Set(outgoingGetDirty)

				executeDone = server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
			}

			// Check VStateReport after pulse change
			{
				server.IncrementPulse(ctx)

				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectTranscriptReport.Wait(ctx, 1))
				assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
				assert.Equal(t, 1, typedChecker.VStateReport.Count())
			}

			mc.Finish()
		})
	}
}

func TestVirtual_CallMethod_On_CompletelyDeactivatedObject(t *testing.T) {
	insrail.LogCase(t, "C4975")
	defer commonTestUtils.LeakTester(t)
	stateTestCases := []struct {
		name        string
		objectState isolation.StateFlag
	}{
		{
			name:        "call on validated state",
			objectState: isolation.CallValidated,
		},
		{
			name:        "call on dirty state",
			objectState: isolation.CallDirty,
		},
	}

	for _, stateTest := range stateTestCases {
		t.Run(stateTest.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, t, nil)

			isolation := contract.MethodIsolation{Interference: isolation.CallIntolerable, State: stateTest.objectState}
			methodName := "MyFavorMethod"
			server.ReplaceRunner(runnerMock)

			server.Init(ctx)

			var (
				object    = server.RandomGlobalWithPulse()
				prevPulse = server.GetPulse().PulseNumber
			)

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
				assert.Equal(t, object, res.Callee.GetValue())
				contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
				assert.NoError(t, sysErr)
				assert.Contains(t, contractErr.Error(), "try to call method on deactivated object")
				return false // no resend msg
			})

			server.IncrementPulseAndWaitIdle(ctx)
			Method_PrepareObject(ctx, server, rms.StateStatusInactive, object, prevPulse)

			outgoing := server.BuildRandomOutgoingWithPulse()
			runnerMock.AddExecutionClassify(outgoing, isolation, nil)

			pl := rms.VCallRequest{
				CallType:       rms.CallTypeMethod,
				CallFlags:      rms.BuildCallFlags(isolation.Interference, isolation.State),
				Caller:         rms.NewReference(server.GlobalCaller()),
				Callee:         rms.NewReference(object),
				CallSiteMethod: methodName,
				CallOutgoing:   rms.NewReference(outgoing),
				Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
			}

			execDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
			server.SendPayload(ctx, &pl)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, execDone)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_CallDeactivate_Intolerable(t *testing.T) {
	insrail.LogCase(t, "C5469")

	table := []struct {
		name  string
		state isolation.StateFlag
	}{
		{name: "ValidatedState", state: isolation.CallValidated},
		{name: "DirtyState", state: isolation.CallDirty},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			defer commonTestUtils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, mc, nil)
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				class        = server.RandomGlobalWithPulse()
				objectGlobal = server.RandomGlobalWithPulse()
			)

			// Create object
			{
				report := server.StateReportBuilder().Object(objectGlobal).Ready().Class(class).Report()
				server.IncrementPulse(ctx)

				wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, &report)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)
			}

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			outgoing := server.BuildRandomOutgoingWithPulse()

			// Add VCallResult check
			{
				typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
					assert.Equal(t, []byte("finish Deactivate"), result.ReturnArguments.GetBytes())

					return false
				})
			}

			// Add executor mock for method `Destroy`
			{
				runnerMock.AddExecutionClassify(outgoing, contract.MethodIsolation{Interference: isolation.CallIntolerable, State: testCase.state}, nil)

				requestResult := requestresult.New([]byte("outgoing call"), objectGlobal)
				requestResult.SetDeactivate(descriptor.NewObject(objectGlobal, server.RandomLocalWithPulse(), class, []byte("initial state"), false))
				runnerMock.AddExecutionMock(outgoing).AddStart(
					nil,
					&execution.Update{
						Type:     execution.OutgoingCall,
						Result:   requestResult,
						Outgoing: execution.NewRPCBuilder(server.BuildRandomOutgoingWithPulse(), objectGlobal).Deactivate(),
					},
				).AddContinue(
					func(result []byte) {
						contractErr, sysErr := foundation.UnmarshalMethodResult(result)
						require.NoError(t, sysErr)
						assert.Equal(t, "interference violation: deactivate call from intolerable call", contractErr.Error())
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("finish Deactivate"), objectGlobal),
					},
				)
			}

			// Deactivate object with wrong callFlags
			{
				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, testCase.state)
				pl.Callee.Set(objectGlobal)
				pl.CallSiteMethod = "Destroy"
				pl.CallOutgoing.Set(outgoing)

				execDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, execDone)
			}

			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
			assert.Equal(t, 1, typedChecker.VCallResult.Count())
			mc.Finish()
		})
	}
}

func TestVirtual_DeactivateObject_ChangePulse(t *testing.T) {
	insrail.LogCase(t, "C5461")

	const (
		origDirtyMem     = "original dirty memory"
		origValidatedMem = "original validated memory"

		deactivateMethodName = "Deactivate"
	)

	defer commonTestUtils.LeakTester(t)
	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)

	oneExecutionEnded := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		class               = server.RandomGlobalWithPulse()
		objectRef           = server.RandomGlobalWithPulse()
		p1                  = server.GetPulse().PulseNumber
		deactivateIsolation = contract.MethodIsolation{
			Interference: isolation.CallTolerable,
			State:        isolation.CallDirty,
		}
	)

	server.IncrementPulseAndWaitIdle(ctx)
	outgoing := server.BuildRandomOutgoingWithPulse()

	synchronizeExecution := synchronization.NewPoint(1)
	{ // setup runner mock for deactivation call
		runnerMock.AddExecutionClassify(outgoing, deactivateIsolation, nil)
		requestResult := requestresult.New([]byte(deactivateMethodName+" result"), objectRef)
		requestResult.SetDeactivate(descriptor.NewObject(objectRef, server.RandomLocalWithPulse(), class, insolar.MustSerialize(initialBalance), false))

		runnerMock.AddExecutionMock(outgoing).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, objectRef, ctx.Request.Callee.GetValue())
				assert.Equal(t, []byte(origDirtyMem), ctx.ObjectDescriptor.Memory())
			},
			&execution.Update{
				Type:     execution.OutgoingCall,
				Result:   requestResult,
				Outgoing: execution.NewRPCBuilder(outgoing, objectRef).Deactivate(),
			},
		).AddContinue(
			func(result []byte) {
				synchronizeExecution.Synchronize()
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("finish Deactivate"), objectRef),
			},
		)
	}

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	{
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			assert.Equal(t, objectRef, res.Callee.GetValue())
			assert.Equal(t, outgoing, res.CallOutgoing.GetValue())
			assert.Equal(t, []byte("finish Deactivate"), res.ReturnArguments.GetBytes())
			return false
		})
		typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, rms.StateStatusReady, report.Status)
			assert.True(t, report.DelegationSpec.IsZero())
			assert.Equal(t, int32(0), report.UnorderedPendingCount)
			assert.Equal(t, int32(1), report.OrderedPendingCount)
			require.NotNil(t, report.ProvidedContent)
			require.NotNil(t, report.ProvidedContent.LatestDirtyState)
			assert.False(t, report.ProvidedContent.LatestDirtyState.Deactivated)
			assert.Equal(t, []byte(origDirtyMem), report.ProvidedContent.LatestDirtyState.Memory.GetBytes())
			require.NotNil(t, report.ProvidedContent.LatestValidatedState)
			assert.False(t, report.ProvidedContent.LatestValidatedState.Deactivated)
			assert.Equal(t, []byte(origDirtyMem), report.ProvidedContent.LatestValidatedState.Memory.GetBytes())
			return false
		})
		typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
			assert.Equal(t, objectRef, report.Object.GetValue())
			assert.Equal(t, outgoing.GetLocal().Pulse(), report.AsOf)
			assert.NotEmpty(t, report.ObjectTranscript.Entries) // todo fix assert
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *rms.VDelegatedCallRequest) bool {
			assert.Equal(t, objectRef, request.Callee.GetValue())
			assert.Equal(t, outgoing, request.CallOutgoing.GetValue())
			token := server.DelegationToken(request.CallOutgoing.GetValue(), server.GlobalCaller(), request.Callee.GetValue())

			response := rms.VDelegatedCallResponse{
				Callee:                 request.Callee,
				CallIncoming:           request.CallIncoming,
				ResponseDelegationSpec: token,
			}
			server.SendPayload(ctx, &response)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *rms.VDelegatedRequestFinished) bool {
			require.NotNil(t, finished.LatestState)
			assert.True(t, finished.LatestState.Deactivated)
			assert.Nil(t, finished.LatestState.Memory.GetBytes())
			return false
		})
	}
	{
		report := utils.NewStateReportBuilder().Pulse(p1).Object(objectRef).Ready().
			Class(class).DirtyMemory([]byte(origDirtyMem)).ValidatedMemory([]byte(origValidatedMem)).Report()

		wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, &report)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)
	}

	{
		pl := utils.GenerateVCallRequestMethod(server)
		pl.Callee.Set(objectRef)
		pl.CallSiteMethod = "Deactivate"
		pl.CallOutgoing.Set(outgoing)

		server.SendPayload(ctx, pl)
	}

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
	server.IncrementPulseAndWaitIdle(ctx)
	synchronizeExecution.WakeUp()
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, oneExecutionEnded)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectTranscriptReport.Wait(ctx, 1))

	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())
	assert.Equal(t, 1, typedChecker.VStateReport.Count())
	assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())
	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	server.Stop()
	mc.Finish()
}

func TestVirtual_CallMethod_After_Deactivation(t *testing.T) {
	insrail.LogCase(t, "C5509")
	defer commonTestUtils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)

	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		class               = server.RandomGlobalWithPulse()
		objectRef           = server.RandomGlobalWithPulse()
		p1                  = server.GetPulse().PulseNumber
		deactivateIsolation = contract.MethodIsolation{
			Interference: isolation.CallTolerable,
			State:        isolation.CallDirty,
		}
	)

	// Create object
	{
		server.IncrementPulseAndWaitIdle(ctx)
		Method_PrepareObject(ctx, server, rms.StateStatusReady, objectRef, p1)
	}

	outgoingDeactivate := server.BuildRandomOutgoingWithPulse()
	outgoingSomeMethod := server.BuildRandomOutgoingWithPulse()
	twoExecutionEnded := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	synchronizeExecution := synchronization.NewPoint(1)
	defer synchronizeExecution.Done()

	// mock
	{
		descr := descriptor.NewObject(objectRef, server.RandomLocalWithPulse(), class, []byte("deactivate state"), false)
		requestResult := requestresult.New([]byte("done"), objectRef)
		requestResult.SetDeactivate(descr)

		objectExecutionMock := runnerMock.AddExecutionMock(outgoingDeactivate)
		objectExecutionMock.AddStart(nil, &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: execution.NewRPCBuilder(server.RandomGlobalWithPulse(), objectRef).Deactivate(),
		}).AddContinue(func(result []byte) {
			synchronizeExecution.Synchronize()
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
		runnerMock.AddExecutionClassify(outgoingDeactivate, deactivateIsolation, nil)
		runnerMock.AddExecutionClassify(outgoingSomeMethod, deactivateIsolation, nil)
	}

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)

	// Add check
	{
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			require.Equal(t, objectRef, res.Callee.GetValue())

			switch res.CallOutgoing.GetValue() {
			case outgoingDeactivate:
				assert.Equal(t, []byte("done"), res.ReturnArguments.GetBytes())
			case outgoingSomeMethod:
				contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
				require.NoError(t, sysErr)
				assert.Contains(t, contractErr.Error(), "try to call method on deactivated object")
				assert.Equal(t, outgoingSomeMethod, res.CallOutgoing.GetValue())
			default:
				assert.Fail(t, "unreachable")
			}

			return false
		})
	}

	// Deactivate
	{
		pl := utils.GenerateVCallRequestMethod(server)
		pl.Callee.Set(objectRef)
		pl.CallSiteMethod = "Destroy"
		pl.CallOutgoing.Set(outgoingDeactivate)

		server.SendPayload(ctx, pl)
	}
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())

	// vCallRequest
	{
		pl := utils.GenerateVCallRequestMethod(server)
		pl.Callee.Set(objectRef)
		pl.CallSiteMethod = "SomeMethod"
		pl.CallOutgoing.Set(outgoingSomeMethod)

		server.SendPayload(ctx, pl)
	}

	time.Sleep(100 * time.Millisecond)
	synchronizeExecution.WakeUp()

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, twoExecutionEnded)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_Deactivation_Deduplicate(t *testing.T) {
	insrail.LogCase(t, "C5507")
	defer commonTestUtils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		outgoingDeactivate = server.BuildRandomOutgoingWithPulse()
		isolation          = contract.MethodIsolation{
			Interference: isolation.CallTolerable,
			State:        isolation.CallDirty,
		}

		// Constructor
		constructorWrapper = utils.GenerateVCallRequestConstructor(server)
		outgoing           = constructorWrapper.GetOutgoing()
		objectRef          = constructorWrapper.GetObject()
		constructorRequest = constructorWrapper.Get()
		class              = constructorRequest.Callee.GetValue()
	)

	// mock
	{
		// Deactivate mock
		descr := descriptor.NewObject(objectRef, server.RandomLocalWithPulse(), class, []byte("deactivate state"), false)
		requestResult := requestresult.New([]byte("deactivated"), objectRef)
		requestResult.SetDeactivate(descr)

		deactivationMock := runnerMock.AddExecutionMock(outgoingDeactivate)
		deactivationMock.AddStart(nil, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
		runnerMock.AddExecutionClassify(outgoingDeactivate, isolation, nil)

		// Constructor mock
		result := requestresult.New([]byte("new"), outgoing)
		result.SetActivate(class, []byte("state"))

		constructorMock := runnerMock.AddExecutionMock(outgoing)
		constructorMock.AddStart(nil, &execution.Update{
			Type:   execution.Done,
			Result: result,
		})
		runnerMock.AddExecutionClassify(outgoing, isolation, nil)
	}

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	// Add check
	{
		typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
			switch res.CallOutgoing.GetValue() {
			case outgoing:
				assert.Equal(t, []byte("new"), res.ReturnArguments.GetBytes())
				assert.Equal(t, objectRef, res.Callee.GetValue())
			case outgoingDeactivate:
				assert.Equal(t, []byte("deactivated"), res.ReturnArguments.GetBytes())
				assert.Equal(t, objectRef, res.Callee.GetValue())
			}
			return false
		})
	}

	// Deactivation
	deactivateRequest := utils.GenerateVCallRequestMethod(server)
	deactivateRequest.Callee.Set(objectRef)
	deactivateRequest.CallSiteMethod = "Destroy"
	deactivateRequest.CallOutgoing.Set(outgoingDeactivate)

	requests := []*rms.VCallRequest{&constructorRequest, deactivateRequest, &constructorRequest, deactivateRequest}
	for _, r := range requests {
		await := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		server.SendPayload(ctx, r)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
	}

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	assert.Equal(t, 4, typedChecker.VCallResult.Count())

	mc.Finish()
}

func TestVirtual_DeduplicateCallAfterDeactivation_PrevVE(t *testing.T) {
	defer commonTestUtils.LeakTester(t)
	insrail.LogCase(t, "C5560")

	table := []struct {
		name  string
		flags contract.MethodIsolation
	}{
		{name: "tolerable+dirty", flags: tolerableFlags()},
		{name: "intolerable+validated", flags: intolerableFlags()},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, t, nil)

			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				objectGlobal      = server.RandomGlobalWithPulse()
				prevPulse         = server.GetPulse().PulseNumber
				outgoingPrevPulse = server.BuildRandomOutgoingWithPulse()
			)

			// Create object
			{
				report := server.StateReportBuilder().Object(objectGlobal).Inactive().Report()
				server.IncrementPulse(ctx)

				wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, &report)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)
			}

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			// Add checker mock
			{
				typedChecker.VFindCallRequest.Set(func(request *rms.VFindCallRequest) bool {
					assert.Equal(t, objectGlobal, request.Callee.GetValue())
					assert.Equal(t, outgoingPrevPulse, request.Outgoing.GetValue())
					assert.Equal(t, prevPulse, request.LookAt)

					pl := &rms.VFindCallResponse{
						LookedAt: request.LookAt,
						Callee:   request.Callee,
						Outgoing: request.Outgoing,
						Status:   rms.CallStateFound,
						CallResult: &rms.VCallResult{
							CallType:        rms.CallTypeMethod,
							CallFlags:       rms.BuildCallFlags(testCase.flags.Interference, testCase.flags.State),
							Callee:          request.Callee,
							Caller:          rms.NewReference(server.GlobalCaller()),
							CallOutgoing:    request.Outgoing,
							CallIncoming:    rms.NewReference(server.RandomGlobalWithPulse()),
							ReturnArguments: rms.NewBytes([]byte("result from past")),
						},
					}
					server.SendPayload(ctx, pl)
					return false
				})
				typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
					assert.Equal(t, objectGlobal, result.Callee.GetValue())
					assert.Equal(t, outgoingPrevPulse, result.CallOutgoing.GetValue())
					assert.Equal(t, []byte("result from past"), result.ReturnArguments.GetBytes())
					return false
				})
			}

			// Add execution mock classify
			runnerMock.AddExecutionClassify(outgoingPrevPulse, testCase.flags, nil)

			// VCallRequest from previous pulse
			{
				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = rms.BuildCallFlags(testCase.flags.Interference, testCase.flags.State)
				pl.Callee.Set(objectGlobal)
				pl.CallSiteMethod = "Foo"
				pl.CallOutgoing.Set(outgoingPrevPulse)

				executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			}

			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VCallResult.Count())
			assert.Equal(t, 1, typedChecker.VFindCallRequest.Count())

			mc.Finish()
		})
	}
}

func TestVirtual_DeactivateObject_FinishPartialDeactivation(t *testing.T) {
	insrail.LogCase(t, "C5508")
	const origMem = "object orig mem"

	defer commonTestUtils.LeakTester(t)

	cases := []struct {
		name      string
		isolation contract.MethodIsolation
	}{
		{
			name:      "Tolerable-Dirty",
			isolation: tolerableFlags(),
		},
		{
			name:      "Intolerable-Validated",
			isolation: intolerableFlags(),
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)

			runnerMock := logicless.NewServiceMock(ctx, mc, nil)
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				class               = server.RandomGlobalWithPulse()
				objectRef           = server.RandomGlobalWithPulse()
				p1                  = server.GetPulse().PulseNumber
				deactivateIsolation = contract.MethodIsolation{
					Interference: isolation.CallTolerable,
					State:        isolation.CallDirty,
				}
				stateRef = server.RandomRecordOf(objectRef)
				outgoing = server.BuildRandomOutgoingWithPulse()
				incoming = reference.NewRecordOf(objectRef, outgoing.GetLocal())

				pl *rms.VCallRequest
			)

			server.IncrementPulseAndWaitIdle(ctx)
			checkOutgoing := server.BuildRandomOutgoingWithPulse()

			{
				runnerMock.AddExecutionClassify(checkOutgoing, testCase.isolation, nil)
				if testCase.isolation.State == isolation.CallValidated {
					runnerMock.AddExecutionMock(checkOutgoing).AddStart(
						func(ctx execution.Context) {
							assert.Equal(t, objectRef, ctx.Request.Callee.GetValue())
							assert.Equal(t, []byte(origMem), ctx.ObjectDescriptor.Memory())
						},
						&execution.Update{
							Type:   execution.Done,
							Result: requestresult.New([]byte("check done"), objectRef),
						},
					)
				}
			}

			typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
			typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
				assert.Equal(t, objectRef, res.Callee.GetValue())
				assert.Equal(t, checkOutgoing, res.CallOutgoing.GetValue())
				if testCase.isolation.State == isolation.CallValidated {
					assert.Equal(t, []byte("check done"), res.ReturnArguments.GetBytes())
				} else {
					contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments.GetBytes())
					require.NoError(t, sysErr)
					assert.Contains(t, contractErr.Error(), "try to call method on deactivated object")
				}
				return false
			})
			typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
				assert.Equal(t, objectRef, report.Object.GetValue())
				assert.Equal(t, rms.StateStatusInactive, report.Status)
				assert.True(t, report.DelegationSpec.IsZero())
				assert.Equal(t, int32(0), report.UnorderedPendingCount)
				assert.Equal(t, int32(0), report.OrderedPendingCount)
				assert.Nil(t, report.ProvidedContent)
				return false
			})
			typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
				assert.Equal(t, objectRef, report.Object.GetValue())
				assert.Equal(t, checkOutgoing.GetLocal().Pulse(), report.AsOf)
				if testCase.isolation.Interference == isolation.CallTolerable {
					// VCallResult error: "try to call method on deactivated object"
					// todo: we should write VCallResult with 4-type error in transcript ?
					assert.Empty(t, report.ObjectTranscript.Entries)
				} else {
					assert.NotEmpty(t, report.ObjectTranscript.Entries)

					transcript := report.ObjectTranscript
					assert.Equal(t, 2, len(transcript.Entries))

					request1, ok := transcript.Entries[0].Get().(*rms.Transcript_TranscriptEntryIncomingRequest)
					require.True(t, ok)
					require.Equal(t, checkOutgoing, request1.Request.CallOutgoing.GetValue())
					utils.AssertVCallRequestEqual(t, pl, &request1.Request)

					result1, ok := transcript.Entries[1].Get().(*rms.Transcript_TranscriptEntryIncomingResult)
					require.True(t, ok)
					require.Equal(t, p1, result1.ObjectState.GetValue().GetLocal().Pulse())
					require.Equal(t, checkOutgoing, result1.Reason.GetValue())
				}
				return false
			})
			typedChecker.VDelegatedCallResponse.Set(func(response *rms.VDelegatedCallResponse) bool {
				assert.Equal(t, objectRef, response.Callee.GetValue())
				require.NotEmpty(t, response.ResponseDelegationSpec)
				assert.Equal(t, objectRef, response.ResponseDelegationSpec.Callee.GetValue())
				assert.Equal(t, server.GlobalCaller(), response.ResponseDelegationSpec.DelegateTo.GetValue())
				return false
			})

			{ // create object with pending deactivation
				report := rms.VStateReport{
					Status:                      rms.StateStatusReady,
					AsOf:                        p1,
					Object:                      rms.NewReference(objectRef),
					OrderedPendingCount:         1,
					OrderedPendingEarliestPulse: p1,
					LatestDirtyState:            rms.NewReference(stateRef),
					ProvidedContent: &rms.VStateReport_ProvidedContentBody{
						LatestDirtyState: &rms.ObjectState{
							Reference: rms.NewReference(stateRef),
							Class:     rms.NewReference(class),
							Memory:    rms.NewBytes([]byte(origMem)),
						},
						LatestValidatedState: &rms.ObjectState{
							Reference: rms.NewReference(stateRef),
							Class:     rms.NewReference(class),
							Memory:    rms.NewBytes([]byte(origMem)),
						},
					},
				}
				server.SendPayload(ctx, &report)
				server.WaitActiveThenIdleConveyor()
			}

			{ // fill object pending table
				dcrAwait := server.Journal.WaitStopOf(&handlers.SMVDelegatedCallRequest{}, 1)
				dcr := rms.VDelegatedCallRequest{
					Callee:       rms.NewReference(objectRef),
					CallFlags:    rms.BuildCallFlags(deactivateIsolation.Interference, deactivateIsolation.State),
					CallOutgoing: rms.NewReference(outgoing),
					CallIncoming: rms.NewReference(incoming),
				}
				server.SendPayload(ctx, &dcr)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, dcrAwait)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
			}

			{ // send delegation request finished with deactivate flag
				deactivateRequest := utils.GenerateVCallRequestMethod(server)
				deactivateRequest.Callee.Set(objectRef)
				deactivateRequest.CallSiteMethod = "Destroy"
				deactivateRequest.CallOutgoing.Set(outgoing)

				prevStateRef := server.RandomRecordOf(objectRef)
				pendingTranscript := utils.BuildIncomingTranscript(*deactivateRequest, prevStateRef, stateRef)

				pl := rms.VDelegatedRequestFinished{
					CallType:          rms.CallTypeMethod,
					Callee:            rms.NewReference(objectRef),
					CallOutgoing:      rms.NewReference(outgoing),
					CallIncoming:      rms.NewReference(incoming),
					CallFlags:         rms.BuildCallFlags(deactivateIsolation.Interference, deactivateIsolation.State),
					PendingTranscript: pendingTranscript,
					LatestState: &rms.ObjectState{
						Reference:   rms.NewReference(stateRef),
						Memory:      rms.NewBytes(nil),
						Deactivated: true,
					},
				}
				await := server.Journal.WaitStopOf(&handlers.SMVDelegatedRequestFinished{}, 1)
				server.SendPayload(ctx, &pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
			}

			{
				pl = utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = rms.BuildCallFlags(testCase.isolation.Interference, testCase.isolation.State)
				pl.Callee.Set(objectRef)
				pl.CallSiteMethod = "Check"
				pl.CallOutgoing.Set(checkOutgoing)

				await := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
			}

			server.IncrementPulse(ctx)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VObjectTranscriptReport.Wait(ctx, 1))

			assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())
			assert.Equal(t, 1, typedChecker.VStateReport.Count())
			assert.Equal(t, 1, typedChecker.VDelegatedCallResponse.Count())
			assert.Equal(t, 1, typedChecker.VCallResult.Count())

			server.Stop()
			mc.Finish()
		})
	}
}
