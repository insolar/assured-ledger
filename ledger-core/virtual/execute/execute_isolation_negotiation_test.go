package execute

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
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
				Interference: isolation.CallIntolerable,
				State:        isolation.CallDirty,
			},
			expectedError: true,
		},
		{
			name: "method immutable",
			callIsolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallValidated,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallValidated,
			},
			expectedIsolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallValidated,
			},
		},
		{
			name: "method mutable",
			callIsolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallDirty,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallDirty,
			},
			expectedIsolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallDirty,
			},
		},
		{
			name: "mixed interference",
			callIsolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallDirty,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallDirty,
			},
			expectedIsolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallDirty,
			},
		},
		{
			name: "bad interference",
			callIsolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallDirty,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallDirty,
			},
			expectedError: true,
		},
		{
			name: "mixed state",
			callIsolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallDirty,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallValidated,
			},
			expectedIsolation: contract.MethodIsolation{
				Interference: isolation.CallIntolerable,
				State:        isolation.CallDirty,
			},
		},
		{
			name: "bad state",
			callIsolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallValidated,
			},
			methodIsolation: contract.MethodIsolation{
				Interference: isolation.CallTolerable,
				State:        isolation.CallDirty,
			},
			expectedError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer executeLeakCheck(t)

			var (
				ctx = instestlogger.TestContext(t)
				mc  = minimock.NewController(t)

				pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
				pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
				smGlobalRef     = reference.NewRecordOf(gen.UniqueGlobalRefWithPulse(pd.PulseNumber), gen.UniqueLocalRefWithPulse(pd.PulseNumber))
				smObject        = object.NewStateMachineObject(smGlobalRef)
				sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
			)

			request := &rms.VCallRequest{
				CallType:       rms.CallTypeConstructor,
				CallFlags:      rms.BuildCallFlags(tc.callIsolation.Interference, tc.callIsolation.State),
				CallSiteMethod: "New",
				CallOutgoing:   rms.NewReference(gen.UniqueGlobalRefWithPulse(pd.PulseNumber)),
				Caller:         rms.NewReference(gen.UniqueGlobalRefWithPulse(pd.PulseNumber)),
				Callee:         rms.NewReference(smGlobalRef),
				Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
			}

			smExecute := SMExecute{
				execution: execution.Context{
					Context: ctx,
					Isolation: contract.MethodIsolation{
						Interference: tc.callIsolation.Interference,
						State:        tc.callIsolation.State,
					},
					ObjectDescriptor: descriptor.NewObject(
						reference.Global{}, reference.Local{}, smGlobalRef, []byte(""), false,
					),
				},
				Payload:           request,
				pulseSlot:         &pulseSlot,
				objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},
				methodIsolation:   tc.methodIsolation,
			}

			smExecute = expectedInitState(ctx, smExecute)

			execCtx := smachine.NewExecutionContextMock(mc)

			if tc.expectedError {
				// expected SM sends an error in stepSendCallResult
				execCtx.JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepSendCallResult))
			} else {
				execCtx.JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepDeduplicate))
			}

			smExecute.stepIsolationNegotiation(execCtx)

			if !tc.expectedError {
				assert.Equal(t, tc.expectedIsolation, smExecute.execution.Isolation)
			}

			mc.Finish()
		})
	}
}
