// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	messageSenderWrapper "github.com/insolar/assured-ledger/ledger-core/v2/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/stepchecker"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/utils"
)

func TestSMObject_MigrateFail_IfStateIsEmptyAndNoCounters(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject        = newSMObjectWithPulseAndDescriptor()
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	stepChecker := stepchecker.New()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepGetState)
	}

	smObject.SharedState.SetState(Empty)

	{ // check Init set migration and Jmp to stepGetState
		compareDefaultMigration := func(fn smachine.MigrateFunc) {
			require.True(t, utils.CmpStateFuncs(smObject.migrate, fn))
		}
		initCtx := smachine.NewInitializationContextMock(mc).
			ShareMock.Return(sharedStateData).
			PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			SetDefaultMigrationMock.Set(compareDefaultMigration)

		smObject.Init(initCtx)
	}

	{ // check migration will panic if state is Empty and no pending counters set
		migrationCtx := smachine.NewMigrationContextMock(mc).
			AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely})

		require.Panics(t, func() { smObject.migrate(migrationCtx) })
	}

	{ // finish test
		require.NoError(t, stepChecker.CheckDone())
		mc.Finish()
	}
}

func TestSMObject_SendVStateUnavailableAfterMigration_IfStateMissing(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject             = newSMObjectWithPulseAndDescriptor()
		msgVStateUnavailable = 0
		sharedStateData      = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	stepChecker := stepchecker.New()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepGetState)
		stepChecker.AddStep(exec.stepSendVStateUnavailable)
		stepChecker.AddStep(exec.stepWaitIndefinitely)
	}

	smObject.SharedState.SetState(Missing)
	smObject.IncrementPotentialPendingCounter(contract.MethodIsolation{
		Interference: contract.CallIntolerable,
		State:        contract.CallValidated,
	})

	messageService := messageSenderWrapper.NewServiceMockWrapper(mc)
	checkMessageFn := func(msg payload.Marshaler) {
		_, ok := msg.(*payload.VStateUnavailable)
		require.True(t, ok)
		msgVStateUnavailable++
	}
	messageService.SendRole.SetCheckMessage(checkMessageFn)
	messageSender := messageService.NewAdapterMock().SetDefaultPrepareAsyncCall(inslogger.TestContext(t))

	smObject.messageSender = messageSender.Mock()

	{ // check migration is set
		compareDefaultMigration := func(fn smachine.MigrateFunc) {
			require.True(t, utils.CmpStateFuncs(smObject.migrate, fn))
		}
		initCtx := smachine.NewInitializationContextMock(mc).
			ShareMock.Return(sharedStateData).
			PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
			SetDefaultMigrationMock.Set(compareDefaultMigration).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.Init(initCtx)
	}

	{ // check migration is successful
		migrationCtx := smachine.NewMigrationContextMock(mc).
			AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely}).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.migrate(migrationCtx)
	}

	{ // check execution of step after migration
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.stepSendVStateUnavailable(execCtx)
		require.Equal(t, 1, msgVStateUnavailable)
	}

	{ // finish test
		require.NoError(t, stepChecker.CheckDone())
		mc.Finish()
	}
}

func TestSMObject_SendVStateReportAfterMigration_IfStateEmptyAndCountersSet(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject             = newSMObjectWithPulseAndDescriptor()
		msgVStateReportCount = 0
		sharedStateData      = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	stepChecker := stepchecker.New()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepGetState)
		stepChecker.AddStep(exec.stepSendVStateReport)
		stepChecker.AddStep(exec.stepWaitIndefinitely)
	}

	smObject.SharedState.SetState(Empty)
	smObject.IncrementPotentialPendingCounter(contract.MethodIsolation{
		Interference: contract.CallIntolerable,
		State:        contract.CallValidated,
	})
	messageService := messageSenderWrapper.NewServiceMockWrapper(mc)
	checkMessageFn := func(msg payload.Marshaler) {
		_, ok := msg.(*payload.VStateReport)
		require.True(t, ok)
		msgVStateReportCount++
	}
	messageService.SendRole.SetCheckMessage(checkMessageFn)
	messageSender := messageService.NewAdapterMock().SetDefaultPrepareAsyncCall(inslogger.TestContext(t))

	smObject.messageSender = messageSender.Mock()

	{ // check migration is set
		compareDefaultMigration := func(fn smachine.MigrateFunc) {
			require.True(t, utils.CmpStateFuncs(smObject.migrate, fn))
		}
		initCtx := smachine.NewInitializationContextMock(mc).
			ShareMock.Return(sharedStateData).
			PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
			SetDefaultMigrationMock.Set(compareDefaultMigration).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.Init(initCtx)
	}

	{ // check migration is successful
		migrationCtx := smachine.NewMigrationContextMock(mc).
			AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely}).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.migrate(migrationCtx)
	}

	{ // check execution of step after migration
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.stepSendVStateReport(execCtx)
		require.Equal(t, 1, msgVStateReportCount)
	}

	{ // finish test
		require.NoError(t, stepChecker.CheckDone())
		mc.Finish()
	}
}

