// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package conveyor

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

var emptyEntropyFn = func() longbits.Bits256 {
	return longbits.Bits256{}
}

type emptySM struct {
	smachine.StateMachineDeclTemplate

	pulseSlot *PulseSlot

	pn         pulse.Number
	eventValue interface{}
	expiry     time.Time
	burner     bool
}

func (sm *emptySM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *emptySM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&sm.pulseSlot)
}

func (sm *emptySM) GetInitStateFor(machine smachine.StateMachine) smachine.InitFunc {
	if sm != machine {
		panic("illegal value")
	}
	return sm.stepInit
}

func (sm *emptySM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	if !sm.burner {
		return ctx.Stop()
	}
	return ctx.Jump(sm.burn)
}

func (sm *emptySM) burn(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Yield().ThenRepeat()
}

func handleFactory(_ context.Context, input InputEvent, _ InputContext) (InputSetup, error) {
	switch input.(type) {
	default:
		panic(fmt.Sprintf("unknown event type, got %T", input))
	}
}

func TestNewPulseConveyor(t *testing.T) {
	t.Run("bad input", func(t *testing.T) {
		require.Panics(t, func() {
			NewPulseConveyor(nil, PulseConveyorConfig{
				MinCachePulseAge:      0,
				MaxPastPulseAge:       0,
			}, handleFactory, nil)
		})

		require.Panics(t, func() {
			NewPulseConveyor(nil, PulseConveyorConfig{
				MinCachePulseAge:      1,
				MaxPastPulseAge:       0,
			}, handleFactory, nil)
		})

		require.Panics(t, func() {
			NewPulseConveyor(nil, PulseConveyorConfig{
				MinCachePulseAge:      1,
				MaxPastPulseAge:       1,
			}, handleFactory, nil)
		})
	})

	t.Run("ok", func(t *testing.T) {
		machineConfig := smachine.SlotMachineConfig{
			PollingPeriod:   500 * time.Millisecond,
			PollingTruncate: 1 * time.Millisecond,
			SlotPageSize:    1000,
			ScanCountLimit:  100000,
		}

		NewPulseConveyor(nil, PulseConveyorConfig{
			ConveyorMachineConfig: machineConfig,
			SlotMachineConfig:     machineConfig,
			MinCachePulseAge:      100,
			MaxPastPulseAge:       1000,
		}, handleFactory, nil)
	})
}

const maxPastPulseAge = 100

func newTestPulseConveyor(ctx context.Context, t *testing.T,
	preFactoryFn func(pulse.Number, pulse.Range), cycleFn PulseConveyorCycleFunc,
) (*PulseConveyor, chan struct{}) {
	instestlogger.SetTestOutput(t)

	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:   500 * time.Millisecond,
		PollingTruncate: 1 * time.Millisecond,
		SlotPageSize:    1000,
		ScanCountLimit:  100000,
	}

	conveyor := NewPulseConveyor(ctx, PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		MinCachePulseAge:      maxPastPulseAge / 2,
		MaxPastPulseAge:       maxPastPulseAge,
	}, func(_ context.Context, input InputEvent, ic InputContext) (InputSetup, error) {
		if preFactoryFn != nil {
			preFactoryFn(ic.PulseNumber, ic.PulseRange)
		}
		burn := false
		if b, ok := input.(bool); ok && b {
			burn = true
		}
		return InputSetup{
			CreateFn: func(ctx smachine.ConstructionContext) smachine.StateMachine {
				return &emptySM{ burner: burn}
			}}, nil
	}, nil)

	emerChan := make(chan struct{})
	conveyor.StartWorkerExt(emerChan, nil, cycleFn)
	return conveyor, emerChan
}

