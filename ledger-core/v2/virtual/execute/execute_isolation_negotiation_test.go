// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

func Test_Execute_stepIsolationNegotiation(t *testing.T) {

	for _, tc := range []struct {
		name string

		methodIsolation contract.MethodIsolation
		callIsolation   contract.MethodIsolation

		expectedIsolation contract.MethodIsolation
		expectedError     bool
	}{
		{
			name:              "constuctor",
			methodIsolation:   contract.ConstructorIsolation(),
			callIsolation:     contract.ConstructorIsolation(),
			expectedIsolation: contract.ConstructorIsolation(),
		},
		{
			name:            "bad constuctor",
			methodIsolation: contract.ConstructorIsolation(),
			callIsolation: contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallDirty,
			},
			expectedError: true,
		},
		{
			name: "method immutable",
			callIsolation: contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallValidated,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallValidated,
			},
			expectedIsolation: contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallValidated,
			},
		},
		{
			name: "method mutable",
			callIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
			expectedIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
		},
		{
			name: "mixed interference",
			callIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallDirty,
			},
			expectedIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
		},
		{
			name: "bad interference",
			callIsolation: contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallDirty,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
			expectedError: true,
		},
		{
			name: "mixed state",
			callIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallValidated,
			},
			expectedIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
		},
		{
			name: "bad state",
			callIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallValidated,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
			expectedError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				ctx = inslogger.TestContext(t)
				mc  = minimock.NewController(t)

				pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
				pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
				catalog         = object.NewCatalogMock(mc)
				smObjectID      = gen.UniqueIDWithPulse(pd.PulseNumber)
				smGlobalRef     = reference.NewSelf(smObjectID)
				smObject        = object.NewStateMachineObject(smGlobalRef)
				sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
			)

			smExecute := SMExecute{
				execution: execution.Context{
					Context: ctx,
					Isolation: contract.MethodIsolation{
						Interference: tc.callIsolation.Interference,
						State:        tc.callIsolation.State,
					},
				},
				objectCatalog:     catalog,
				pulseSlot:         &pulseSlot,
				objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},
				methodIsolation:   tc.methodIsolation,
			}

			stepChecker := testutils.NewSMStepChecker()
			{
				exec := SMExecute{}
				stepChecker.AddStep(exec.stepTakeLock)
			}

			execCtx := smachine.NewExecutionContextMock(mc)

			if tc.expectedError {
				// expected SM stop with Error
				execCtx.ErrorMock.Set(func(e1 error) (s1 smachine.StateUpdate) {
					require.Error(t, e1)
					return smachine.StateUpdate{}
				})
			} else {
				execCtx.JumpMock.Set(stepChecker.CheckJumpW(t))
			}

			smExecute.stepIsolationNegotiation(execCtx)

			if !tc.expectedError {
				assert.Equal(t, tc.expectedIsolation, smExecute.execution.Isolation)
			}

			mc.Finish()
		})
	}
}
