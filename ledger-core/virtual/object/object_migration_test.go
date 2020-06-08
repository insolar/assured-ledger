// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	messageSenderWrapper "github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/testutils/stepchecker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/utils"
)

func TestSMObject_InitSetMigration(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject        = newSMObjectWithPulse()
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
	)

	compareDefaultMigration := func(fn smachine.MigrateFunc) {
		require.True(t, utils.CmpStateFuncs(smObject.migrate, fn))
	}
	initCtx := smachine.NewInitializationContextMock(mc).
		ShareMock.Return(sharedStateData).
		PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
		JumpMock.Return(smachine.StateUpdate{}).
		SetDefaultMigrationMock.Set(compareDefaultMigration)

	smObject.Init(initCtx)

	mc.Finish()
}

func TestSMObject_Migration_IfStateIsEmptyAndNoCounters(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulse()
	)

	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))
	smObject.SharedState.SetState(Empty)

	migrationCtx := smachine.NewMigrationContextMock(mc).
		JumpMock.
		Inspect(assertRequiredMigration(t, smObject.stepSendVStateReport)).
		Return(smachine.StateUpdate{}).
		AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely})

	smObject.migrate(migrationCtx)

	mc.Finish()
}

func assertRequiredMigration(t *testing.T, expected smachine.StateFunc) func(gotJump smachine.StateFunc) {
	return func(gotJump smachine.StateFunc) {
		assert.True(t, utils.CmpStateFuncs(expected, gotJump))
	}
}

func TestSMObject_MigrationJmpToStepSendVStateUnavailable_IfStateMissing(t *testing.T) {
	mc := minimock.NewController(t)

	smObject := newSMObjectWithPulse()

	stepChecker := stepchecker.New()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepSendVStateReport)
	}

	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))
	smObject.SharedState.SetState(Missing)
	smObject.IncrementPotentialPendingCounter(contract.MethodIsolation{
		Interference: contract.CallIntolerable,
		State:        contract.CallValidated,
	})

	migrationCtx := smachine.NewMigrationContextMock(mc).
		AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely}).
		JumpMock.Set(stepChecker.CheckJumpW(t))

	smObject.migrate(migrationCtx)

	require.NoError(t, stepChecker.CheckDone())
	mc.Finish()
}

func TestSMObject_MigrationSendNothing_IfStateUnknown(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulse()
	)
	smObject.SetState(Unknown)

	migrationCtx := smachine.NewMigrationContextMock(mc).
		AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely}).
		LogMock.Return(smachine.Logger{}).
		StayMock.Return(smachine.StateUpdate{})

	smObject.migrate(migrationCtx)

	mc.Finish()
}

func TestSMObject_MigrationJmpToStepSendVStateReport_IfStateEmptyAndCountersSet(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulse()
	)

	stepChecker := stepchecker.New()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepSendVStateReport)
	}

	smObject.SharedState.SetState(Empty)
	smObject.IncrementPotentialPendingCounter(contract.MethodIsolation{
		Interference: contract.CallIntolerable,
		State:        contract.CallValidated,
	})

	migrationCtx := smachine.NewMigrationContextMock(mc).
		AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely}).
		JumpMock.Set(stepChecker.CheckJumpW(t))

	smObject.migrate(migrationCtx)

	require.NoError(t, stepChecker.CheckDone())
	mc.Finish()
}

func TestSMObject_SendVStateReport_IfDescriptorSet(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject             = newSMObjectWithPulse()
		msgVStateReportCount = 0
	)

	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))
	smObject.SharedState.SetState(HasState)

	messageService := messageSenderWrapper.NewServiceMockWrapper(mc)
	checkMessageFn := func(msg payload.Marshaler) {
		stateReport, ok := msg.(*payload.VStateReport)
		require.True(t, ok)
		require.True(t, ok)
		require.NotNil(t, stateReport.ProvidedContent)
		require.NotNil(t, stateReport.ProvidedContent.LatestDirtyState)
		msgVStateReportCount++
	}
	messageService.SendRole.SetCheckMessage(checkMessageFn)
	messageSender := messageService.NewAdapterMock().SetDefaultPrepareAsyncCall(inslogger.TestContext(t))

	smObject.messageSender = messageSender.Mock()

	execCtx := smachine.NewExecutionContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{})

	smObject.stepSendVStateReport(execCtx)
	require.Equal(t, 1, msgVStateReportCount)

	mc.Finish()
}

