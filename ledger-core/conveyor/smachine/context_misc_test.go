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

type TestFunc func(*testSMFinalize, smachine.ExecutionContext) smachine.StateUpdate
type TestFuncSR func(*testSMFinalizeSR, smachine.ExecutionContext) smachine.StateUpdate

type testSMFinalize struct {
	smachine.StateMachineDeclTemplate

	nFinalizeCallsLevel0  int
	nFinalizeCallsLevel1  int
	nFinalizeCallsLevel2  int
	executionFunc TestFunc
	executionFuncSR TestFuncSR
	executionFuncSRSR TestFuncSR
}

func (s *testSMFinalize) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.stepInit
}

func (s *testSMFinalize) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *testSMFinalize) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetFinalizer(s.finalize)
	return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		return s.executionFunc(s, ctx)
	})
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

func (s *testSMFinalize) stepExecutionWithSubroutine(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subroutineSM := &testSMFinalizeSR{executionFunc: s.executionFuncSR, executionFuncSR: s.executionFuncSRSR}
	return ctx.CallSubroutine(subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		s.nFinalizeCallsLevel1 = subroutineSM.nFinalizeCallsLevel0
		s.nFinalizeCallsLevel2 = subroutineSM.nFinalizeCallsLevel1
		return ctx.Jump(s.stepDone)
	})
}

func (s *testSMFinalize) stepDone(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (s *testSMFinalize) finalize(ctx smachine.FinalizationContext) {
	s.nFinalizeCallsLevel0 ++
	return
}

func TestSlotMachine_FinalizeTable(t *testing.T) {

	table := []struct {
		name   string
		execFunc TestFunc
		execFuncSR TestFuncSR
		execFuncSRSR TestFuncSR
		nExpectedFinalizeRunsLevel0 int
		nExpectedFinalizeRunsLevel1 int
		nExpectedFinalizeRunsLevel2 int
	}{
		{
			name:   "Call from Panic",
			execFunc: (*testSMFinalize).stepExecutionPanic,
			execFuncSR: nil,
			execFuncSRSR: nil,
			nExpectedFinalizeRunsLevel0: 1,
			nExpectedFinalizeRunsLevel1: 0,
			nExpectedFinalizeRunsLevel2: 0,

		}, {
			name:   "Call from Error",
			execFunc: (*testSMFinalize).stepExecutionError,
			execFuncSR: nil,
			execFuncSRSR: nil,
			nExpectedFinalizeRunsLevel0: 1,
			nExpectedFinalizeRunsLevel1: 0,
			nExpectedFinalizeRunsLevel2: 0,
		}, {
			name: "No Call from Stop",
			execFunc: (*testSMFinalize).stepExecutionStop,
			execFuncSR: nil,
			execFuncSRSR: nil,
			nExpectedFinalizeRunsLevel0: 0,
			nExpectedFinalizeRunsLevel1: 0,
			nExpectedFinalizeRunsLevel2: 0,
		}, {
			name: "Call from Subroutine with Stop",
			execFunc: (*testSMFinalize).stepExecutionWithSubroutine,
			execFuncSR: (*testSMFinalizeSR).StateStop,
			execFuncSRSR: nil,
			nExpectedFinalizeRunsLevel0: 0,
			nExpectedFinalizeRunsLevel1: 1,
			nExpectedFinalizeRunsLevel2: 0,
		}, {
			name: "Call from Subroutine with Error",
			execFunc: (*testSMFinalize).stepExecutionWithSubroutine,
			execFuncSR: (*testSMFinalizeSR).StateError,
			execFuncSRSR: nil,
			nExpectedFinalizeRunsLevel0: 0,
			nExpectedFinalizeRunsLevel1: 1,
			nExpectedFinalizeRunsLevel2: 0,
		}, {
			name: "Call from Subroutine with Panic",
			execFunc: (*testSMFinalize).stepExecutionWithSubroutine,
			execFuncSR: (*testSMFinalizeSR).StatePanic,
			execFuncSRSR: nil,
			nExpectedFinalizeRunsLevel0: 0,
			nExpectedFinalizeRunsLevel1: 1,
			nExpectedFinalizeRunsLevel2: 0,
		}, {
			name: "Call from Subroutine from Subroutine with Stop",
			execFunc: (*testSMFinalize).stepExecutionWithSubroutine,
			execFuncSR: (*testSMFinalizeSR).StateGoDeeper,
			execFuncSRSR: (*testSMFinalizeSR).StateStop,
			nExpectedFinalizeRunsLevel0: 0,
			nExpectedFinalizeRunsLevel1: 1,
			nExpectedFinalizeRunsLevel2: 1,
		}, {
			name: "Call from Subroutine from Subroutine with Error",
			execFunc: (*testSMFinalize).stepExecutionWithSubroutine,
			execFuncSR: (*testSMFinalizeSR).StateGoDeeper,
			execFuncSRSR: (*testSMFinalizeSR).StateError,
			nExpectedFinalizeRunsLevel0: 0,
			nExpectedFinalizeRunsLevel1: 1,
			nExpectedFinalizeRunsLevel2: 1,
		}, {
			name: "Call from Subroutine from Subroutine with Panic",
			execFunc: (*testSMFinalize).stepExecutionWithSubroutine,
			execFuncSR: (*testSMFinalizeSR).StateGoDeeper,
			execFuncSRSR: (*testSMFinalizeSR).StatePanic,
			nExpectedFinalizeRunsLevel0: 0,
			nExpectedFinalizeRunsLevel1: 1,
			nExpectedFinalizeRunsLevel2: 1,
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

			s := &testSMFinalize{}
			s.executionFunc = test.execFunc
			s.executionFuncSR = test.execFuncSR
			s.executionFuncSRSR = test.execFuncSRSR
			m.AddNew(ctx, s, smachine.CreateDefaultValues{})
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

			assert.Equal(t, test.nExpectedFinalizeRunsLevel0, s.nFinalizeCallsLevel0)
			assert.Equal(t, test.nExpectedFinalizeRunsLevel1, s.nFinalizeCallsLevel1)
			assert.Equal(t, test.nExpectedFinalizeRunsLevel2, s.nFinalizeCallsLevel2)
		})
	}
}


