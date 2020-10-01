// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine_test

import (
	"context"
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

	helper := newTestsHelper()
	s := testSM{}
	helper.add(ctx, &s)
	helper.migrate()
	helper.iter(nil)

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

	helper := newTestsHelper()
	s := smTestRestoreStep{}
	helper.add(ctx, &s)
	require.False(t, s.wasContinued)

	if !helper.m.ScheduleCall(func(callContext smachine.MachineCallContext) {}, true) {
		panic(throw.IllegalState())
	}

	// make 1 iteration
	helper.iter(nil)

	require.True(t, s.wasContinued)

	helper.migrate()
	helper.iter(nil)

	require.False(t, s.wasContinued)
}

type smReplaceAndMigrate1 struct {
	smachine.StateMachineDeclTemplate

	replaced  bool
	migrated1 bool
	migrated2 bool
	executed  bool
}

func (s *smReplaceAndMigrate1) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.Init
}

func (s *smReplaceAndMigrate1) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *smReplaceAndMigrate1) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(func(ctx smachine.MigrationContext) smachine.StateUpdate {
		s.migrated1 = true
		return ctx.Stop()
	})
	return ctx.Jump(s.stepExecute)
}

func (s *smReplaceAndMigrate1) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.replaced = true
	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &smReplaceAndMigrate2{migrated: &s.migrated2, executed: &s.executed}
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
		Transition: s.stepExecute,
		Migration: func(ctx smachine.MigrationContext) smachine.StateUpdate {
			*s.migrated = true
			return ctx.Stop()
		},
	})
}

func (s *smReplaceAndMigrate2) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
	*s.executed = true
	return ctx.Stop()
}

func TestSlotMachine_ReplaceAndMigrate(t *testing.T) {
	ctx := instestlogger.TestContext(t)

	helper := newTestsHelper()
	s := smReplaceAndMigrate1{}
	helper.add(ctx, &s)
	helper.iter(func() bool {
		return !s.replaced
	})

	assert.True(t, s.replaced)

	helper.migrate()
	helper.iter(nil)

	assert.False(t, s.migrated1)
	assert.True(t, s.migrated2)
	assert.False(t, s.executed)
}

type testsHelper struct {
	scanCountLimit int
	m              *smachine.SlotMachine
	wFactory       *sworker.AttachableSimpleSlotWorker
	neverSignal    *synckit.SignalVersion
}

func newTestsHelper() *testsHelper {
	res := testsHelper{}
	res.scanCountLimit = 1000

	signal := synckit.NewVersionedSignal()
	res.m = smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:    1000,
		PollingPeriod:   10 * time.Millisecond,
		PollingTruncate: 1 * time.Microsecond,
		ScanCountLimit:  res.scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	res.wFactory = sworker.NewAttachableSimpleSlotWorker()
	res.neverSignal = synckit.NewNeverSignal()

	return &res
}

func (h *testsHelper) add(ctx context.Context, sm smachine.StateMachine) {
	h.m.AddNew(ctx, sm, smachine.CreateDefaultValues{})
}

func (h *testsHelper) iter(fn func() bool) {
	for repeatNow := true; repeatNow && (fn == nil || fn()); {
		h.wFactory.AttachTo(h.m, h.neverSignal, uint32(h.scanCountLimit), func(worker smachine.AttachedSlotWorker) {
			repeatNow, _ = h.m.ScanOnce(0, worker)
		})
	}
}

func (h *testsHelper) migrate() {
	if !h.m.ScheduleCall(func(callContext smachine.MachineCallContext) {
		callContext.Migrate(nil)
	}, true) {
		panic(throw.IllegalState())
	}
}