func TestSMObject_SendVStateReport_IfDescriptorNotSetAndStateEmpty(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject             = newSMObjectWithPulse()
		msgVStateReportCount = 0
	)
	smObject.SetState(Empty)

	messageService := messageSenderWrapper.NewServiceMockWrapper(mc)
	checkMessageFn := func(msg payload.Marshaler) {
		stateReport, ok := msg.(*payload.VStateReport)
		require.True(t, ok)
		require.True(t, stateReport.LatestDirtyState.IsZero())
		require.NotNil(t, stateReport.ProvidedContent)
		require.Nil(t, stateReport.ProvidedContent.LatestDirtyState)
		msgVStateReportCount++
	}
	messageService.SendRole.SetCheckMessage(checkMessageFn)
	messageSender := messageService.NewAdapterMock().SetDefaultPrepareAsyncCall(inslogger.TestContext(t))

	smObject.messageSender = messageSender.Mock()

	require.Equal(t, nil, smObject.descriptor)
	execCtx := smachine.NewExecutionContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{})

	smObject.stepSendVStateReport(execCtx)
	require.Equal(t, 1, msgVStateReportCount)

	mc.Finish()
}

func TestSMObject_DoNotSendVStateReportTwiceButIncMigrationCounter(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject             = newSMObjectWithPulse()
		msgVStateReportCount = 0
	)

	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))
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

	{ // first call send VStateReport
		require.Equal(t, stateWasNotSend, smObject.migrateState)
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Return(smachine.StateUpdate{})

		smObject.stepSendVStateReport(execCtx)
		require.Equal(t, 1, msgVStateReportCount)
		require.Equal(t, stateSent, smObject.migrateState)
	}

	{ // next calls do not send VStateReport
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

		smObject = newSMObjectWithPulse()
	)
	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))

	{ // stepWaitIndefinitely continue if not migrated and no pending exist
		smObject.migrateState = stateWasNotSend
		smObject.PotentialOrderedPendingCount = 0
		smObject.PotentialUnorderedPendingCount = 0

		cb := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Return(smachine.StateUpdate{})
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Return(cb)

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // stepWaitIndefinitely continue if not migrated and no pending exist
		smObject.migrateState = stateSent
		smObject.PotentialOrderedPendingCount = 0
		smObject.PotentialUnorderedPendingCount = 0

		cb := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Return(smachine.StateUpdate{})
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Return(cb)

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // stepWaitIndefinitely continue if migrated and pending exist
		smObject.migrateState = readyToStop
		smObject.PotentialOrderedPendingCount = 1
		smObject.PotentialUnorderedPendingCount = 0

		cb := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Return(smachine.StateUpdate{})
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Return(cb)

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // stepWaitIndefinitely continue if not migrated and no pending exist
		smObject.migrateState = readyToStop
		smObject.PotentialOrderedPendingCount = 0
		smObject.PotentialUnorderedPendingCount = 1

		cb := smachine.NewStateConditionalBuilderMock(mc).
			ThenRepeatMock.Return(smachine.StateUpdate{})
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Return(cb)

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // stepWaitIndefinitely call ctx.Stop if migrated and no pending exist
		smObject.migrateState = readyToStop
		smObject.PotentialOrderedPendingCount = 0
		smObject.PotentialUnorderedPendingCount = 0

		execCtx := smachine.NewExecutionContextMock(mc).
			StopMock.Return(smachine.StateUpdate{})

		smObject.stepWaitIndefinitely(execCtx)
	}

	{ // finish test
		mc.Finish()
	}
}

func newSMObjectWithPulse() *SMObject {
	var (
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
		smObject    = NewStateMachineObject(smGlobalRef)
	)

	smObject.pulseSlot = &pulseSlot

	return smObject
}
