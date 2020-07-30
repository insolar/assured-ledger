// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
