// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"

	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"

	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_DeactivateObject(t *testing.T) {
	insrail.LogCase(t, "C5134")

	table := []struct {
		name         string
		stateIsEqual bool
	}{
		{name: "ValidatedState==DirtyState", stateIsEqual: true},
		{name: "ValidatedState!=DirtyState", stateIsEqual: false},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer testutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewServer(nil, t)
			defer server.Stop()

			var (
				class        = testwallet.GetClass()
				objectGlobal = reference.NewSelf(server.RandomLocalWithPulse())

				dirtyStateRef     = server.RandomLocalWithPulse()
				validatedStateRef = server.RandomLocalWithPulse()
				pulseNumberFirst  = server.GetPulse().PulseNumber

				waitVStateReport = make(chan struct{})
			)

			server.IncrementPulseAndWaitIdle(ctx)

			// Send VStateReport with Dirty, Validated states
			{
				validatedState := makeRawWalletState(initialBalance)
				dirtyState := validatedState
				if !test.stateIsEqual {
					dirtyState = makeRawWalletState(initialBalance + 100)
				}

				content := &payload.VStateReport_ProvidedContentBody{
					LatestDirtyState: &payload.ObjectState{
						Reference: dirtyStateRef,
						Class:     class,
						State:     dirtyState,
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
				server.SendPayload(ctx, pl)
				testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1))
			}

			outgoingDestroy := server.BuildRandomOutgoingWithPulse()

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			// Add VStateReport check
			{
				typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
					assert.Equal(t, objectGlobal, report.Object)
					assert.Nil(t, report.ProvidedContent)
					assert.Equal(t, payload.Inactive, report.Status)

					waitVStateReport <- struct{}{}
					return false
				})
				typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
					assert.Equal(t, outgoingDestroy, result.CallOutgoing)
					return false
				})
			}

			// Deactivate object
			{
				pl := &payload.VCallRequest{
					CallType:            payload.CTMethod,
					CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
					Callee:              objectGlobal,
					CallSiteDeclaration: class,
					CallSiteMethod:      "Destroy",
					CallOutgoing:        outgoingDestroy,
					Arguments:           insolar.MustSerialize([]interface{}{}),
				}
				server.SendPayload(ctx, pl)
				testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&execute.SMExecute{}, 1))
			}
			server.IncrementPulse(ctx)

			testutils.WaitSignalsTimed(t, 10*time.Second, waitVStateReport)
			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VStateReport.Count())
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
				callType payload.CallTypeNew
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
						object    = reference.NewSelf(server.RandomLocalWithPulse())
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
						CallType:            callTypeTest.callType,
						CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
						Caller:              server.GlobalCaller(),
						Callee:              object,
						CallSiteDeclaration: gen.UniqueGlobalRef(),
						CallSiteMethod:      methodName,
						CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
						Arguments:           insolar.MustSerialize([]interface{}{}),
					}
					server.SendPayload(ctx, &pl)

					commonTestUtils.WaitSignalsTimed(t, 10*time.Second, gotResult)

					mc.Finish()
				})
			}
		})
	}
}

