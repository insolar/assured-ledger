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

	/*** Initialize SlotMachine ***/
	signal := synckit.NewVersionedSignal()
	sm := smachine.NewSlotMachine(smachine.SlotMachineConfig{
		SlotPageSize:         100,
		PollingPeriod:        100 * time.Millisecond,
		PollingTruncate:      1 * time.Microsecond,
		BoostNewSlotDuration: 10 * time.Millisecond,
		ScanCountLimit:       scanCountLimit,
		LogAdapterCalls:      true,
		SlotMachineLogger:    example.MachineLogger{},
	}, signal.NextBroadcast, signal.NextBroadcast, nil)

	/*** Add injectables ***/

	sm.AddDependency(example.NewGameAdapter(context.Background(), example.NewGameChooseService()))

	/*** Add SMs ***/

	for i := 0; i < 2; i++ {
		sm.AddNew(context.Background(), &example.PlayerSM{}, smachine.CreateDefaultValues{})
	}

	/*** Run the SlotMachine (ans all SMs) ***/

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()

	defer func() {
		sm.OccupiedSlotCount()
	}()

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
		case !sm.IsActive():
			return
		case sm.OccupiedSlotCount() < 2:
			return
		default:
			wakeupSignal.Wait()
			//runtime.KeepAlive(wakeupSignal)
			//time.Sleep(3 * time.Second)
		}
	}
}
