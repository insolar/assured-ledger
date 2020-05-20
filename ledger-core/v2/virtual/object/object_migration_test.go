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
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

func TestSMObject_SendVStateReport_After_Migration(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		pd                   = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot            = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID           = gen.IDWithPulse(pd.PulseNumber)
		smGlobalRef          = reference.NewSelf(smObjectID)
		smObject             = NewStateMachineObject(smGlobalRef)
		msgVStateReportCount = 0
		sharedStateData      = smachine.NewUnboundSharedData(&smObject.SharedState)
	)
	defer mc.Finish()

	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepGetState)
		stepChecker.AddStep(exec.stepSendVStateReport)
		stepChecker.AddStep(exec.stepWaitIndefinitely)
	}
	defer func() { require.NoError(t, stepChecker.CheckDone()) }()

	smObject.SharedState.SetState(Empty)
	smObject.pulseSlot = &pulseSlot

	messageService := testutils.NewMessageServiceMock(mc)
	checkMessageFn := func(msg payload.Marshaler) {
		_, ok := msg.(*payload.VStateReport)
		require.True(t, ok)
		msgVStateReportCount++
	}
	messageService.SendRole.SetCheckMessage(checkMessageFn)
	messageSender := messageService.NewAdapterMock().SetDefaultPrepareAsyncCall(ctx)

	smObject.messageSender = messageSender.Mock()
	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))

	{ // check migration is set
		compareDefaultMigration := func(fn smachine.MigrateFunc) {
			require.True(t, testutils.CmpStateFuncs(smObject.migrateDuringExecution, fn))
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
		smObject.migrateDuringExecution(migrationCtx)
	}

	{ // check execution of step after migration
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t))
		smObject.stepSendVStateReport(execCtx)
	}

	require.Equal(t, 1, msgVStateReportCount)
}
