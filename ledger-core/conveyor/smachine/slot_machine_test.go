// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type testSM struct {
	smachine.StateMachineDeclTemplate

	ok    bool
	notOk bool
}

func (s *testSM) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.stepInit
}

func (s *testSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *testSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.JumpExt(smachine.SlotStep{
		Transition: func(ctx smachine.ExecutionContext) smachine.StateUpdate {
			s.ok = true
			return ctx.Stop()
		},
		Migration: func(ctx smachine.MigrationContext) smachine.StateUpdate {
			s.notOk = true
			return ctx.Stop()
		},
	})
}

func TestSlotMachine_AddSMAndMigrate(t *testing.T) {
	ctx := instestlogger.TestContext(t)

	scanCountLimit := 1000

	signal := synckit.NewVersionedSignal()
	m := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:    1000,
		PollingPeriod:   10 * time.Millisecond,
		PollingTruncate: 1 * time.Microsecond,
		ScanCountLimit:  scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()

	s := testSM{}
	m.AddNew(ctx, &s, smachine.CreateDefaultValues{})
	if !m.ScheduleCall(func(callContext smachine.MachineCallContext) {
		callContext.Migrate(nil)
	}, true) {
		panic(throw.IllegalState())
	}

	// make 1 iteration
	for {
		var (
			repeatNow bool
		)

		workerFactory.AttachTo(m, neverSignal, uint32(scanCountLimit), func(worker smachine.AttachedSlotWorker) {
			repeatNow, _ = m.ScanOnce(0, worker)
		})

		if repeatNow {
			continue
		}

		break
	}

	assert.True(t, s.ok)
	assert.False(t, s.notOk)
}

type smTestRestoreStep struct {
	smachine.StateMachineDeclTemplate

	wasContinued bool
}

func (s *smTestRestoreStep) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.stepInit
}

func (s *smTestRestoreStep) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *smTestRestoreStep) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(s.migrate)
	return ctx.Jump(s.stepWaitInfinity)
}

func (s *smTestRestoreStep) stepWaitInfinity(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.wasContinued = true
	return ctx.Sleep().ThenRepeat()
}

func (s *smTestRestoreStep) migrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	s.wasContinued = false
	return ctx.RestoreStep(ctx.AffectedStep())
}

func TestSlotMachine_RestoreStep(t *testing.T) {
	ctx := instestlogger.TestContext(t)

	scanCountLimit := 1000

	signal := synckit.NewVersionedSignal()
	m := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:    1000,
		PollingPeriod:   10 * time.Millisecond,
		PollingTruncate: 1 * time.Microsecond,
		ScanCountLimit:  scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()

	s := smTestRestoreStep{}
	m.AddNew(ctx, &s, smachine.CreateDefaultValues{})

	require.False(t, s.wasContinued)

	if !m.ScheduleCall(func(callContext smachine.MachineCallContext) {}, true) {
		panic(throw.IllegalState())
	}

	iterFn := func() {
		for {
			var repeatNow bool
			workerFactory.AttachTo(m, neverSignal, uint32(scanCountLimit), func(worker smachine.AttachedSlotWorker) {
				repeatNow, _ = m.ScanOnce(0, worker)
			})
			if repeatNow {
				continue
			}
			break
		}
	}

	// make 1 iteration
	iterFn()

	require.True(t, s.wasContinued)

	if !m.ScheduleCall(func(callContext smachine.MachineCallContext) {
		callContext.Migrate(nil)
	}, true) {
		panic(throw.IllegalState())
	}
	iterFn()
	require.False(t, s.wasContinued)
}


type smReplaceAndMigrate1 struct {
	smachine.StateMachineDeclTemplate

	migrated bool
	executed bool
}

func (s *smReplaceAndMigrate1) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.Init
}

func (s *smReplaceAndMigrate1) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *smReplaceAndMigrate1) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepExecute)
}

func (s *smReplaceAndMigrate1) stepExecute(ctx smachine.ExecutionContext)  smachine.StateUpdate {
	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &smReplaceAndMigrate2{migrated: &s.migrated, executed: &s.executed}
	})
}

type smReplaceAndMigrate2 struct {
	smachine.StateMachineDeclTemplate

	migrated *bool
	executed *bool
}

func (s *smReplaceAndMigrate2) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.Init
}

func (s *smReplaceAndMigrate2) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *smReplaceAndMigrate2) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.JumpExt(smachine.SlotStep{
		Transition: 		s.stepExecute,
		Migration: func(ctx smachine.MigrationContext) smachine.StateUpdate {
			*s.migrated = true
			return ctx.Stop()
		},
	})
}

func (s *smReplaceAndMigrate2) stepExecute(ctx smachine.ExecutionContext)  smachine.StateUpdate {
	*s.executed = true
	return ctx.Stop()
}

func TestSlotMachine_ReplaceAndMigrate(t *testing.T) {
	ctx := instestlogger.TestContext(t)

	scanCountLimit := 1000

	signal := synckit.NewVersionedSignal()
	m := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:    1000,
		PollingPeriod:   10 * time.Millisecond,
		PollingTruncate: 1 * time.Microsecond,
		ScanCountLimit:  scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()

	s := smReplaceAndMigrate1{}
	m.AddNew(ctx, &s, smachine.CreateDefaultValues{})

	if !m.ScheduleCall(func(callContext smachine.MachineCallContext) {
		callContext.Migrate(nil)
	}, true) {
		panic(throw.IllegalState())
	}

	// make 1 iteration
	for {
		var (
			repeatNow bool
		)

		workerFactory.AttachTo(m, neverSignal, uint32(scanCountLimit), func(worker smachine.AttachedSlotWorker) {
			repeatNow, _ = m.ScanOnce(0, worker)
		})

		if repeatNow {
			continue
		}

		break
	}

	assert.True(t, s.migrated)
	assert.False(t, s.executed)
}
