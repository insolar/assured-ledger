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
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
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
				class            = server.RandomGlobalWithPulse()
				objectGlobal     = server.RandomGlobalWithPulse()
				pulseNumberFirst = server.GetPulse().PulseNumber

				validatedStateRef = server.RandomLocalWithPulse()
				dirtyStateRef     = validatedStateRef

				validatedState = []byte("initial state")
				dirtyState     = validatedState
			)

			server.IncrementPulseAndWaitIdle(ctx)

			// Send VStateReport
			{
				if !test.stateIsEqual {
					dirtyState = []byte("dirty state")
					dirtyStateRef = server.RandomLocalWithPulse()
				}

				content := &payload.VStateReport_ProvidedContentBody{
					LatestDirtyState: &payload.ObjectState{
						Reference:   dirtyStateRef,
						Class:       class,
						State:       dirtyState,
						Deactivated: test.dirtyIsDeactivated,
					},
					LatestValidatedState: &payload.ObjectState{
						Reference: validatedStateRef,
						Class:     class,
						State:     validatedState,
					},
				}

				pl := &payload.VStateReport{
					Status:          payload.Ready,
					Object:          objectGlobal,
					AsOf:            pulseNumberFirst,
					ProvidedContent: content,
				}

				if test.entirelyDeactivated {
					pl.ProvidedContent = nil
					pl.Status = payload.Inactive
				}

				wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, pl)
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
				objectAExecutionMock := runnerMock.AddExecutionMock(outgoingDestroy.String())
				objectAExecutionMock.AddStart(nil, &execution.Update{
					Type:     execution.OutgoingCall,
					Outgoing: execution.NewRPCBuilder(outgoingDestroy, objectGlobal).Deactivate(),
				},
				).AddContinue(nil, &execution.Update{
					Type:   execution.Done,
					Result: requestresult.New([]byte("deactivate result"), objectGlobal),
				},
				)
				runnerMock.AddExecutionClassify(outgoingDestroy.String(), tolerableFlags(), nil)

				// get methods
				runnerMock.AddExecutionClassify(outgoingGetValidated.String(), intolerableFlags(), nil)
				runnerMock.AddExecutionClassify(outgoingGetDirty.String(), contract.MethodIsolation{Interference: contract.CallIntolerable, State: contract.CallDirty}, nil)

				runnerMock.AddExecutionMock(outgoingGetValidated.String()).AddStart(
					func(ctx execution.Context) {
						require.Equal(t, objectGlobal, ctx.Request.Callee)
						require.Equal(t, validatedState, ctx.ObjectDescriptor.Memory())
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("validated result"), gen.UniqueGlobalRef()),
					},
				)
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			// typedChecker mock
			{
				typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
					assert.Equal(t, objectGlobal, report.Object)
					assert.Equal(t, payload.Inactive, report.Status)
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
				typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
					assert.Equal(t, objectGlobal, result.Callee)

					switch result.CallOutgoing {
					case outgoingDestroy:
						assert.Equal(t, []byte("deactivate result"), result.ReturnArguments)
					case outgoingGetValidated:
						if test.entirelyDeactivated {
							contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments)
							require.NoError(t, sysErr)
							assert.Contains(t, contractErr.Error(), "try to call method on deactivated object")
						} else {
							assert.Equal(t, []byte("validated result"), result.ReturnArguments)
						}
					case outgoingGetDirty:
						contractErr, sysErr := foundation.UnmarshalMethodResult(result.ReturnArguments)
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
					pl.Callee = objectGlobal
					pl.CallOutgoing = outgoingDestroy

					executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
					server.SendPayload(ctx, pl)
					commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
				}
			}

			// VCallRequest get state dirty/validated
			{
				// get validated state
				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated)
				pl.Callee = objectGlobal
				pl.CallSiteMethod = "GetValidated"
				pl.CallOutgoing = outgoingGetValidated

				executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)

				// get dirty state
				pl = utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
				pl.Callee = objectGlobal
				pl.CallSiteMethod = "GetDirty"
				pl.CallOutgoing = outgoingGetDirty

				executeDone = server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
			}

			// Check VStateReport after pulse change
			{
				server.IncrementPulse(ctx)

				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
				assert.Equal(t, 1, typedChecker.VStateReport.Count())
			}

			mc.Finish()
		})
	}
}

