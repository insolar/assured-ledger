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
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object/finalizedstate"
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

func TestSMObject_MigrationStop_IfStateIsEmptyAndNoCounters(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulse()
	)

	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))
	smObject.SharedState.SetState(Empty)

	migrationCtx := smachine.NewMigrationContextMock(mc).
		StopMock.Return(smachine.StateUpdate{}).
		LogMock.Return(smachine.Logger{})

	smObject.migrate(migrationCtx)

	mc.Finish()
}

func TestSMObject_MigrationCreateStateReport_IfStateMissing(t *testing.T) {
	mc := minimock.NewController(t)

	smObject := newSMObjectWithPulse()

	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, reference.Global{}, nil, reference.Global{}))
	smObject.SharedState.SetState(Missing)
	smObject.IncrementPotentialPendingCounter(contract.MethodIsolation{
		Interference: contract.CallIntolerable,
		State:        contract.CallValidated,
	})

	report := smObject.BuildStateReport()

	migrationCtx := smachine.NewMigrationContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{}).UnpublishAllMock.Return().
		ShareMock.Return(smachine.NewUnboundSharedData(&report)).
		PublishMock.Set(func(key interface{}, data interface{}) (b1 bool) {
		assert.Equal(t, finalizedstate.BuildReportKey(report.Object, smObject.pulseSlot.PulseData().PulseNumber), key)
		assert.NotNil(t, data)
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

func TestSMObject_MigrationCreateStateReport_IfStateEmptyAndCountersSet(t *testing.T) {
	var (
		mc = minimock.NewController(t)

		smObject = newSMObjectWithPulse()
	)

	smObject.SharedState.SetState(Empty)
	smObject.IncrementPotentialPendingCounter(contract.MethodIsolation{
		Interference: contract.CallIntolerable,
		State:        contract.CallValidated,
	})

	report := smObject.BuildStateReport()

	migrationCtx := smachine.NewMigrationContextMock(mc).
		JumpMock.Return(smachine.StateUpdate{}).UnpublishAllMock.Return().
		ShareMock.Return(smachine.NewUnboundSharedData(&report)).
		PublishMock.Set(func(key interface{}, data interface{}) (b1 bool) {
		assert.Equal(t, finalizedstate.BuildReportKey(report.Object, smObject.pulseSlot.PulseData().PulseNumber), key)
		assert.NotNil(t, data)
		return true
	})

	smObject.migrate(migrationCtx)

	mc.Finish()
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
