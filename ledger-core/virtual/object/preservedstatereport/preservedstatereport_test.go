// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package preservedstatereport

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	messageSenderWrapper "github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

func buildStateReport(status payload.VStateReport_StateStatus, state descriptor.Object) payload.VStateReport {
	var (
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		smObjectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
	)

	res := payload.VStateReport{
		Object:          smGlobalRef,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{},
		Status:          status,
	}

	if state != nil {
		res.LatestDirtyState = state.HeadRef()
		class, _ := state.Class()
		res.ProvidedContent.LatestDirtyState = &payload.ObjectState{
			Reference: state.StateID(),
			Class:     class,
			State:     state.Memory(),
		}
	}
	return res
}

func newSMReportWithPulse() *SMPreservedStateReport {
	var (
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
		smReport    = &SMPreservedStateReport{
			Reference: smGlobalRef,
			pulseSlot: &pulseSlot,
		}
	)

	return smReport
}

func TestSMStateReport_SendVStateReport_IfDescriptorSet(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc = minimock.NewController(t)

		msgVStateReportCount = 0
	)

	smReport := newSMReportWithPulse()
	smReport.Report = buildStateReport(payload.StateStatusReady, descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, false))

	messageService := messageSenderWrapper.NewServiceMockWrapper(mc)
	checkMessageFn := func(msg payload.Marshaler) {
		stateReport, ok := msg.(*payload.VStateReport)
		require.True(t, ok)
		require.NotNil(t, stateReport.ProvidedContent)
		require.NotNil(t, stateReport.ProvidedContent.LatestDirtyState)
		msgVStateReportCount++
	}
	messageService.SendRole.SetCheckMessage(checkMessageFn)
	messageSender := messageService.NewAdapterMock().SetDefaultPrepareAsyncCall(instestlogger.TestContext(t))

	smReport.messageSender = messageSender.Mock()

	execCtx := smachine.NewExecutionContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{}).
		SetDefaultMigrationMock.Set(testutils.AssertMigration(t, smReport.migrationDefault))

	smReport.stepSendVStateReport(execCtx)
	require.Equal(t, 1, msgVStateReportCount)

	mc.Finish()
}

func TestSMStateReport_SendVStateReport_IfDescriptorNotSetAndStateEmpty(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc = minimock.NewController(t)

		msgVStateReportCount = 0
	)

	smReport := newSMReportWithPulse()
	smReport.Report = buildStateReport(payload.StateStatusEmpty, nil)

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
	messageSender := messageService.NewAdapterMock().SetDefaultPrepareAsyncCall(instestlogger.TestContext(t))

	smReport.messageSender = messageSender.Mock()

	execCtx := smachine.NewExecutionContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{}).
		SetDefaultMigrationMock.Set(testutils.AssertMigration(t, smReport.migrationDefault))

	smReport.stepSendVStateReport(execCtx)
	require.Equal(t, 1, msgVStateReportCount)

	mc.Finish()
}
