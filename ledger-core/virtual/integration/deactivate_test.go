// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"strings"
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
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"

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

			server, ctx := utils.NewServerWithErrorFilter(nil, t, func(s string) bool {
				return !strings.Contains(s, "(*SMExecute).stepSaveNewObject")
			})
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
			pulseNumberSecond := server.GetPulse().PulseNumber

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
					assert.Equal(t, pulseNumberSecond, report.ProvidedContent.LatestValidatedState.Reference.Pulse())
					assert.Equal(t, pulseNumberSecond, report.ProvidedContent.LatestDirtyState.Reference.Pulse())

					assert.Equal(t, payload.Ready, report.Status)
					assert.True(t, report.ProvidedContent.LatestValidatedState.Deactivated)
					assert.True(t, report.ProvidedContent.LatestDirtyState.Deactivated)

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
					errorMsg: "attempt to call method on object that is completely deactivated",
				},
			}

			for _, callTypeTest := range callTypeTestCases {
				t.Run(callTypeTest.name, func(t *testing.T) {
					defer commontestutils.LeakTester(t)

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
						require.Contains(t, contractErr.Error(), callTypeTest.errorMsg)
						require.NoError(t, sysErr)

						gotResult <- struct{}{}

						return false // no resend msg
					})

					pl := payload.VCallRequest{
						CallType:            callTypeTest.callType,
						CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, stateTest.objectState),
						Caller:              server.GlobalCaller(),
						Callee:              object,
						CallSiteDeclaration: gen.UniqueGlobalRef(),
						CallSiteMethod:      "MyFavorMethod",
						CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
						Arguments:           insolar.MustSerialize([]interface{}{}),
					}
					server.SendPayload(ctx, &pl)

					commontestutils.WaitSignalsTimed(t, 10*time.Second, gotResult)

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
func TestVirtual_CallMethod_On_DeactivatedDirtyState(t *testing.T) {
	t.Log("C5470")
	defer commontestutils.LeakTester(t)

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

		commontestutils.WaitSignalsTimed(t, 10*time.Second, gotResult)
	}

	////
	// Here object dirty state is deactivated
	////

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
						require.Contains(t, contractErr.Error(), "try to call method on deactivated object")
						require.NoError(t, sysErr)
					} else {
						require.Equal(t, requestResult.RawResult, res.ReturnArguments)
					}

					gotResult <- struct{}{}

					return false // no resend msg
				})

				callMethod := "SomeCallMethod" + test.name
				isolation = contract.MethodIsolation{Interference: contract.CallIntolerable, State: test.objectState}
				if test.shouldExecute {
					objectExecutionMock := runnerMock.AddExecutionMock(callMethod)
					objectExecutionMock.AddStart(nil, &execution.Update{
						Type:   execution.Done,
						Result: requestResult,
					},
					)
					runnerMock.AddExecutionClassify(callMethod, isolation, nil)
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
				commontestutils.WaitSignalsTimed(t, 10*time.Second, gotResult)
			})
		}
	}

	mc.Finish()
}
