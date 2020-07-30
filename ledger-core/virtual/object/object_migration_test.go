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
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object/finalizedstate"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
)

func TestSMObject_InitSetMigration(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc = minimock.NewController(t)

		smObject        = newSMObjectWithPulse()
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
	)
	smObject.globalLimiter = tool.NewRunnerLimiter(4)

	compareDefaultMigration := func(fn smachine.MigrateFunc) {
		require.True(t, commontestutils.CmpStateFuncs(smObject.migrate, fn))
	}
	initCtx := smachine.NewInitializationContextMock(mc).
		ShareMock.Return(sharedStateData).
		PublishMock.Expect(smObject.Reference, sharedStateData).Return(true).
		JumpMock.Return(smachine.StateUpdate{}).
		SetDefaultMigrationMock.Set(compareDefaultMigration)

	smObject.Init(initCtx)

	mc.Finish()
}

func TestSMObject_MigrationCreateStateReport_IfStateMissing(t *testing.T) {
	defer commontestutils.LeakTester(t)

	mc := minimock.NewController(t)

	smObject := newSMObjectWithPulse()

	smObject.SetDescriptorDirty(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil))
	smObject.SharedState.SetState(Missing)

	report := smObject.BuildStateReport()

	migrationCtx := smachine.NewMigrationContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{}).UnpublishAllMock.Return().
		LogMock.Return(smachine.Logger{}).
		ShareMock.Return(smachine.NewUnboundSharedData(&report)).
		PublishMock.Set(func(key interface{}, data interface{}) (b1 bool) {
		assert.NotNil(t, data)
		switch d := data.(type) {
		case finalizedstate.ReportKey:
			assert.Equal(t, finalizedstate.BuildReportKey(report.Object), d)
		case callsummary.SummarySyncKey:
			assert.Equal(t, callsummary.BuildSummarySyncKey(report.Object), d)
		}
		return true
	})

	smObject.migrate(migrationCtx)

	mc.Finish()
}

func TestSMObject_MigrationStop_IfStateUnknown(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulse()
	)
	smObject.SetState(Unknown)

	migrationCtx := smachine.NewMigrationContextMock(mc).
		StopMock.Return(smachine.StateUpdate{}).
		LogMock.Return(smachine.Logger{})

	smObject.migrate(migrationCtx)

	mc.Finish()
}

func TestSMObject_MigrationCreateStateReport_IfStateIsEmptyAndNoCounters(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulse()
	)

	smObject.SetDescriptorDirty(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil))
	smObject.SharedState.SetState(Empty)

	var sharedData smachine.SharedDataLink
	migrationCtx := smachine.NewMigrationContextMock(mc).
		LogMock.Return(smachine.Logger{}).
		UnpublishAllMock.Return().
		ShareMock.Set(
		func(data interface{}, flags smachine.ShareDataFlags) (s1 smachine.SharedDataLink) {
			switch data.(type) {
			case *payload.VStateReport:
			case *smachine.SyncLink:
				// no-op
			default:
				t.Fatal("Unexpected data type")
			}
			require.Equal(t, smachine.ShareDataFlags(0), flags)
			sharedData = smachine.NewUnboundSharedData(data)
			return sharedData
		}).
		PublishMock.Set(
		func(key interface{}, data interface{}) (b1 bool) {
			switch key.(type) {
			case finalizedstate.ReportKey:
				refKey := finalizedstate.BuildReportKey(smObject.Reference)
				require.Equal(t, refKey, key)
				require.Equal(t, sharedData, data)
			case callsummary.SummarySyncKey:
				assert.Equal(t, callsummary.BuildSummarySyncKey(smObject.Reference), key)
				require.Equal(t, sharedData, data)
			}
			return true
		}).
		JumpMock.Return(smachine.StateUpdate{})

	smObject.migrate(migrationCtx)

	mc.Finish()
}

func TestSMObject_MigrationCreateStateReport_IfStateEmptyAndCountersSet(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulse()
	)

	smObject.SharedState.SetState(Empty)

	report := smObject.BuildStateReport()

	publishDuringMigrate := func(key interface{}, data interface{}) (b1 bool) {
		assert.NotNil(t, data)

		switch k := key.(type) {
		case finalizedstate.ReportKey:
			assert.Equal(t, finalizedstate.BuildReportKey(report.Object), k)
		case callsummary.SummarySyncKey:
			assert.Equal(t, callsummary.BuildSummarySyncKey(report.Object), k)
		default:
			t.Fatal("Unexpected published key")
		}

		switch data.(type) {
		case smachine.SharedDataLink:
		default:
			t.Fatal("Unexpected published data")
		}

		return true
	}

	migrationCtx := smachine.NewMigrationContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{}).UnpublishAllMock.Return().
		LogMock.Return(smachine.Logger{}).
		ShareMock.Return(smachine.NewUnboundSharedData(&report)).
		PublishMock.Set(publishDuringMigrate)

	smObject.migrate(migrationCtx)

	mc.Finish()
}

func newSMObjectWithPulse() *SMObject {
	var (
		pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID  = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef = reference.NewSelf(smObjectID)
		smObject    = NewStateMachineObject(smGlobalRef)
	)

	smObject.pulseSlot = &pulseSlot

	return smObject
}
