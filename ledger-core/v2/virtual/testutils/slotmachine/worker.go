// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotmachine

import (
	"math"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker"
)

type Worker struct {
	slotMachine    *ControlledSlotMachine
	stepController chan struct{}
}

func NewWorker(parent *ControlledSlotMachine) *Worker {
	return &Worker{
		slotMachine:    parent,
		stepController: make(chan struct{}),
	}
}

func (w *Worker) Start() {
	var (
		workerFactory = sworker.NewAttachableSimpleSlotWorker()
	)

	go func() {
		machine := w.slotMachine.slotMachine
		for {
			var (
				repeatNow    bool
				nextPollTime time.Time
			)
			eventMark := w.slotMachine.internalSignal.Mark()
			fn := func(worker smachine.AttachedSlotWorker) {
				repeatNow, nextPollTime = machine.ScanOnce(0, worker)
			}
			workerFactory.AttachTo(machine, w.slotMachine.externalSignal.Mark(), math.MaxUint32, fn)

			if !machine.IsActive() {
				break
			}

			if repeatNow || eventMark.HasSignal() {
				continue
			}

			select {
			case <-eventMark.Channel():
			case <-func() <-chan time.Time {
				if !nextPollTime.IsZero() {
					return time.After(time.Until(nextPollTime))
				}
				return nil
			}():
			}
		}
	}()
}
