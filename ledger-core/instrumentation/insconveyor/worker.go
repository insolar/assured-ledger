package insconveyor

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ConveyorWorker struct {
	conveyor *conveyor.PulseConveyor
	stopped  sync.WaitGroup
	cycleFn  conveyor.PulseConveyorCycleFunc
}

func (w *ConveyorWorker) Stop() {
	w.conveyor.StopNoWait()
	w.stopped.Wait()
}

func (w *ConveyorWorker) AttachTo(conveyor *conveyor.PulseConveyor) {
	if conveyor == nil {
		panic(throw.IllegalValue())
	}
	if w.conveyor != nil {
		panic(throw.IllegalState())
	}
	w.conveyor = conveyor
	w.stopped.Add(1)
	conveyor.StartWorkerExt(nil, func() {
		w.stopped.Done()
	}, w.cycleFn)
}

func NewConveyorWorker(cycleFn  conveyor.PulseConveyorCycleFunc) ConveyorWorker {
	return ConveyorWorker{cycleFn: cycleFn}
}