func TestVirtual_CallMethod_On_CompletelyDeactivatedObject(t *testing.T) {
	insrail.LogCase(t, "C4975")
	stateTestCases := []struct {
		name        string
		objectState contract.StateFlag
	}{
		{
			name:        "call on validated state",
			objectState: contract.CallValidated,
		},
		{
			name:        "call on dirty state",
			objectState: contract.CallDirty,
		},
	}

	for _, stateTest := range stateTestCases {
		t.Run(stateTest.name, func(t *testing.T) {

			callTypeTestCases := []struct {
				name     string
				callType payload.CallType
				errorMsg string
			}{
				{
					name:     "call method",
					callType: payload.CTMethod,
					errorMsg: "try to call method on deactivated object",
				},
			}

			for _, callTypeTest := range callTypeTestCases {
				t.Run(callTypeTest.name, func(t *testing.T) {
					defer commonTestUtils.LeakTester(t)

					mc := minimock.NewController(t)

					server, ctx := utils.NewUninitializedServer(nil, t)
					defer server.Stop()

					runnerMock := logicless.NewServiceMock(ctx, t, func(execution execution.Context) string {
						return execution.Request.CallSiteMethod
					})

					isolation := contract.MethodIsolation{Interference: contract.CallIntolerable, State: stateTest.objectState}
					methodName := "MyFavorMethod" + callTypeTest.name
					runnerMock.AddExecutionClassify(methodName, isolation, nil)
					server.ReplaceRunner(runnerMock)

					server.Init(ctx)
					server.IncrementPulseAndWaitIdle(ctx)

					var (
						object    = server.RandomGlobalWithPulse()
						prevPulse = server.GetPulse().PulseNumber
					)

					server.IncrementPulseAndWaitIdle(ctx)
					Method_PrepareObject(ctx, server, payload.Inactive, object, prevPulse)

					gotResult := make(chan struct{})

					typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
					typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {

						assert.Equal(t, object, res.Callee)
						contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
						require.NoError(t, sysErr)
						require.Contains(t, contractErr.Error(), callTypeTest.errorMsg)

						gotResult <- struct{}{}

						return false // no resend msg
					})

					pl := payload.VCallRequest{
						CallType:       callTypeTest.callType,
						CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
						Caller:         server.GlobalCaller(),
						Callee:         object,
						CallSiteMethod: methodName,
						CallOutgoing:   server.BuildRandomOutgoingWithPulse(),
						Arguments:      insolar.MustSerialize([]interface{}{}),
					}
					server.SendPayload(ctx, &pl)

					commonTestUtils.WaitSignalsTimed(t, 10*time.Second, gotResult)

					mc.Finish()
				})
			}
		})
	}
}

