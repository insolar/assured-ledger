package slotdebugger

import (
	"math"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

type worker struct {
	slotMachine    *StepController
	stepController chan struct{}
	finished       chan struct{}
	started        bool
}

func newWorker(parent *StepController) *worker {
	return &worker{
		slotMachine:    parent,
		stepController: make(chan struct{}),
		finished:       make(chan struct{}),
	}
}

func (w *worker) finishedSignal() synckit.SignalChannel {
	return w.finished
}

func (w *worker) Start() {
	if w.started {
		return
	}
	w.started = true
	var (
		workerFactory = sworker.NewAttachableSimpleSlotWorker()
	)

	go func() {
		defer close(w.finished)

		machine := w.slotMachine.SlotMachine
		for {
			var (
				repeatNow    bool
				nextPollTime time.Time
			)
			eventMark := w.slotMachine.internalSignal.Mark()
			workerFactory.AttachTo(machine, w.slotMachine.externalSignal.Mark(), math.MaxUint32, func(worker smachine.AttachedSlotWorker) {
				repeatNow, nextPollTime = machine.ScanOnce(0, worker)
			})

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
