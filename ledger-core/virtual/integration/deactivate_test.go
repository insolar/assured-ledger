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
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_DeactivateObject(t *testing.T) {
	t.Log("C5134")

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
				pulseNumber       = server.GetPulse().PulseNumber

				outgoingDestroy  = server.BuildRandomOutgoingWithPulse()
				waitVStateReport = make(chan struct{})
			)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			// Add VStateReport check
			{
				typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
					assert.Equal(t, objectGlobal, report.Object)
					assert.Equal(t, pulseNumber, report.ProvidedContent.LatestValidatedState.Reference.Pulse())
					assert.Equal(t, pulseNumber, report.ProvidedContent.LatestDirtyState.Reference.Pulse())

					assert.Equal(t, payload.Ready, report.Status)
					// TODO: must be inactive and without content
					// TODO: remove error filter in server
					// assert.Equal(t, payload.Inactive, report.Status)
					// assert.True(t, report.ProvidedContent.LatestValidatedState.Deactivated)
					// assert.True(t, report.ProvidedContent.LatestDirtyState.Deactivated)

					waitVStateReport <- struct{}{}
					return false
				})
				typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
					assert.Equal(t, outgoingDestroy, result.CallOutgoing)
					return false
				})
			}

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
					ProvidedContent: content,
				}
				server.SendPayload(ctx, pl)
				testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1))
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
	t.Log("C4975")
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
				{
					name:     "call constructor",
					callType: payload.CTConstructor,
					errorMsg: "attempt to construct object that was completely deactivated",
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
						object = reference.NewSelf(server.RandomLocalWithPulse())
					)

					Method_PrepareObject(ctx, server, payload.Inactive, object)

					gotResult := make(chan struct{})

					typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
					typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {

						assert.Equal(t, res.Callee, object)
						contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
						require.Equal(t, &foundation.Error{callTypeTest.errorMsg}, contractErr)
						require.NoError(t, sysErr)

						gotResult <- struct{}{}

						return false // no resend msg
					})

					pl := payload.VCallRequest{
						CallType:            callTypeTest.callType,
						CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, stateTest.objectState),
						Caller:              server.GlobalCaller(),
						Callee:              object,
						CallSiteDeclaration: testwallet.GetClass(),
						CallSiteMethod:      "MyFavorMethod",
						CallOutgoing:        reference.NewSelf(object.GetLocal()),
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

func TestVirtual_CallMethod_On_DeactivatedDirtyState(t *testing.T) {
	t.Skip("https://insolar.atlassian.net/browse/PLAT-416")
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
	)

	{
		// Create object
		Method_PrepareObject(ctx, server, payload.Ready, object)
	}

	isolation := contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
	{
		// execution mock for deactivation
		descr := descriptor.NewObject(object, server.RandomLocalWithPulse(), server.RandomGlobalWithPulse(), makeRawWalletState(initialBalance))
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
			// TODO: check result !!!!!!!!!
			require.Equal(t, res.Callee, object)
			gotResult <- struct{}{}
			return false // no resend msg
		})
		pl := payload.VCallRequest{
			CallType:            payload.CTMethod,
			CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
			Caller:              server.GlobalCaller(),
			Callee:              object,
			CallSiteDeclaration: testwallet.GetClass(),
			CallSiteMethod:      deactivateMethod,
			CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		server.SendPayload(ctx, &pl)

		commontestutils.WaitSignalsTimed(t, 10*time.Second, gotResult)
	}

	// Here object dirty state is deactivated

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
				gotResult := make(chan struct{})
				typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					if !test.shouldExecute {
						require.Equal(t, res.Callee, object)
						contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
						require.Equal(t, &foundation.Error{"attempt to call method on object state that is deactivated"}, contractErr)
						require.NoError(t, sysErr)
					} else {
						// TODO: check result !!!!!!!!!
					}

					gotResult <- struct{}{}

					return false // no resend msg
				})

				callMethod := "SomeCallMethod"
				isolation = contract.MethodIsolation{Interference: contract.CallTolerable, State: test.objectState}
				if test.shouldExecute {
					objectExecutionMock := runnerMock.AddExecutionMock(callMethod)
					objectExecutionMock.AddStart(nil, &execution.Update{
						Type:   execution.Done,
						Result: requestresult.New([]byte("123"), gen.UniqueGlobalRef()),
					},
					)
					runnerMock.AddExecutionClassify(callMethod, isolation, nil)
				}

				pl := payload.VCallRequest{
					CallType:            payload.CTMethod,
					CallFlags:           payload.BuildCallFlags(isolation.Interference, isolation.State),
					Caller:              server.GlobalCaller(),
					Callee:              object,
					CallSiteDeclaration: testwallet.GetClass(),
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
