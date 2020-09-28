// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine_test

import (
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/stretchr/testify/assert"
)

type testSMFinalize struct {
	smachine.StateMachineDeclTemplate

	nFinalizeCalls    int
	executionFunc smachine.StateFunc
}

func (s *testSMFinalize) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.stepInit
}

func (s *testSMFinalize) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *testSMFinalize) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetFinalizer(s.finalize)
	return ctx.Jump(s.executionFunc)
}

func (s *testSMFinalize) stepExecutionPanic(ctx smachine.ExecutionContext) smachine.StateUpdate {
	panic(throw.IllegalState())
	return ctx.Stop()
}

func (s *testSMFinalize) stepExecutionError(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Error(throw.New("Test error"))
}

func (s *testSMFinalize) stepExecutionStop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (s *testSMFinalize) finalize(ctx smachine.FinalizationContext) {
	s.nFinalizeCalls ++
	return
}

func TestSlotMachine_FinalizeTable(t *testing.T) {
	s := testSMFinalize{}

	table := []struct {
		name   string
		execFunc smachine.StateFunc
		nExpectedFinalizeRuns int
	}{
		{
			name:   "Call from Panic",
			execFunc: s.stepExecutionPanic,
			nExpectedFinalizeRuns: 1,
		}, {
			name:   "Call from Error",
			execFunc: s.stepExecutionError,
			nExpectedFinalizeRuns: 1,
		}, {
			name: "No Call from Stop",
			execFunc: s.stepExecutionStop,
			nExpectedFinalizeRuns: 0,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)
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

			s := testSMFinalize{}
			s.executionFunc = test.execFunc
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

			assert.Equal(t, test.nExpectedFinalizeRuns, s.nFinalizeCalls)
		})
	}
}