func TestVirtual_CallDeactivate_Intolerable(t *testing.T) {
	insrail.LogCase(t, "C5469")

	table := []struct {
		name  string
		state contract.StateFlag
	}{
		{name: "ValidatedState", state: contract.CallValidated},
		{name: "DirtyState", state: contract.CallDirty},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			defer commonTestUtils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				class        = server.RandomGlobalWithPulse()
				objectGlobal = server.RandomGlobalWithPulse()
				prevPulse    = server.GetPulse().PulseNumber
			)

			// Create object
			{
				server.IncrementPulse(ctx)

				report := &payload.VStateReport{
					Status: payload.Ready,
					Object: objectGlobal,
					AsOf:   prevPulse,
					ProvidedContent: &payload.VStateReport_ProvidedContentBody{
						LatestDirtyState: &payload.ObjectState{
							Reference: reference.Local{},
							Class:     class,
							State:     []byte("initial state"),
						},
						LatestValidatedState: &payload.ObjectState{
							Reference: reference.Local{},
							Class:     class,
							State:     []byte("initial state"),
						},
					},
				}
				wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, report)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			// Add VCallResult check
			{
				typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
					require.Equal(t, []byte("finish Deactivate"), result.ReturnArguments)

					return false
				})
			}

			// Add executor mock for method `Destroy`
			{
				runnerMock.AddExecutionClassify("Destroy", contract.MethodIsolation{Interference: contract.CallIntolerable, State: testCase.state}, nil)

				requestResult := requestresult.New([]byte("outgoing call"), objectGlobal)
				requestResult.SetDeactivate(descriptor.NewObject(objectGlobal, server.RandomLocalWithPulse(), class, []byte("initial state"), false))
				runnerMock.AddExecutionMock("Destroy").AddStart(
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
						require.Equal(t, "interference violation: deactivate call from intolerable call", contractErr.Error())
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
				pl.CallFlags = payload.BuildCallFlags(contract.CallIntolerable, testCase.state)
				pl.Callee = objectGlobal
				pl.CallSiteMethod = "Destroy"

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

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		class               = server.RandomGlobalWithPulse()
		objectRef           = server.RandomGlobalWithPulse()
		p1                  = server.GetPulse().PulseNumber
		deactivateIsolation = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}
	)

	server.IncrementPulseAndWaitIdle(ctx)
	outgoing := server.BuildRandomOutgoingWithPulse()

	synchronizeExecution := synchronization.NewPoint(1)
	{ // setup runner mock for deactivation call
		runnerMock.AddExecutionClassify(deactivateMethodName, deactivateIsolation, nil)
		requestResult := requestresult.New([]byte(deactivateMethodName+" result"), objectRef)
		requestResult.SetDeactivate(descriptor.NewObject(objectRef, server.RandomLocalWithPulse(), class, insolar.MustSerialize(initialBalance), false))

		runnerMock.AddExecutionMock(deactivateMethodName).AddStart(
			func(ctx execution.Context) {
				require.Equal(t, objectRef, ctx.Request.Callee)
				require.Equal(t, []byte(origDirtyMem), ctx.ObjectDescriptor.Memory())
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

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, objectRef, res.Callee)
			require.Equal(t, outgoing, res.CallOutgoing)
			require.Equal(t, []byte("finish Deactivate"), res.ReturnArguments)
			return false
		})
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			require.Equal(t, objectRef, report.Object)
			require.Equal(t, payload.Ready, report.Status)
			require.True(t, report.DelegationSpec.IsZero())
			require.Equal(t, int32(0), report.UnorderedPendingCount)
			require.Equal(t, int32(1), report.OrderedPendingCount)
			require.NotNil(t, report.ProvidedContent)
			require.NotNil(t, report.ProvidedContent.LatestDirtyState)
			require.False(t, report.ProvidedContent.LatestDirtyState.Deactivated)
			require.Equal(t, []byte(origDirtyMem), report.ProvidedContent.LatestDirtyState.State)
			require.NotNil(t, report.ProvidedContent.LatestValidatedState)
			require.False(t, report.ProvidedContent.LatestValidatedState.Deactivated)
			require.Equal(t, []byte(origDirtyMem), report.ProvidedContent.LatestValidatedState.State)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
			require.Equal(t, objectRef, request.Callee)
			require.Equal(t, outgoing, request.CallOutgoing)
			token := server.DelegationToken(request.CallOutgoing, server.GlobalCaller(), request.Callee)

			response := payload.VDelegatedCallResponse{
				Callee:                 request.Callee,
				CallIncoming:           request.CallIncoming,
				ResponseDelegationSpec: token,
			}
			server.SendPayload(ctx, &response)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			require.NotNil(t, finished.LatestState)
			require.True(t, finished.LatestState.Deactivated)
			require.Nil(t, finished.LatestState.State)
			return false
		})
	}
	{
		report := payload.VStateReport{
			Status: payload.Ready,
			AsOf:   p1,
			Object: objectRef,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: reference.Local{},
					Class:     class,
					State:     []byte(origDirtyMem),
				},
				LatestValidatedState: &payload.ObjectState{
					Reference: reference.Local{},
					Class:     class,
					State:     []byte(origValidatedMem),
				},
			},
		}
		wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, &report)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)
	}

	{
		pl := utils.GenerateVCallRequestMethod(server)
		pl.Callee = objectRef
		pl.CallSiteMethod = "Deactivate"
		pl.CallOutgoing = outgoing

		server.SendPayload(ctx, pl)
	}

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
	server.IncrementPulseAndWaitIdle(ctx)
	synchronizeExecution.WakeUp()
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, oneExecutionEnded)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VStateReport.Count())
	require.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())
	require.Equal(t, 1, typedChecker.VCallResult.Count())

	server.Stop()
	mc.Finish()
}

