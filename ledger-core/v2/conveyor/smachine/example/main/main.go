/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package main

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/example"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

func main() {
	const scanCountLimit = 1e4

	signal := synckit.NewVersionedSignal()
	sm := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:         100,
		PollingPeriod:        10 * time.Millisecond,
		PollingTruncate:      1 * time.Microsecond,
		BoostNewSlotDuration: 10 * time.Millisecond,
		ScanCountLimit:       scanCountLimit,
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	for i := 0; i < 1e4; i++ {
		sm.AddNew(context.Background(), &example.PlayerSM{}, smachine.CreateDefaultValues{})
	}

	//for i := 0; i < 1; i++ {
	//	sm.AddNew(context.Background(), &example.StateMachine1{}, smachine.CreateDefaultValues{})
	//}

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()

	for {
		var (
			repeatNow    bool
			nextPollTime time.Time
		)
		wakeupSignal := signal.Mark()
		workerFactory.AttachTo(sm, neverSignal, scanCountLimit, func(worker smachine.AttachedSlotWorker) {
			repeatNow, nextPollTime = sm.ScanOnce(0, worker)
		})
		switch {
		case repeatNow:
			continue
		case !nextPollTime.IsZero():
			time.Sleep(time.Until(nextPollTime))
		default:
			wakeupSignal.Wait()
			//runtime.KeepAlive(wakeupSignal)
			//time.Sleep(3 * time.Second)
		}
	}
}
