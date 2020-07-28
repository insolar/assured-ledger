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

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
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