// 1. Create object
// 2. Deactivate object partially( only Dirty state )
// 3. Send request on Dirty state - get error
// 4. Send request on Validated state - get response
// TODO: Remove this test when https://insolar.atlassian.net/browse/PLAT-706 will be implemented
func TestVirtual_CallMethod_On_DeactivatedDirtyState(t *testing.T) {
	defer commonTestUtils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, t, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		object           = reference.NewSelf(server.RandomLocalWithPulse())
		runnerResult     = []byte("123")
		deactivateMethod = "deactivatingMethod"
		prevPulse        = server.GetPulse().PulseNumber
	)

	server.IncrementPulseAndWaitIdle(ctx)
	{
		// Create object
		Method_PrepareObject(ctx, server, payload.Ready, object, prevPulse)
	}

	isolation := contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
	{
		// execution mock for deactivation
		descr := descriptor.NewObject(object, server.RandomLocalWithPulse(), server.RandomGlobalWithPulse(), insolar.MustSerialize(initialBalance), false)
		requestResult := requestresult.New(runnerResult, object)
		requestResult.SetDeactivate(descr)

		objectAExecutionMock := runnerMock.AddExecutionMock(deactivateMethod)
		objectAExecutionMock.AddStart(nil, &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: execution.NewRPCBuilder(server.RandomGlobalWithPulse(), object).Deactivate(),
		},
		).AddContinue(nil, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		},
		)
		runnerMock.AddExecutionClassify(deactivateMethod, isolation, nil)
	}

	{
		// send request to deactivate object
		gotResult := make(chan struct{})

		typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			require.Equal(t, runnerResult, res.ReturnArguments)
			require.Equal(t, res.Callee, object)
			gotResult <- struct{}{}
			return false // no resend msg
		})
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
			Caller:              server.GlobalCaller(),
			Callee:              object,
			CallSiteDeclaration: gen.UniqueGlobalRef(),
			CallSiteMethod:      deactivateMethod,
			CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		server.SendPayload(ctx, &pl)

		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, gotResult)
	}

	// //
	// Here object dirty state is deactivated
	// //

	{
		// send call request on deactivated object
		testcase := []struct {
			name          string
			objectState   contract.StateFlag
			shouldExecute bool
		}{
			{
				name:          "call on dirty state",
				objectState:   contract.CallDirty,
				shouldExecute: false,
			},
			{
				name:          "call on validated state",
				objectState:   contract.CallValidated,
				shouldExecute: true,
			},
		}

		for _, test := range testcase {
			t.Run(test.name, func(t *testing.T) {
				requestResult := requestresult.New([]byte("838383"), gen.UniqueGlobalRef())
				gotResult := make(chan struct{})
				typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					if !test.shouldExecute {
						require.Equal(t, res.Callee, object)
						contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
						require.NoError(t, sysErr)
						require.Contains(t, contractErr.Error(), "try to call method on deactivated object")
					} else {
						require.Equal(t, requestResult.RawResult, res.ReturnArguments)
					}

					gotResult <- struct{}{}

					return false // no resend msg
				})

				callMethod := "SomeCallMethod" + test.name
				isolation = contract.MethodIsolation{Interference: contract.CallIntolerable, State: test.objectState}
				runnerMock.AddExecutionClassify(callMethod, isolation, nil)
				objectExecutionMock := runnerMock.AddExecutionMock(callMethod)
				if test.shouldExecute {
					objectExecutionMock.AddStart(nil, &execution.Update{
						Type:   execution.Done,
						Result: requestResult,
					},
					)
				} else {
					objectExecutionMock.AddStart(nil, &execution.Update{
						Type:  execution.Error,
						Error: errors.New("erroneous situation: this execution should not happen"),
					},
					)
				}

				pl := payload.VCallRequest{
					CallType:            payload.CTMethod,
					CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
					Caller:              server.GlobalCaller(),
					Callee:              object,
					CallSiteDeclaration: gen.UniqueGlobalRef(),
					CallSiteMethod:      callMethod,
					CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
					Arguments:           insolar.MustSerialize([]interface{}{}),
				}
				server.SendPayload(ctx, &pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, gotResult)
			})
		}
	}

	mc.Finish()
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
			defer testutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})
			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				class        = gen.UniqueGlobalRef()
				objectGlobal = reference.NewSelf(server.RandomLocalWithPulse())
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
				testutils.WaitSignalsTimed(t, 10*time.Second, wait)
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
				pl := &payload.VCallRequest{
					CallType:            payload.CTMethod,
					CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, testCase.state),
					Callee:              objectGlobal,
					CallSiteDeclaration: class,
					CallSiteMethod:      "Destroy",
					CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
					Arguments:           insolar.MustSerialize([]interface{}{}),
				}
				server.SendPayload(ctx, pl)
				testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&execute.SMExecute{}, 1))
			}

			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
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
		class               = gen.UniqueGlobalRef()
		objectRef           = reference.NewSelf(server.RandomLocalWithPulse())
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
	typedChecker.VStateRequest.Set(func(req *payload.VStateRequest) bool {
		require.Equal(t, p1, req.AsOf)
		require.Equal(t, objectRef, req.Object)

		flags := payload.RequestLatestDirtyState | payload.RequestLatestValidatedState |
			payload.RequestOrderedQueue | payload.RequestUnorderedQueue
		require.Equal(t, flags, req.RequestedContent)

		content := &payload.VStateReport_ProvidedContentBody{
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
		}

		report := payload.VStateReport{
			Status:          payload.Ready,
			AsOf:            req.AsOf,
			Object:          objectRef,
			ProvidedContent: content,
		}
		server.SendPayload(ctx, &report)
		return false
	})

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

	{
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(deactivateIsolation.Interference, deactivateIsolation.State),
			Caller:              server.GlobalCaller(),
			Callee:              objectRef,
			CallSiteDeclaration: class,
			CallSiteMethod:      "Deactivate",
			CallOutgoing:        outgoing,
		}
		server.SendPayload(ctx, &pl)
	}

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
	server.IncrementPulseAndWaitIdle(ctx)
	synchronizeExecution.WakeUp()
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, oneExecutionEnded)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 1, typedChecker.VStateReport.Count())
	require.Equal(t, 1, typedChecker.VStateRequest.Count())
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
		class               = gen.UniqueGlobalRef()
		objectRef           = reference.NewSelf(server.RandomLocalWithPulse())
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
	synchronizeExecution := synchronization.NewPoint(1)
	twoExecutionEnded := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	// mock
	{
		descr := descriptor.NewObject(objectRef, server.RandomLocalWithPulse(), class, []byte("deactivate state"), false)
		requestResult := requestresult.New([]byte("done"), objectRef)
		requestResult.SetDeactivate(descr)

		objectExecutionMock := runnerMock.AddExecutionMock("Destroy")
		objectExecutionMock.AddStart(nil, &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: execution.NewRPCBuilder(server.RandomGlobalWithPulse(), objectRef).Deactivate(),
		}).AddContinue(
			func(result []byte) {
				synchronizeExecution.Synchronize()
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			},
		)
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
		deactivateRequest := &payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(deactivateIsolation.Interference, deactivateIsolation.State),
			Callee:              objectRef,
			CallSiteDeclaration: class,
			CallSiteMethod:      "Destroy",
			CallOutgoing:        outgoingDeactivate,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		server.SendPayload(ctx, deactivateRequest)
	}

	// vCallRequest

	pl := &payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(deactivateIsolation.Interference, deactivateIsolation.State),
		Callee:              objectRef,
		CallSiteDeclaration: class,
		CallSiteMethod:      "SomeMethod",
		CallOutgoing:        outgoingSomeMethod,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
	server.SendPayload(ctx, pl)
	time.Sleep(100 * time.Millisecond)
	synchronizeExecution.WakeUp()

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, twoExecutionEnded)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	require.Equal(t, 2, typedChecker.VCallResult.Count())
	server.Stop()
	mc.Finish()
}