func TestPulseConveyor_AddInput(t *testing.T) {
	t.Run("no pulse yet", func(t *testing.T) {
		defer testutils.LeakTester(t)

		ctx := context.Background()
		pn := pulse.Number(pulse.MinTimePulse + 1)

		conveyor, emerChan := newTestPulseConveyor(ctx, t, func(inputPN pulse.Number, pr pulse.Range) {
			require.NotNil(t, pr)
			require.True(t, pr.RightBoundData().IsExpectedPulse())
			require.Equal(t, pn, inputPN)
		}, nil)

		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.AddInput(ctx, pn, InputEvent(nil)))
	})

	t.Run("1 pulse", func(t *testing.T) {
		ctx := context.Background()
		pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
		startPn := pd.PulseNumber
		pn := startPn + pulse.Number(pd.NextPulseDelta)

		conveyor, emerChan := newTestPulseConveyor(ctx, t, func(inputPN pulse.Number, pr pulse.Range) {
			require.NotNil(t, pr)
			require.True(t, pr.RightBoundData().IsExpectedPulse())
			require.Equal(t, pn, inputPN)
		}, nil)

		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))
		require.NoError(t, conveyor.AddInput(ctx, pn, InputEvent(nil)))
		require.Error(t, conveyor.AddInput(ctx, 1, InputEvent(nil)))
	})

	t.Run("present pulse", func(t *testing.T) {
		ctx := context.Background()
		pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
		startPn := pd.PulseNumber

		conveyor, emerChan := newTestPulseConveyor(ctx, t, func(inputPN pulse.Number, pr pulse.Range) {
			require.NotNil(t, pr)
			require.False(t, pr.RightBoundData().IsExpectedPulse())
			require.Equal(t, startPn, inputPN)
		}, nil)
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))

		require.NoError(t, conveyor.AddInput(ctx, startPn, InputEvent(nil)))
	})

	t.Run("past pulse, that has slots", func(t *testing.T) {
		ctx := context.Background()
		pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
		firstPn := pd.PulseNumber
		nextPd := pd.CreateNextPulse(emptyEntropyFn)
		pn := nextPd.PulseNumber

		conveyor, emerChan := newTestPulseConveyor(ctx, t, func(inputPN pulse.Number, pr pulse.Range) {
			require.NotNil(t, pr)
			require.False(t, pr.RightBoundData().IsExpectedPulse())
		}, nil)
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))

		require.NoError(t, conveyor.AddInput(ctx, firstPn, InputEvent(nil)))

		require.NoError(t, conveyor.PreparePulseChange(nil))

		require.NoError(t, conveyor.CommitPulseChange(nextPd.AsRange(), time.Now(), nil))

		require.NoError(t, conveyor.AddInput(ctx, pn, InputEvent(nil)))
	})

	t.Run("antique pulse, never had pulseData", func(t *testing.T) {
		ctx := context.Background()
		pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
		firstPn := pd.PulseNumber
		nextPd := pd.CreateNextPulse(emptyEntropyFn)

		conveyor, emerChan := newTestPulseConveyor(ctx, t, func(inputPN pulse.Number, pr pulse.Range) {
			require.Nil(t, pr)
			require.Equal(t, firstPn, inputPN)
		}, nil)
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(nextPd.AsRange(), time.Now(), nil))

		require.NoError(t, conveyor.AddInput(ctx, firstPn, InputEvent(nil)))
	})

	t.Run("antique pulse, cached pulseData", func(t *testing.T) {
		ctx := context.Background()
		const delta = 10
		pd := pulse.NewFirstPulsarData(delta, longbits.Bits256{})
		firstPn := pd.PulseNumber

		conveyor, emerChan := newTestPulseConveyor(ctx, t, func(inputPN pulse.Number, pr pulse.Range) {
			require.NotNil(t, pr)
			require.False(t, pr.RightBoundData().IsExpectedPulse())
			require.Equal(t, firstPn, inputPN)
		}, nil)
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))

		// add less than cache limit
		for i := (maxPastPulseAge / (2 * delta)) - 1; i > 0; i-- {
			pd = pd.CreateNextPulse(emptyEntropyFn)
			require.NoError(t, conveyor.PreparePulseChange(nil))
			require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))
		}

		require.NoError(t, conveyor.AddInput(ctx, firstPn, InputEvent(nil)))
	})

	t.Run("antique pulse, evicted pulseData", func(t *testing.T) {
		ctx := context.Background()
		const delta = 10
		pd := pulse.NewFirstPulsarData(delta, longbits.Bits256{})
		firstPn := pd.PulseNumber

		conveyor, emerChan := newTestPulseConveyor(ctx, t, func(inputPN pulse.Number, pr pulse.Range) {
			require.Nil(t, pr)
			require.Equal(t, firstPn, inputPN)
		}, nil)
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))

		// add more than cache limit
		for i := (maxPastPulseAge / (2 * delta)) + 1; i > 0; i-- {
			pd = pd.CreateNextPulse(emptyEntropyFn)
			require.NoError(t, conveyor.PreparePulseChange(nil))
			require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))
		}

		require.Nil(t, conveyor.pdm.getCachedPulseSlot(firstPn))

		require.NoError(t, conveyor.AddInput(ctx, firstPn, InputEvent(nil)))
	})
}