func TestVirtual_CallMethod_After_Deactivation(t *testing.T) {
	insrail.LogCase(t, "C5509")
	defer commonTestUtils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})

	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		class               = server.RandomGlobalWithPulse()
		objectRef           = server.RandomGlobalWithPulse()
		p1                  = server.GetPulse().PulseNumber
		deactivateIsolation = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}
	)

	// Create object
	{
		server.IncrementPulseAndWaitIdle(ctx)
		Method_PrepareObject(ctx, server, payload.Ready, objectRef, p1)
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

		objectExecutionMock := runnerMock.AddExecutionMock("Destroy")
		objectExecutionMock.AddStart(nil, &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: execution.NewRPCBuilder(server.RandomGlobalWithPulse(), objectRef).Deactivate(),
		}).AddContinue(func(result []byte) {
			synchronizeExecution.Synchronize()
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
		runnerMock.AddExecutionClassify("Destroy", deactivateIsolation, nil)
		runnerMock.AddExecutionClassify("SomeMethod", deactivateIsolation, nil)
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	// Add check
	{
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, objectRef, res.Callee)
			if res.CallOutgoing == outgoingDeactivate {
				require.Equal(t, []byte("done"), res.ReturnArguments)
			} else {
				contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
				require.NoError(t, sysErr)
				assert.Contains(t, contractErr.Error(), "try to call method on deactivated object")
				require.Equal(t, outgoingSomeMethod, res.CallOutgoing)
			}
			return false
		})
	}

	// Deactivate
	{
		pl := utils.GenerateVCallRequestMethod(server)
		pl.Callee = objectRef
		pl.CallSiteMethod = "Destroy"
		pl.CallOutgoing = outgoingDeactivate

		server.SendPayload(ctx, pl)
	}
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())

	// vCallRequest
	{
		pl := utils.GenerateVCallRequestMethod(server)
		pl.Callee = objectRef
		pl.CallSiteMethod = "SomeMethod"
		pl.CallOutgoing = outgoingSomeMethod

		server.SendPayload(ctx, pl)
	}

	time.Sleep(100 * time.Millisecond)
	synchronizeExecution.WakeUp()

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, twoExecutionEnded)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 2, typedChecker.VCallResult.Count())

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
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class              = server.RandomGlobalWithPulse()
		outgoing           = server.BuildRandomOutgoingWithPulse()
		objectRef          = reference.NewSelf(outgoing.GetLocal())
		outgoingDeactivate = server.BuildRandomOutgoingWithPulse()
		isolation          = contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}
	)

	// mock
	{
		// Deactivate mock
		descr := descriptor.NewObject(objectRef, server.RandomLocalWithPulse(), class, []byte("deactivate state"), false)
		requestResult := requestresult.New([]byte("deactivated"), objectRef)
		requestResult.SetDeactivate(descr)

		deactivationMock := runnerMock.AddExecutionMock(outgoingDeactivate.String())
		deactivationMock.AddStart(nil, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
		runnerMock.AddExecutionClassify(outgoingDeactivate.String(), isolation, nil)

		// Constructor mock
		result := requestresult.New([]byte("new"), outgoing)
		result.SetActivate(reference.Global{}, class, []byte("state"))

		constructorMock := runnerMock.AddExecutionMock(outgoing.String())
		constructorMock.AddStart(nil, &execution.Update{
			Type:   execution.Done,
			Result: result,
		})
		runnerMock.AddExecutionClassify(outgoing.String(), isolation, nil)
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	// Add check
	{
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			switch res.CallOutgoing {
			case outgoing:
				require.Equal(t, []byte("new"), res.ReturnArguments)
				require.Equal(t, objectRef, res.Callee)
			case outgoingDeactivate:
				require.Equal(t, []byte("deactivated"), res.ReturnArguments)
				require.Equal(t, objectRef, res.Callee)
			}
			return false
		})
	}

	// Constructor
	constructRequest := utils.GenerateVCallRequestConstructor(server)
	constructRequest.Callee = objectRef
	constructRequest.CallSiteMethod = "Destroy"
	constructRequest.CallOutgoing = outgoing

	// Deactivation
	deactivateRequest := utils.GenerateVCallRequestMethod(server)
	deactivateRequest.Callee = objectRef
	deactivateRequest.CallSiteMethod = "Destroy"
	deactivateRequest.CallOutgoing = outgoingDeactivate

	requests := []*payload.VCallRequest{constructRequest, deactivateRequest, constructRequest, deactivateRequest}
	for _, r := range requests {
		await := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
		server.SendPayload(ctx, r)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
	}

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	require.Equal(t, 4, typedChecker.VCallResult.Count())

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
				server.IncrementPulse(ctx)

				report := &payload.VStateReport{
					Status:          payload.Inactive,
					Object:          objectGlobal,
					AsOf:            prevPulse,
					ProvidedContent: nil,
				}
				wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, report)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, wait)
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			// Add checker mock
			{
				typedChecker.VFindCallRequest.Set(func(request *payload.VFindCallRequest) bool {
					assert.Equal(t, objectGlobal, request.Callee)
					assert.Equal(t, outgoingPrevPulse, request.Outgoing)
					assert.Equal(t, prevPulse, request.LookAt)

					pl := &payload.VFindCallResponse{
						LookedAt: request.LookAt,
						Callee:   request.Callee,
						Outgoing: request.Outgoing,
						Status:   payload.FoundCall,
						CallResult: &payload.VCallResult{
							CallType:        payload.CTMethod,
							CallFlags:       payload.BuildCallFlags(testCase.flags.Interference, testCase.flags.State),
							Callee:          request.Callee,
							Caller:          server.GlobalCaller(),
							CallOutgoing:    request.Outgoing,
							CallIncoming:    server.RandomGlobalWithPulse(),
							ReturnArguments: []byte("result from past"),
						},
					}
					server.SendPayload(ctx, pl)
					return false
				})
				typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
					assert.Equal(t, objectGlobal, result.Callee)
					assert.Equal(t, outgoingPrevPulse, result.CallOutgoing)
					assert.Equal(t, []byte("result from past"), result.ReturnArguments)
					return false
				})
			}

			// Add execution mock classify
			runnerMock.AddExecutionClassify(outgoingPrevPulse.String(), testCase.flags, nil)

			// VCallRequest from previous pulse
			{
				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = payload.BuildCallFlags(testCase.flags.Interference, testCase.flags.State)
				pl.Callee = objectGlobal
				pl.CallSiteMethod = "Foo"
				pl.CallOutgoing = outgoingPrevPulse

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
					Interference: contract.CallTolerable,
					State:        contract.CallDirty,
				}
				stateRef = server.RandomGlobalWithPulse()
				outgoing = server.BuildRandomOutgoingWithPulse()
				incoming = reference.NewRecordOf(objectRef, outgoing.GetLocal())
			)

			server.IncrementPulseAndWaitIdle(ctx)
			checkOutgoing := server.BuildRandomOutgoingWithPulse()

			{
				runnerMock.AddExecutionClassify(checkOutgoing.String(), testCase.isolation, nil)
				if testCase.isolation.State == contract.CallValidated {
					runnerMock.AddExecutionMock(checkOutgoing.String()).AddStart(
						func(ctx execution.Context) {
							require.Equal(t, objectRef, ctx.Request.Callee)
							require.Equal(t, []byte(origMem), ctx.ObjectDescriptor.Memory())
						},
						&execution.Update{
							Type:   execution.Done,
							Result: requestresult.New([]byte("check done"), objectRef),
						},
					)
				}
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
				require.Equal(t, objectRef, res.Callee)
				require.Equal(t, checkOutgoing, res.CallOutgoing)
				if testCase.isolation.State == contract.CallValidated {
					require.Equal(t, []byte("check done"), res.ReturnArguments)
				} else {
					contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
					require.NoError(t, sysErr)
					require.Contains(t, contractErr.Error(), "try to call method on deactivated object")
				}
				return false
			})
			typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
				require.Equal(t, objectRef, report.Object)
				require.Equal(t, payload.Inactive, report.Status)
				require.True(t, report.DelegationSpec.IsZero())
				require.Equal(t, int32(0), report.UnorderedPendingCount)
				require.Equal(t, int32(0), report.OrderedPendingCount)
				require.Nil(t, report.ProvidedContent)
				return false
			})
			typedChecker.VDelegatedCallResponse.Set(func(response *payload.VDelegatedCallResponse) bool {
				require.Equal(t, objectRef, response.Callee)
				require.NotEmpty(t, response.ResponseDelegationSpec)
				require.Equal(t, objectRef, response.ResponseDelegationSpec.Callee)
				require.Equal(t, server.GlobalCaller(), response.ResponseDelegationSpec.DelegateTo)
				return false
			})

			{ // create object with pending deactivation
				report := payload.VStateReport{
					Status:                      payload.Ready,
					AsOf:                        p1,
					Object:                      objectRef,
					OrderedPendingCount:         1,
					OrderedPendingEarliestPulse: p1,
					LatestDirtyState:            stateRef,
					ProvidedContent: &payload.VStateReport_ProvidedContentBody{
						LatestDirtyState: &payload.ObjectState{
							Reference: stateRef.GetLocal(),
							Class:     class,
							State:     []byte(origMem),
						},
						LatestValidatedState: &payload.ObjectState{
							Reference: stateRef.GetLocal(),
							Class:     class,
							State:     []byte(origMem),
						},
					},
				}
				server.SendPayload(ctx, &report)
				server.WaitActiveThenIdleConveyor()
			}

			{ // fill object pending table
				dcrAwait := server.Journal.WaitStopOf(&handlers.SMVDelegatedCallRequest{}, 1)
				dcr := payload.VDelegatedCallRequest{
					Callee:       objectRef,
					CallFlags:    payload.BuildCallFlags(deactivateIsolation.Interference, deactivateIsolation.State),
					CallOutgoing: outgoing,
					CallIncoming: incoming,
				}
				server.SendPayload(ctx, &dcr)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, dcrAwait)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
			}

			{ // send delegation request finished with deactivate flag
				pl := payload.VDelegatedRequestFinished{
					CallType:     payload.CTMethod,
					Callee:       objectRef,
					CallOutgoing: outgoing,
					CallIncoming: incoming,
					CallFlags:    payload.BuildCallFlags(deactivateIsolation.Interference, deactivateIsolation.State),
					LatestState: &payload.ObjectState{
						State:       nil,
						Deactivated: true,
					},
				}
				await := server.Journal.WaitStopOf(&handlers.SMVDelegatedRequestFinished{}, 1)
				server.SendPayload(ctx, &pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
			}

			{
				pl := utils.GenerateVCallRequestMethod(server)
				pl.CallFlags = payload.BuildCallFlags(testCase.isolation.Interference, testCase.isolation.State)
				pl.Callee = objectRef
				pl.CallSiteMethod = "Check"
				pl.CallOutgoing = checkOutgoing

				await := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, await)
			}

			server.IncrementPulse(ctx)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VStateReport.Wait(ctx, 1))
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VStateReport.Count())
			require.Equal(t, 1, typedChecker.VDelegatedCallResponse.Count())
			require.Equal(t, 1, typedChecker.VCallResult.Count())

			server.Stop()
			mc.Finish()
		})
	}
}
