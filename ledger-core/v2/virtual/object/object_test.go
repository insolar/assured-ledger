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

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/stepchecker"
)

func Test_Delay(t *testing.T) {
	mc := minimock.NewController(t)

	var (
		pd          = pulse.NewPulsarData(pulse.OfNow(), 10, 10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueIDWithPulse(pd.PulseNumber)
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
			PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
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
	mc := minimock.NewController(t)

	var (
		pd          = pulse.NewPulsarData(pulse.OfNow(), 10, 10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueIDWithPulse(pd.PulseNumber)
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
			PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			SetDefaultMigrationMock.Return()

		smObject.Init(initCtx)
	}

	smObject.SharedState.ActiveMutablePendingCount = 1
	smObject.SharedState.SetState(HasState)

	{ // we should be able to start
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.stepGetState(execCtx)
	}

	{ // check that we jump to creation of child SM
		checkTypeAndCall := func(cb1 smachine.CreateFunc, cb2 smachine.PostInitFunc) smachine.SlotLink {
			constructionCtxMock := smachine.NewConstructionContextMock(t)

			resultSM := cb1(constructionCtxMock)
			if assert.IsType(t, resultSM, &SMAwaitDelegate{}) &&
				assert.Equal(t, smObject.MutableExecute, resultSM.(*SMAwaitDelegate).sync) {

				cb2()
			}
			return smachine.SlotLink{}
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			InitChildWithPostInitMock.Set(checkTypeAndCall)

		smObject.stepGotState(execCtx)
	}

	assert.NotNil(t, smObject.AwaitPendingOrdered)

	active, inactive := smObject.readyToWorkCtl.SyncLink().GetCounts()
	assert.Equal(t, -1, active)
	assert.Equal(t, 0, inactive)

	mc.Finish()
}

func TestSMObject_Semi_CheckAwaitDelegateIsStarted(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		objectReference = gen.UniqueReference()
		smObject        = NewStateMachineObject(objectReference)
	)

	smObject.SetState(HasState)
	smObject.ActiveMutablePendingCount = 1

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.PrepareMockedMessageSender(mc)

	{
		pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot := conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		slotMachine.AddDependency(&pulseSlot)
	}

	smWrapper := slotMachine.AddStateMachine(ctx, smObject)

	slotMachine.Start()
	defer slotMachine.Stop()

	require.Equal(t, 1, slotMachine.GetOccupiedSlotCount())

	slotMachine.Run(smWrapper.TilStep(smObject.stepReadyToWork))

	require.Equal(t, 2, slotMachine.GetOccupiedSlotCount())

	mc.Finish()
}