type testSMFinalizeSR struct {
	smachine.StateMachineDeclTemplate
	nFinalizeCallsLevel0 int
	nFinalizeCallsLevel1 int
	executionFunc TestFuncSR
	executionFuncSR TestFuncSR
}

func (testSMFinalizeSR) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*testSMFinalizeSR).Init
}

/* -------- Instance ------------- */

func (s *testSMFinalizeSR) GetSubroutineInitState(smachine.SubroutineStartContext) smachine.InitFunc {
	return s.Init
}

func (s *testSMFinalizeSR) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *testSMFinalizeSR) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetFinalizer(s.finalize)
	return ctx.Jump(s.State0)
}

func (s *testSMFinalizeSR) State0(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		return s.executionFunc(s, ctx)
	})
}

func (s *testSMFinalizeSR) StateError(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Error(throw.New("Test error"))
}

func (s *testSMFinalizeSR) StatePanic(ctx smachine.ExecutionContext) smachine.StateUpdate {
	panic(throw.IllegalState())
	return ctx.Jump(s.StateStop)
}

func (s *testSMFinalizeSR) StateStop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (s *testSMFinalizeSR) StateGoDeeper(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subroutineSM := &testSMFinalizeSR{executionFunc: s.executionFuncSR}
	return ctx.CallSubroutine(subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		s.nFinalizeCallsLevel1 = subroutineSM.nFinalizeCallsLevel0
		return ctx.Jump(s.StateStop)
	})
}

func (s *testSMFinalizeSR) finalize(ctx smachine.FinalizationContext) {
	s.nFinalizeCallsLevel0 ++
	return
}
