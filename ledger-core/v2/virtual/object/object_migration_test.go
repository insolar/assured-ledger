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
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

func TestSMObject_MigrateFail_IfStateIsEmptyAndNoCounters(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject        = newSMObjectWithPulseAndDescriptor()
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepGetState)
	}

	smObject.SharedState.SetState(Empty)

	{ // check Init set migration and Jmp to stepGetState
		compareDefaultMigration := func(fn smachine.MigrateFunc) {
			require.True(t, testutils.CmpStateFuncs(smObject.migrate, fn))
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

	stepChecker := testutils.NewSMStepChecker()
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

	messageService := testutils.NewMessageServiceMock(mc)
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
			require.True(t, testutils.CmpStateFuncs(smObject.migrate, fn))
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

	stepChecker := testutils.NewSMStepChecker()
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
	messageService := testutils.NewMessageServiceMock(mc)
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
			require.True(t, testutils.CmpStateFuncs(smObject.migrate, fn))
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

	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepGetState)
	}

	{ // check migration is set
		compareDefaultMigration := func(fn smachine.MigrateFunc) {
			require.True(t, testutils.CmpStateFuncs(smObject.migrate, fn))
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

func newSMObjectWithPulseAndDescriptor() *SMObject {
	var (
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.IDWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
		smObject    = NewStateMachineObject(smGlobalRef)
	)

	smObject.pulseSlot = &pulseSlot
	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))

	return smObject
}