func TestPulseConveyor_Cache(t *testing.T) {
	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:   500 * time.Millisecond,
		PollingTruncate: 1 * time.Millisecond,
		SlotPageSize:    1000,
		ScanCountLimit:  100000,
	}

	ctx := context.Background()

	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})

	firstPn := pd.PulseNumber

	nextPd := pd.CreateNextPulse(emptyEntropyFn)

	conveyor := NewPulseConveyor(ctx, PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        100 * time.Millisecond,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, func(_ context.Context, input InputEvent, ic InputContext) (InputSetup, error) {
		t.FailNow()
		return InputSetup{}, nil
	}, nil)

	emerChan := make(chan struct{})
	conveyor.StartWorker(emerChan, nil)
	defer func() {
		close(emerChan)
		conveyor.Stop()
	}()

	dm := conveyor.GetDataManager()

	// There are no pulses
	prevPN, prevBeat := dm.GetPrevBeatData()
	require.Equal(t, pulse.Unknown, prevPN)
	require.Nil(t, prevBeat.Range)

	require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))

	// There is only one pulse, hence no prev
	prevPN, prevBeat = dm.GetPrevBeatData()
	require.Equal(t, pulse.Unknown, prevPN)
	require.Nil(t, prevBeat.Range)

	require.NoError(t, conveyor.PreparePulseChange(nil))
	require.NoError(t, conveyor.CommitPulseChange(nextPd.AsRange(), time.Now(), nil))

	// There is more than one pulse
	prevPN, prevBeat = dm.GetPrevBeatData()
	require.Equal(t, firstPn, prevPN)
	require.Equal(t, pd.AsRange(), prevBeat.Range)
}

func TestPulseConveyor_PulsePreparing(t *testing.T) {
	ctx := context.Background()
	const delta = 10
	pd := pulse.NewFirstPulsarData(delta, longbits.Bits256{})

	cycleCount := atomickit.NewUint64(0)
	nextCycleSig := make(chan struct{}, 1)
	cycleLock := sync.Mutex{}

	conveyor, emerChan := newTestPulseConveyor(ctx, t, nil,
		func(state CycleState) {
			if state == Scanning {
				return
			}

			cycleLock.Lock()
			defer cycleLock.Unlock()

			select {
			case nextCycleSig <- struct{}{}:
			default:
			}

			switch state {
			case ScanIdle:
				cycleCount.Add(1)
			case ScanActive:
				cycleCount.Add(1<<32)
			}
		})

	defer func() {
		close(emerChan)
		conveyor.Stop()
	}()

	require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))
	require.NoError(t, conveyor.AddInput(ctx, pd.PulseNumber, true))

	pd = pd.CreateNextPulse(emptyEntropyFn)
	require.NoError(t, conveyor.AddInput(ctx, pd.PulseNumber, true))

	require.NoError(t, conveyor.PreparePulseChange(nil))
	require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))

	// make sure AddInput is handled etc
	for i := 10; i > 0; i-- {
		<-nextCycleSig
	}

	pd = pd.CreateNextPulse(emptyEntropyFn)

	require.NoError(t, conveyor.PreparePulseChange(func(PreparedState) {
		cycleLock.Lock()
	}))

	// after this moment there are only priority SMs allowed
	// and cycle is blocked

	// flush mark and reset counters
	select {
	case <-nextCycleSig:
	default:
	}
	require.NotZero(t, cycleCount.Swap(0))

	cycleLock.Unlock()
	<-nextCycleSig

	// here it may have a few active cycles (PreparePulseChange, then stepPreparingChange) and one idle cycle
	for uint32(cycleCount.Load()) == 0 { // wait for idle cycle
		time.Sleep(time.Millisecond)
	}

	expectedCounts := cycleCount.Load()

	require.EqualValues(t, 1, uint32(expectedCounts))
	require.GreaterOrEqual(t, 2, int(expectedCounts>>32))

	time.Sleep(10*time.Millisecond)
	// should be no more new cycles
	require.EqualValues(t, expectedCounts, cycleCount.Load())

	require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))

	// normal activities are restored
	<-nextCycleSig // possible residual
	<-nextCycleSig // extra cycle 1
	<-nextCycleSig // extra cycle 2

	count := cycleCount.Load()

	require.EqualValues(t, uint32(expectedCounts), uint32(count)) // should be NO more idle cycles as SMs are burning
	require.LessOrEqual(t, 2+(expectedCounts>>32), count>>32) // at least +2 as waited for 2 signals
}
