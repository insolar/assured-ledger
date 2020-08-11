// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func TestAntique_InheritPulseSlot(t *testing.T) {
	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:   500 * time.Millisecond,
		PollingTruncate: 1 * time.Millisecond,
		SlotPageSize:    1000,
		ScanCountLimit:  100000,
	}

	ctx := context.Background()
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})

	firstPn := pd.PulseNumber

	doneCounter := &atomickit.Uint32{}
	semaCounter := smsync.NewFixedSemaphore(100, "SM counter")

	const checkDepth = 5
	conveyor := NewPulseConveyor(ctx, PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, func(_ context.Context, input InputEvent, ic InputContext) (InputSetup, error) {
		require.Equal(t, "inputEvent", input)
		return InputSetup{
			CreateFn: func(ctx smachine.ConstructionContext) smachine.StateMachine {
				return &testAntiqueSM{counter: checkDepth, doneCounter: doneCounter, semaCounter: semaCounter}
			}}, nil
	}, nil)

	emerChan := make(chan struct{})
	conveyor.StartWorker(emerChan, func() {})
	defer func() {
		close(emerChan)
		conveyor.Stop()
	}()

	require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))

	for i := 5; i > 0; i-- {
		pd = pd.CreateNextPulse(emptyEntropyFn)
		require.NoError(t, conveyor.PreparePulseChange(nil))
		require.NoError(t, conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	require.NoError(t, conveyor.AddInputExt(firstPn, "inputEvent", smachine.CreateDefaultValues{
		Context: ctx,
		TerminationHandler: func(data smachine.TerminationData) {
			wg.Done()
		},
	}))

	wg.Wait()

	for {
		if active, _ := semaCounter.GetCounts(); active == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.Equal(t, 1+checkDepth, int(doneCounter.Load()))
}

type testAntiqueSM struct {
	smachine.StateMachineDeclTemplate
	pulseSlot   *PulseSlot
	semaCounter smachine.SyncLink
	doneCounter *atomickit.Uint32
	counter     int
}

func (p *testAntiqueSM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	if p.counter <= 0 {
		_ = injector.Inject(&p.pulseSlot)
	}
}

func (p *testAntiqueSM) GetInitStateFor(machine smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *testAntiqueSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *testAntiqueSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.Acquire(p.semaCounter)

	if p.counter <= 0 {
		switch {
		case p.pulseSlot == nil:
			panic(throw.IllegalState())
		case p.pulseSlot.State() != Antique:
			panic(throw.IllegalState())
		}
		p.doneCounter.Add(1)
		return ctx.Stop()
	}

	p.doneCounter.Add(1)
	p.counter--
	return ctx.Jump(p.stepGoDown)
}

func (p *testAntiqueSM) stepGoDown(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.NewChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		ctx.SetDependencyInheritanceMode(smachine.InheritAllDependencies)
		return &testAntiqueSM{counter: p.counter, semaCounter: p.semaCounter, doneCounter: p.doneCounter}
	})

	return ctx.Yield().ThenJump(p.stepStop) // this makes sure that the child's init happens before stop
}

func (p *testAntiqueSM) stepStop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}
