package main

import (
	"context"
	"fmt"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/main/example"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

func main() {
	const scanCountLimit = 1e6

	signal := synckit.NewVersionedSignal()
	sm := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:         1000,
		PollingPeriod:        10 * time.Millisecond,
		PollingTruncate:      1 * time.Microsecond,
		BoostNewSlotDuration: 10 * time.Millisecond,
		ScanCountLimit:       scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	sm.PutDependency("example.ServiceAdapterA", example.CreateServiceAdapterA())
	sm.PutDependency("example.catalogC", example.CreateCatalogC())

	// for i := 0; i < 1e4; i++ {
	//	sm.AddNew(context.Background(), &example.StateMachine2{Yield: false}, smachine.CreateDefaultValues{})
	// }

	for i := 0; i < 1; i++ {
		sm.AddNew(context.Background(), &example.StateMachine1{}, smachine.CreateDefaultValues{})
	}

	workerFactory := sworker.NewAttachableSimpleSlotWorker()

	startNano := time.Now().UnixNano()
	startBase := example.IterationCount

	iterBase := example.IterationCount
	iterStart := time.Now().UnixNano()

	neverSignal := synckit.NewNeverSignal()

	prev := 0
	for i := 0; ; i++ {
		var (
			repeatNow    bool
			nextPollTime time.Time
		)
		wakeupSignal := signal.Mark()
		workerFactory.AttachTo(sm, neverSignal, scanCountLimit, func(worker smachine.AttachedSlotWorker) {
			repeatNow, nextPollTime = sm.ScanOnce(0, worker)
		})

		//fmt.Printf("%03d %v ================================== slots=%v of %v\n", i, time.Now(), sm.OccupiedSlotCount(), sm.AllocatedSlotCount())
		//fmt.Printf("%03d %v =============== repeatNow=%v nextPollTime=%v\n", i, time.Now(), repeatNow, nextPollTime)

		if i >= prev+1e8/scanCountLimit {
			prev = i
			fmt.Printf("%03d ================================== iters=%d speedIter=%7.0f speedTotal=%7.0f\n", i, example.IterationCount,
				float64(example.IterationCount-iterBase)/(float64(time.Now().UnixNano()-iterStart)/float64(time.Second)),
				float64(example.IterationCount-startBase)/(float64(time.Now().UnixNano()-startNano)/float64(time.Second)),
			)
			fmt.Printf("%03d %v ================================== slots=%v of %v\n", i, time.Now(), sm.OccupiedSlotCount(), sm.AllocatedSlotCount())
			active, inactive := example.Limiter.GetCounts()
			fmt.Println("Limiter: ", active, inactive)

			iterBase = example.IterationCount
			iterStart = time.Now().UnixNano()
			if example.IterationCount < 1000000 {
				startNano = time.Now().UnixNano()
				startBase = example.IterationCount
			}
			//sm.Cleanup(worker)
			//fmt.Printf("%03d %v ================================== slots=%v of %v\n", i, time.Now(), sm.OccupiedSlotCount(), sm.AllocatedSlotCount())
		}

		if repeatNow {
			continue
		}
		if !nextPollTime.IsZero() {
			time.Sleep(time.Until(nextPollTime))
		} else {
			wakeupSignal.Wait()
			//runtime.KeepAlive(wakeupSignal)
			//time.Sleep(3 * time.Second)
		}
	}
}
