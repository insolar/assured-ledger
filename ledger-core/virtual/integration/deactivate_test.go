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
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_DeactivateObject(t *testing.T) {
	defer testutils.LeakTester(t)
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

// 1. Create object
// 2. Deactivate object
// 3. Change Pulse
// 4. Send request on Dirty state - get error
// 5. Send request on Validated state - get error
func TestVirtual_CallMethod_On_DeactivatedDirtyState(t *testing.T) {
	defer testutils.LeakTester(t)
	insrail.LogCase(t, "C5506")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, t, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)

	server.Init(ctx)
	server.IncrementPulseAndWaitIdle(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

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
		descr := descriptor.NewObject(object, server.RandomLocalWithPulse(), server.RandomGlobalWithPulse(), []byte("some memory"), false)
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
		// send request for deactivate object
		gotResult := make(chan struct{})

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
			CallSiteDeclaration: testwallet.GetClass(),
			CallSiteMethod:      deactivateMethod,
			CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
			Arguments:           insolar.MustSerialize([]interface{}{}),
		}
		server.SendPayload(ctx, &pl)

		testutils.WaitSignalsTimed(t, 10*time.Second, gotResult)
	}

	reportDelivered := make(chan struct{})
	typedChecker.VStateReport.Set(func(r *payload.VStateReport) bool {
		server.SendPayload(ctx, r)
		reportDelivered <- struct{}{}
		return false
	})

	// after increment pulse DirtyState and ValidatedState should be Deactivated
	server.IncrementPulseAndWaitIdle(ctx)
	<-reportDelivered

	{
		// send call request on deactivated object
		testcase := []struct {
			name        string
			objectState contract.StateFlag
		}{
			{
				name:        "call on dirty state",
				objectState: contract.CallDirty,
			},
			{
				name:        "call on validated state",
				objectState: contract.CallValidated,
			},
		}

		expectedError := throw.E("try to call method on deactivated object", struct {
			ObjectReference string
		}{
			ObjectReference: object.String(),
		})

		for _, test := range testcase {
			t.Run(test.name, func(t *testing.T) {
				gotResult := make(chan struct{})
				typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					require.Equal(t, res.Callee, object)
					contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
					require.NotNil(t, contractErr)
					require.Equal(t, expectedError.Error(), contractErr.Error())
					require.NoError(t, sysErr)

					gotResult <- struct{}{}

					return false // no resend msg
				})

				callMethod := "SomeCallMethod" + test.name
				isolation = contract.MethodIsolation{Interference: contract.CallIntolerable, State: test.objectState}

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
				testutils.WaitSignalsTimed(t, 10*time.Second, gotResult)
			})
		}
	}

	mc.Finish()
}
