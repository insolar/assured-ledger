// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

func TestSMObject_SendVStateReport_After_Migration(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		pd                   = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot            = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID           = gen.IDWithPulse(pd.PulseNumber)
		smGlobalRef          = reference.NewSelf(smObjectID)
		smObject             = NewStateMachineObject(smGlobalRef)
		msgVStateReportCount = 0
		sharedStateData      = smachine.NewUnboundSharedData(&smObject.SharedState)
	)
	defer mc.Finish()
	smObject.SharedState.SetState(Empty)
	smObject.pulseSlot = &pulseSlot
	messageService := messagesender.NewServiceMock(mc).
		SendRoleMock.Set(
		func(ctx context.Context, msg payload.Marshaler, role insolar.DynamicRole, object reference.Global,
			pn insolar.PulseNumber, opts ...messagesender.SendOption) (err error) {
			_, ok := msg.(*payload.VStateReport)
			require.True(t, ok)
			msgVStateReportCount++
			return nil
		})
	messageSender := messageSenderAdapter.NewMessageSenderMock(mc).
		PrepareNotifyMock.Set(func(e1 smachine.ExecutionContext, fn func(context.Context, messagesender.Service)) (n1 smachine.NotifyRequester) {
		fn(context.Background(), messageService)
		return smachine.NewNotifyRequesterMock(mc).SendMock.Return()
	})
	smObject.messageSender = messageSender
	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))

	stepChecker := testutils.NewSMStepChecker()
	{
		exec := SMObject{}
		stepChecker.AddStep(exec.stepGetState)
		stepChecker.AddStep(exec.stepSendVStateReport)
		stepChecker.AddStep(exec.stepWaitIndefinitely)
	}
	initCtx := smachine.NewInitializationContextMock(mc).
		ShareMock.Return(sharedStateData).
		PublishMock.Expect(smObject.Reference.String(), sharedStateData).Return(true).
		JumpMock.Set(testutils.CheckWrapper(stepChecker, t)).
		SetDefaultMigrationMock.Set(func(fn smachine.MigrateFunc) {
		require.True(t, testutils.CmpStateFuncs(smObject.migrateDuringExecution, fn))
	})
	smObject.Init(initCtx)

	migrationCtx := smachine.NewMigrationContextMock(mc).
		AffectedStepMock.Return(smachine.SlotStep{Transition: smObject.stepWaitIndefinitely}).
		JumpMock.Set(testutils.CheckWrapper(stepChecker, t))
	smObject.migrateDuringExecution(migrationCtx)

	execCtx := smachine.NewExecutionContextMock(mc).
		JumpMock.Set(testutils.CheckWrapper(stepChecker, t))
	smObject.stepSendVStateReport(execCtx)
	require.Equal(t, 1, msgVStateReportCount)
}
