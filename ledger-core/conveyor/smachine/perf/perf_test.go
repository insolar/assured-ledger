package perf

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func BenchmarkLoops(b *testing.B) {
	const workCost = 0 //
	b.Run("Full Loops", func(b *testing.B) {
		const shortLoops = false
		runBench(b, 1, 0, workCost, shortLoops, false)
		runBench(b, 10, 0, workCost, shortLoops, false)
		runBench(b, 1000, 0, workCost, shortLoops, false)
		runBench(b, 100_000, 0, workCost, shortLoops, false)
	})

	b.Run("Full Loops vs Idle", func(b *testing.B) {
		const shortLoops = false
		runBench(b, 1000, 0, workCost, shortLoops, false)
		runBench(b, 1000, 10, workCost, shortLoops, false)
		runBench(b, 1000, 1000, workCost, shortLoops, false)
		runBench(b, 1000, 100_000, workCost, shortLoops, false)
//		runBench(b, 1000, 10_000_000, workCost, shortLoops, false)
	})

	b.Run("Short Loops", func(b *testing.B) {
		const shortLoops = true
		runBench(b, 1, 0, workCost, shortLoops, false)
		runBench(b, 10, 0, workCost, shortLoops, false)
		runBench(b, 1000, 0, workCost, shortLoops, false)
		runBench(b, 100_000, 0, workCost, shortLoops, false)
	})
}

func runBench(b *testing.B, activeCount, idleCount, workLoop int, shortLoop, boost bool, ) {
	name := ""
	if idleCount == 0 {
		name = fmt.Sprintf("%6d SM", activeCount)
	} else {
		name = fmt.Sprintf("%d of %d SM", activeCount, activeCount + idleCount)
	}
	const scanCountLimit = math.MaxUint32

	b.Run(name, func(b *testing.B) {
		workLoopsTotal := atomickit.Uint64{}
		b.ReportAllocs()
		runBenchSlotMachine(scanCountLimit, b.N, boost, func(m *smachine.SlotMachine) {
			for i := idleCount; i > 0; i-- {
				sm := &idleSM{ counter: &workLoopsTotal }
				m.AddNew(context.Background(), sm, smachine.CreateDefaultValues{})
			}
			for i := activeCount; i > 0; i-- {
				sm := &benchSM{ shortLoop: shortLoop, workLoop: workLoop, counter: &workLoopsTotal }
				m.AddNew(context.Background(), sm, smachine.CreateDefaultValues{})
			}
			b.ResetTimer()
		})
		b.SetBytes(int64(workLoopsTotal.Load() / uint64(b.N)))
	})
}

func runBenchSlotMachine(scanCountLimit int, cycleCount int, boost bool, initFn func(*smachine.SlotMachine)) {
	boostDuration := 10 * time.Millisecond
	if !boost {
		boostDuration = 0
	}

	signal := synckit.NewVersionedSignal()
	m := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:         1000,
		PollingPeriod:        10 * time.Millisecond,
		PollingTruncate:      1 * time.Microsecond,
		BoostNewSlotDuration: boostDuration,
		ScanCountLimit:       scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()

	initFn(m)

	for ;cycleCount > 0;cycleCount-- {
		var (
			repeatNow    bool
			nextPollTime time.Time
		)
		wakeupSignal := signal.Mark()
		workerFactory.AttachTo(m, neverSignal, uint32(scanCountLimit), func(worker smachine.AttachedSlotWorker) {
			repeatNow, nextPollTime = m.ScanOnce(0, worker)
		})

		if repeatNow {
			continue
		}
		if !nextPollTime.IsZero() {
			time.Sleep(time.Until(nextPollTime))
		} else {
			wakeupSignal.Wait()
		}
	}
}

type benchSM struct {
	smachine.StateMachineDeclTemplate
	counter   *atomickit.Uint64
	shortLoop bool
	workLoop  int
}

func (p *benchSM) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *benchSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *benchSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	p.counter.Add(1)
	if p.shortLoop {
		return ctx.Jump(p.stepWorkShortLoops)
	}
	return ctx.Jump(p.stepWork)
}

func (p *benchSM) stepWork(ctx smachine.ExecutionContext) smachine.StateUpdate {
	for i := p.workLoop; i > 0; i-- {
		p.doWork()
	}
	p.counter.Add(1)
	return ctx.Yield().ThenRepeat()
}

func (p *benchSM) stepWorkShortLoops(ctx smachine.ExecutionContext) smachine.StateUpdate {
	for i := p.workLoop; i > 0; i-- {
		p.doWork()
	}
	p.counter.Add(1)
	return ctx.Repeat(math.MaxUint32)
}

func (p *benchSM) doWork() {
	if time.Now().Unix() < 100000 {
		panic(throw.Impossible())
	}
}

type idleSM struct {
	smachine.StateMachineDeclTemplate
	counter   *atomickit.Uint64
}

func (p *idleSM) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *idleSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *idleSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	p.counter.Add(1)
	return ctx.Jump(p.stepIdle)
}

func (p *idleSM) stepIdle(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
