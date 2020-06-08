// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	messageSenderWrapper "github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

func TestSMStateReport_SendVStateReport_IfDescriptorSet(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject             = newSMObjectWithPulse()
		msgVStateReportCount = 0
	)

	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))
	smObject.SharedState.SetState(HasState)

	smReport := newSMReportWithPulse()
	smReport.report = smObject.BuildStateReport()
	smReport.report.ProvidedContent.LatestDirtyState = smObject.BuildLatestDirtyState()

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

	smReport.messageSender = messageSender.Mock()

	execCtx := smachine.NewExecutionContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{}).
		SetDefaultMigrationMock.Set(testutils.AssertMigration(t, smReport.migrationDefault))

	smReport.stepSendVStateReport(execCtx)
	require.Equal(t, 1, msgVStateReportCount)

	mc.Finish()
}

func TestSMStateReport_SendVStateReport_IfDescriptorNotSetAndStateEmpty(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject             = newSMObjectWithPulse()
		msgVStateReportCount = 0
	)
	smObject.SetState(Empty)

	smReport := newSMReportWithPulse()
	smReport.report = smObject.BuildStateReport()

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

	smReport.messageSender = messageSender.Mock()

	require.Equal(t, nil, smObject.descriptor)
	execCtx := smachine.NewExecutionContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{}).
		SetDefaultMigrationMock.Set(testutils.AssertMigration(t, smReport.migrationDefault))

	smReport.stepSendVStateReport(execCtx)
	require.Equal(t, 1, msgVStateReportCount)

	mc.Finish()
}
