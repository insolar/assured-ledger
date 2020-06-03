// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
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
}

func (sm *emptySM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *emptySM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	injector.MustInject(&sm.pulseSlot)
}

func (sm *emptySM) GetInitStateFor(machine smachine.StateMachine) smachine.InitFunc {
	if sm != machine {
		panic("illegal value")
	}
	return sm.stepInit
}

func (sm *emptySM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func handleFactory(_ pulse.Number, _ pulse.Range, input InputEvent) (pulse.Number, smachine.CreateFunc) {
	switch input.(type) {
	default:
		panic(fmt.Sprintf("unknown event type, got %T", input))
	}
}

func TestNewPulseConveyor(t *testing.T) {
	t.Run("bad input", func(t *testing.T) {
		require.Panics(t, func() {
			NewPulseConveyor(nil, PulseConveyorConfig{
				ConveyorMachineConfig: smachine.SlotMachineConfig{},
				SlotMachineConfig:     smachine.SlotMachineConfig{},
				EventlessSleep:        0,
				MinCachePulseAge:      0,
				MaxPastPulseAge:       0,
			}, handleFactory, nil)
		})

		require.Panics(t, func() {
			NewPulseConveyor(nil, PulseConveyorConfig{
				ConveyorMachineConfig: smachine.SlotMachineConfig{},
				SlotMachineConfig:     smachine.SlotMachineConfig{},
				EventlessSleep:        0,
				MinCachePulseAge:      1,
				MaxPastPulseAge:       0,
			}, handleFactory, nil)
		})

		require.Panics(t, func() {
			NewPulseConveyor(nil, PulseConveyorConfig{
				ConveyorMachineConfig: smachine.SlotMachineConfig{},
				SlotMachineConfig:     smachine.SlotMachineConfig{},
				EventlessSleep:        0,
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
			EventlessSleep:        100 * time.Millisecond,
			MinCachePulseAge:      100,
			MaxPastPulseAge:       1000,
		}, handleFactory, nil)
	})
}

func TestPulseConveyor_AddInput(t *testing.T) {
	t.Run("no pulse yet", func(t *testing.T) {
		defer testutils.LeakTester(t)

		machineConfig := smachine.SlotMachineConfig{
			PollingPeriod:   500 * time.Millisecond,
			PollingTruncate: 1 * time.Millisecond,
			SlotPageSize:    1000,
			ScanCountLimit:  100000,
		}

		ctx := context.Background()

		pn := pulse.Number(pulse.MinTimePulse + 1)

		conveyor := NewPulseConveyor(ctx, PulseConveyorConfig{
			ConveyorMachineConfig: machineConfig,
			SlotMachineConfig:     machineConfig,
			EventlessSleep:        100 * time.Millisecond,
			MinCachePulseAge:      100,
			MaxPastPulseAge:       1000,
		}, func(inputPn pulse.Number, _ pulse.Range, input InputEvent) (pulse.Number, smachine.CreateFunc) {
			require.Equal(t, pn, inputPn)
			require.Nil(t, input)
			return 0, func(ctx smachine.ConstructionContext) smachine.StateMachine {
				return &emptySM{}
			}
		}, nil)

		emerChan := make(chan struct{})
		conveyor.StartWorker(emerChan, func() {})
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.AddInput(ctx, pn, InputEvent(nil)))
	})

	t.Run("1 pulse", func(t *testing.T) {
		machineConfig := smachine.SlotMachineConfig{
			PollingPeriod:   500 * time.Millisecond,
			PollingTruncate: 1 * time.Millisecond,
			SlotPageSize:    1000,
			ScanCountLimit:  100000,
		}

		ctx := context.Background()

		pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
		startPn := pd.PulseNumber
		pn := startPn + pulse.Number(pd.NextPulseDelta)

		conveyor := NewPulseConveyor(ctx, PulseConveyorConfig{
			ConveyorMachineConfig: machineConfig,
			SlotMachineConfig:     machineConfig,
			EventlessSleep:        100 * time.Millisecond,
			MinCachePulseAge:      100,
			MaxPastPulseAge:       1000,
		}, func(inputPn pulse.Number, _ pulse.Range, input InputEvent) (pulse.Number, smachine.CreateFunc) {
			require.Equal(t, pn, inputPn)
			require.Nil(t, input)
			return 0, func(ctx smachine.ConstructionContext) smachine.StateMachine {
				return &emptySM{}
			}
		}, nil)

		emerChan := make(chan struct{})
		conveyor.StartWorker(emerChan, func() {})
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now()))

		require.NoError(t, conveyor.AddInput(ctx, pn, InputEvent(nil)))

		require.Error(t, conveyor.AddInput(ctx, 1, InputEvent(nil)))
	})

	t.Run("present pulse", func(t *testing.T) {
		machineConfig := smachine.SlotMachineConfig{
			PollingPeriod:   500 * time.Millisecond,
			PollingTruncate: 1 * time.Millisecond,
			SlotPageSize:    1000,
			ScanCountLimit:  100000,
		}

		ctx := context.Background()

		pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
		startPn := pd.PulseNumber

		conveyor := NewPulseConveyor(ctx, PulseConveyorConfig{
			ConveyorMachineConfig: machineConfig,
			SlotMachineConfig:     machineConfig,
			EventlessSleep:        100 * time.Millisecond,
			MinCachePulseAge:      100,
			MaxPastPulseAge:       1000,
		}, func(inputPn pulse.Number, _ pulse.Range, input InputEvent) (pulse.Number, smachine.CreateFunc) {
			require.Equal(t, startPn, inputPn)
			require.Nil(t, input)
			return 0, func(ctx smachine.ConstructionContext) smachine.StateMachine {
				return &emptySM{}
			}
		}, nil)

		emerChan := make(chan struct{})
		conveyor.StartWorker(emerChan, func() {})
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now()))

		require.NoError(t, conveyor.AddInput(ctx, startPn, InputEvent(nil)))
	})

	t.Run("past pulse, that has slots", func(t *testing.T) {
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

		pn := nextPd.PulseNumber

		conveyor := NewPulseConveyor(ctx, PulseConveyorConfig{
			ConveyorMachineConfig: machineConfig,
			SlotMachineConfig:     machineConfig,
			EventlessSleep:        100 * time.Millisecond,
			MinCachePulseAge:      100,
			MaxPastPulseAge:       1000,
		}, func(_ pulse.Number, _ pulse.Range, input InputEvent) (pulse.Number, smachine.CreateFunc) {
			require.Nil(t, input)
			return 0, func(ctx smachine.ConstructionContext) smachine.StateMachine {
				return &emptySM{}
			}
		}, nil)

		emerChan := make(chan struct{})
		conveyor.StartWorker(emerChan, func() {})
		defer func() {
			close(emerChan)
			conveyor.Stop()
		}()

		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now()))

		require.NoError(t, conveyor.AddInput(ctx, firstPn, InputEvent(nil)))

		require.NoError(t, conveyor.PreparePulseChange(nil))

		require.NoError(t, conveyor.CommitPulseChange(nextPd.AsRange(), time.Now()))

		require.NoError(t, conveyor.AddInput(ctx, pn, InputEvent(nil)))
	})
}
