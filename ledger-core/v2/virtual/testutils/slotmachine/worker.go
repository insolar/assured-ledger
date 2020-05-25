// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotmachine

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

type Worker struct {
	slotMachine    *smachine.SlotMachine
	signal         *synckit.VersionedSignal
	stepController chan struct{}
}

func NewWorker(parent *ControlledSlotMachine) *Worker {
	return &Worker{
		slotMachine:    parent.slotMachine,
		signal:         parent.signal,
		stepController: make(chan struct{}),
	}
}

func (w *Worker) Start() {
	var (
		scanCountLimit uint32 = 1e6

		neverSignal   = synckit.NewNeverSignal()
		workerFactory = sworker.NewAttachableSimpleSlotWorker()
	)

	go func() {
		for {
			var (
				repeatNow    bool
				nextPollTime time.Time
			)
			wakeupSignal := w.signal.Mark()
			fn := func(worker smachine.AttachedSlotWorker) {
				repeatNow, nextPollTime = w.slotMachine.ScanOnce(0, worker)
			}
			workerFactory.AttachTo(w.slotMachine, neverSignal, scanCountLimit, fn)
			switch {
			case repeatNow:
				continue
			case !nextPollTime.IsZero():
				time.Sleep(time.Until(nextPollTime))
			case !w.slotMachine.IsActive():
				break
			default:
				wakeupSignal.Wait()
			}
		}
	}()
}