func TestSMObject_SendNothingAfterMigration_IfStateUnknown(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject        = newSMObjectWithPulseAndDescriptor()
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	stepChecker := stepchecker.New()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepGetState)
	}

	{ // check migration is set
		compareDefaultMigration := func(fn smachine.MigrateFunc) {
			require.True(t, utils.CmpStateFuncs(smObject.migrate, fn))
		}
		initCtx := smachine.NewInitializationContextMock(mc).
			ShareMock.Return(sharedStateData).
			PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
			SetDefaultMigrationMock.Set(compareDefaultMigration).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smObject.Init(initCtx)
	}

	{ // check migration is successful
		migrationCtx := smachine.NewMigrationContextMock(mc).
			AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely}).
			LogMock.Return(smachine.Logger{}).
			StayMock.Return(smachine.StateUpdate{})

		smObject.migrate(migrationCtx)
	}

	{ // finish test
		require.NoError(t, stepChecker.CheckDone())
		mc.Finish()
	}
}

func TestSMObject_DoNotSendVStateReportTwiceButIncMigrationCounter(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject             = newSMObjectWithPulseAndDescriptor()
		msgVStateReportCount = 0
	)
	smObject.SetState(HasState)

	messageService := messageSenderWrapper.NewServiceMockWrapper(mc)
	checkMessageFn := func(msg payload.Marshaler) {
		_, ok := msg.(*payload.VStateReport)
		require.True(t, ok)
		msgVStateReportCount++
	}
	messageService.SendRole.SetCheckMessage(checkMessageFn)
	messageSender := messageService.NewAdapterMock().SetDefaultPrepareAsyncCall(inslogger.TestContext(t))

	smObject.messageSender = messageSender.Mock()

	{ // first call send VStateReport and increment stateSentCount
		require.Equal(t, stateWasNotSend, smObject.migrateState)
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Return(smachine.StateUpdate{})

		smObject.stepSendVStateReport(execCtx)
		require.Equal(t, 1, msgVStateReportCount)
		require.Equal(t, stateSent, smObject.migrateState)
	}

	{ // next calls do not send VStateReport but increment stateSentCount
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Return(smachine.StateUpdate{})

		smObject.stepSendVStateReport(execCtx)
		require.Equal(t, 1, msgVStateReportCount)
		require.Equal(t, readyToStop, smObject.migrateState)
	}

	{ // finish test
		mc.Finish()
	}
}

func TestSMObject_StopIfVStateReportWasSendAndNoPendingExecution(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulseAndDescriptor()
	)

	{ // stepWaitIndefinitely continue if not migrated and no pending exist
		smObject.migrateState = stateWasNotSend
		smObject.PotentialMutablePendingCount = 0
		smObject.PotentialImmutablePendingCount = 0

		cb := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Return(smachine.StateUpdate{})
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Return(cb)

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // stepWaitIndefinitely continue if not migrated and no pending exist
		smObject.migrateState = stateSent
		smObject.PotentialMutablePendingCount = 0
		smObject.PotentialImmutablePendingCount = 0

		cb := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Return(smachine.StateUpdate{})
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Return(cb)

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // stepWaitIndefinitely continue if migrated and pending exist
		smObject.migrateState = readyToStop
		smObject.PotentialMutablePendingCount = 1
		smObject.PotentialImmutablePendingCount = 0

		cb := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Return(smachine.StateUpdate{})
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Return(cb)

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // stepWaitIndefinitely continue if not migrated and no pending exist
		smObject.migrateState = readyToStop
		smObject.PotentialMutablePendingCount = 0
		smObject.PotentialImmutablePendingCount = 1

		cb := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Return(smachine.StateUpdate{})
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Return(cb)

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // stepWaitIndefinitely call ctx.Stop if migrated and no pending exist
		smObject.migrateState = readyToStop
		smObject.PotentialMutablePendingCount = 0
		smObject.PotentialImmutablePendingCount = 0

		execCtx := smachine.NewExecutionContextMock(mc).
			StopMock.Return(smachine.StateUpdate{})

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // finish test
		mc.Finish()
	}
}

func newSMObjectWithPulseAndDescriptor() *SMObject {
	var (
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
		smObject    = NewStateMachineObject(smGlobalRef)
	)

	smObject.pulseSlot = &pulseSlot
	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))

	return smObject
}
