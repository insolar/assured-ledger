// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ConveyorWorker struct {
	conveyor *conveyor.PulseConveyor
	stopped  sync.WaitGroup
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
	conveyor.StartWorker(nil, func() {
		w.stopped.Done()
	})
}

func NewConveyorWorker() ConveyorWorker {
	return ConveyorWorker{}
}

type AsyncTimeMessage struct {
	*log.Msg `txt:"async time"`

	AsyncComponent     string `opt:""`
	AsyncExecutionTime int64
}

func LogAsyncTime(log smachine.Logger, timeBefore time.Time, component string) {
	log.Trace(AsyncTimeMessage{
		AsyncComponent:     component,
		AsyncExecutionTime: time.Since(timeBefore).Nanoseconds(),
	})
}
