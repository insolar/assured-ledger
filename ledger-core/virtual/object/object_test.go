// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/stepchecker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func Test_Delay(t *testing.T) {
	mc := minimock.NewController(t)

	var (
		pd          = pulse.NewPulsarData(pulse.OfNow(), 10, 10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
	)

	smObject := NewStateMachineObject(smGlobalRef)
	smObject.pulseSlot = &pulseSlot
	sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

	stepChecker := stepchecker.New()
	{
		sm := SMObject{}
		stepChecker.AddStep(sm.stepGetState)
		stepChecker.AddRepeat()
		stepChecker.AddStep(sm.stepSendStateRequest)
	}
	defer func() { require.NoError(t, stepChecker.CheckDone()) }()

	{ // initialization
		initCtx := smachine.NewInitializationContextMock(mc).
			ShareMock.Return(sharedStateData).
			PublishMock.Expect(smObject.Reference, sharedStateData).Return(true).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			SetDefaultMigrationMock.Return()

		smObject.Init(initCtx)
	}

	{ // execution must wait
		resultJump := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Set(stepChecker.CheckRepeatW(t))
		execCtx := smachine.NewExecutionContextMock(mc).
			WaitAnyUntilMock.Return(resultJump)

		smObject.stepGetState(execCtx)
	}

	{ // now rewind time and execution must not wait
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.waitGetStateUntil = time.Now().Add(-1 * time.Second)

		smObject.stepGetState(execCtx)
	}

	mc.Finish()
}

func Test_PendingBlocksExecution(t *testing.T) {

	table := []struct {
		name                string
		pendings            uint8
		state               State
		orderedAdjustment   bool
		unorderedAdjustment bool
		bargeIn             bool
	}{
		{
			name:                "pending with state",
			pendings:            1,
			state:               HasState,
			unorderedAdjustment: true,
			bargeIn:             true,
		},
		{
			name:     "pending constructor",
			pendings: 1,
			state:    Empty,
			bargeIn:  true,
		},
		{
			name:                "no pendings",
			pendings:            0,
			state:               HasState,
			orderedAdjustment:   true,
			unorderedAdjustment: true,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {

			mc := minimock.NewController(t)

			var (
				pd          = pulse.NewPulsarData(pulse.OfNow(), 10, 10, longbits.Bits256{})
				pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
				smObjectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
				smGlobalRef = reference.NewSelf(smObjectID)
			)

			smObject := NewStateMachineObject(smGlobalRef)
			smObject.pulseSlot = &pulseSlot
			sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

			stepChecker := stepchecker.New()
			{
				sm := SMObject{}
				stepChecker.AddStep(sm.stepGetState)
				stepChecker.AddStep(sm.stepGotState)
				stepChecker.AddStep(sm.stepReadyToWork)
			}
			defer func() { assert.NoError(t, stepChecker.CheckDone()) }()

			{ // initialization
				initCtx := smachine.NewInitializationContextMock(mc).
					ShareMock.Return(sharedStateData).
					PublishMock.Expect(smObject.Reference, sharedStateData).Return(true).
					JumpMock.Set(stepChecker.CheckJumpW(t)).
					SetDefaultMigrationMock.Return()

				smObject.Init(initCtx)
			}

			smObject.SharedState.PreviousExecutorOrderedPendingCount = test.pendings
			smObject.SharedState.SetState(test.state)

			{ // we should be able to start
				execCtx := smachine.NewExecutionContextMock(mc).
					JumpMock.Set(stepChecker.CheckJumpW(t))

				smObject.stepGetState(execCtx)
			}

			{
				execCtx := smachine.NewExecutionContextMock(mc).
					JumpMock.Set(stepChecker.CheckJumpW(t))

				if test.bargeIn {
					execCtx.NewBargeInMock.Return(
						smachine.NewBargeInBuilderMock(mc).
							WithJumpMock.Return(smachine.NewNoopBargeIn(smachine.DeadStepLink())),
					)
				}

				if test.orderedAdjustment || test.unorderedAdjustment {
					execCtx.ApplyAdjustmentMock.Set(func(sa smachine.SyncAdjustment) bool {
						switch sa.String() {
						case "ordered calls[=1]":
							require.True(t, test.orderedAdjustment)
						case "unordered calls[=30]":
							require.True(t, test.unorderedAdjustment)
						default:
							require.Failf(t, "unexpected adjustment", "got '%s'", sa.String())
						}
						return true
					})
				}

				smObject.stepGotState(execCtx)
			}

			if test.bargeIn {
				assert.False(t, smObject.SignalPendingsFinished.IsZero())
			} else {
				assert.True(t, smObject.SignalPendingsFinished.IsZero())
			}

			active, inactive := smObject.readyToWorkCtl.SyncLink().GetCounts()
			assert.Equal(t, -1, active)
			assert.Equal(t, 0, inactive)

			mc.Finish()
		})
	}
}

func TestSMObject_stepGotState_Set_PendingListFilled(t *testing.T) {
	var (
		mc          = minimock.NewController(t)
		pd          = pulse.NewPulsarData(pulse.OfNow(), 10, 10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
	)

	smObject := NewStateMachineObject(smGlobalRef)
	smObject.pulseSlot = &pulseSlot
	sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

	sm := SMObject{}
	stepChecker := stepchecker.New()
	stepChecker.AddStep(sm.stepGetState)
	stepChecker.AddStep(sm.stepReadyToWork)

	defer func() { assert.NoError(t, stepChecker.CheckDone()) }()

	{ // initialization
		initCtx := smachine.NewInitializationContextMock(mc).
			ShareMock.Return(sharedStateData).
			PublishMock.Expect(smObject.Reference, sharedStateData).Return(true).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			SetDefaultMigrationMock.Return()

		smObject.Init(initCtx)
	}

	{

		smObject.SharedState.PreviousExecutorOrderedPendingCount = 0
		smObject.SharedState.PreviousExecutorUnorderedPendingCount = 0

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			ApplyAdjustmentMock.Return(true)

		smObject.stepGotState(execCtx)
	}
}

func TestSMObject_checkPendingCounters_DontChangeIt(t *testing.T) {
	var (
		pd          = pulse.NewPulsarData(pulse.OfNow(), 10, 10, longbits.Bits256{})
		smObjectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
	)

	smObject := NewStateMachineObject(smGlobalRef)
	smObject.PreviousExecutorUnorderedPendingCount = 2
	smObject.PreviousExecutorOrderedPendingCount = 2
	smObject.PendingTable.GetList(contract.CallIntolerable).Add(gen.UniqueGlobalRef())
	smObject.PendingTable.GetList(contract.CallTolerable).Add(gen.UniqueGlobalRef())
	smObject.checkPendingCounters(smachine.Logger{})
	require.Equal(t, uint8(2), smObject.PreviousExecutorUnorderedPendingCount)
	require.Equal(t, uint8(2), smObject.PreviousExecutorOrderedPendingCount)
}
