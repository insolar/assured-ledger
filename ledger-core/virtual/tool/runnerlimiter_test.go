// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type runnerLimiterSM struct {
	smachine.StateMachineDeclTemplate

	// DI
	limiter  RunnerLimiter
	t        *testing.T
	testDone chan struct{}
}

func (sm *runnerLimiterSM) GetInitStateFor(machine smachine.StateMachine) smachine.InitFunc {
	if sm != machine {
		panic("illegal value")
	}
	return sm.stepInit
}

func (sm *runnerLimiterSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *runnerLimiterSM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&sm.limiter)
	injector.MustInject(&sm.t)
	injector.MustInject(&sm.testDone)
}

func (sm *runnerLimiterSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {

	{ // direct semaphore SyncLink can be acquired and released
		require.True(sm.t, ctx.Acquire(sm.limiter.semaphore.SyncLink()).IsPassed())
		require.True(sm.t, ctx.Release(sm.limiter.semaphore.SyncLink()))
	}
	child := sm.limiter.NewChildSemaphore(1, "Child_Semaphore")
	{ // hierarchical semaphore SyncLink can be acquired and released
		require.True(sm.t, ctx.Acquire(child.SyncLink()).IsPassed())
		require.True(sm.t, ctx.Release(child.SyncLink()))
	}
	{ // partial release and acquire on PartialLink
		require.True(sm.t, ctx.Acquire(child.SyncLink()).IsPassed())
		require.True(sm.t, ctx.Release(sm.limiter.PartialLink()))
		require.True(sm.t, ctx.Acquire(sm.limiter.PartialLink()).IsPassed())
		require.True(sm.t, ctx.Release(child.SyncLink()))
	}
	{ // Acquire on limiter without partial release will panic
		require.Panics(sm.t, func() {
			ctx.Acquire(sm.limiter.PartialLink())
		})
	}

	return ctx.Jump(sm.stepDone)
}

func (sm *runnerLimiterSM) stepDone(ctx smachine.ExecutionContext) smachine.StateUpdate {
	close(sm.testDone)
	return ctx.Stop()
}

func newTestPulseConveyorWithLimiter(ctx context.Context, t *testing.T) (*conveyor.PulseConveyor, chan struct{}) {
	instestlogger.SetTestOutput(t)

	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:   500 * time.Millisecond,
		PollingTruncate: 1 * time.Millisecond,
		SlotPageSize:    1000,
		ScanCountLimit:  100000,
	}

	conv := conveyor.NewPulseConveyor(ctx, conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        100 * time.Millisecond,
		MinCachePulseAge:      50,
		MaxPastPulseAge:       100,
	}, func(_ context.Context, input conveyor.InputEvent, ic conveyor.InputContext) (conveyor.InputSetup, error) {
		require.Nil(t, input)
		return conveyor.InputSetup{
			CreateFn: func(ctx smachine.ConstructionContext) smachine.StateMachine {
				return &runnerLimiterSM{}
			}}, nil
	}, nil)

	emerChan := make(chan struct{})
	conv.StartWorker(emerChan, nil)
	return conv, emerChan
}

func TestParallelProcessingLimiter(t *testing.T) {
	defer testutils.LeakTester(t)

	ctx := context.Background()
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	startPn := pd.PulseNumber

	globalLimiter := NewRunnerLimiter(1)
	testDone := make(chan struct{})

	conv, emerChan := newTestPulseConveyorWithLimiter(ctx, t)

	defer func() {
		close(emerChan)
		conv.Stop()
	}()

	conv.AddDependency(globalLimiter)
	conv.AddDependency(t)
	conv.AddDependency(testDone)
	require.NoError(t, conv.CommitPulseChange(pd.AsRange(), time.Now(), nil))
	require.NoError(t, conv.AddInput(ctx, startPn, nil))
	select {
	case <-testDone:
	case <-time.After(10 * time.Second):
		close(testDone)
		require.FailNow(t, "timeout")
	}
}
