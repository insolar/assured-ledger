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
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	messageSenderWrapper "github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

func buildStateReport(status rms.VStateReport_StateStatus, state descriptor.Object) rms.VStateReport {
	var (
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		smGlobalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
	)

	res := rms.VStateReport{
		Object:          rms.NewReference(smGlobalRef),
		ProvidedContent: &rms.VStateReport_ProvidedContentBody{},
		Status:          status,
	}

	if state != nil {
		res.LatestDirtyState.Set(state.HeadRef())
		res.ProvidedContent.LatestDirtyState = &rms.ObjectState{
			Reference: rms.NewReferenceLocal(state.State().GetLocal()),
			Class:     rms.NewReference(state.Class()),
			Memory:    rms.NewBytes(state.Memory()),
		}
	}
	return res
}

func newSMReportWithPulse() *SMPreservedStateReport {
	var (
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smGlobalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
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
	smReport.Report = buildStateReport(rms.StateStatusReady, descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, false))

	messageService := messageSenderWrapper.NewServiceMockWrapper(mc)
	checkMessageFn := func(msg rmsreg.GoGoSerializable) {
		stateReport, ok := msg.(*rms.VStateReport)
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
	smReport.Report = buildStateReport(rms.StateStatusEmpty, nil)

	messageService := messageSenderWrapper.NewServiceMockWrapper(mc)
	checkMessageFn := func(msg rmsreg.GoGoSerializable) {
		stateReport, ok := msg.(*rms.VStateReport)
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
