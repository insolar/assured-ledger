// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewAttachedSlotWorker(worker SlotWorkerSupport) AttachedSlotWorker {
	if worker == nil {
		panic(throw.IllegalState())
	}
	return AttachedSlotWorker{internalSlotWorker{worker, false}}
}

func NewDetachableSlotWorker(worker SlotWorkerSupport) DetachableSlotWorker {
	if worker == nil {
		panic(throw.IllegalState())
	}
	return DetachableSlotWorker{internalSlotWorker{worker, false}}
}

func NewFixedSlotWorker(worker SlotWorkerSupport) FixedSlotWorker {
	if worker == nil {
		panic(throw.IllegalState())
	}
	return FixedSlotWorker{ internalSlotWorker{ worker, true}}
}

var _ SlotWorker = internalSlotWorker{}
type internalSlotWorker struct {
	worker  SlotWorkerSupport
	isFixed bool
}

func (v internalSlotWorker) HasSignal() bool {
	return v.worker.HasSignal()
}

func (v internalSlotWorker) IsDetached() bool {
	return v.worker.IsDetached()
}

func (v internalSlotWorker) GetSignalMark() *synckit.SignalVersion {
	return v.worker.GetSignalMark()
}

func (v internalSlotWorker) CanLoopOrHasSignal(loopCount int) (canLoop, hasSignal bool) {
	return v.worker.CanLoopOrHasSignal(loopCount)
}

func (v internalSlotWorker) IsZero() bool {
	return v.worker == nil
}

var _ SlotWorker = DetachableSlotWorker{}

type DetachableSlotWorker struct {
	internalSlotWorker
}

func (v DetachableSlotWorker) AddNestedCallCount(u uint) {
	v.worker.AddNestedCallCount(u)
}

// NonDetachableCall provides a temporary protection from detach
func (v DetachableSlotWorker) NonDetachableCall(fn NonDetachableFunc) (wasExecuted bool) {
	if v.isFixed {
		fn(FixedSlotWorker(v))
		return true
	}

	if !v.worker.TryStartNonDetachableCall() {
		return false
	}
	defer v.worker.EndNonDetachableCall()
	fn(FixedSlotWorker{internalSlotWorker{
		worker:  v.worker,
		isFixed: true,
	}})
	return true
}

// NonDetachableOuterCall checks if this worker can serve another SlotMachine
// and if so provides a temporary protection from detach
func (v DetachableSlotWorker) NonDetachableOuterCall(*SlotMachine, NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (v DetachableSlotWorker) DetachableOuterCall(*SlotMachine, DetachableFunc) (wasExecuted, wasDetached bool) {
	return false, false
}

func (v DetachableSlotWorker) TryDetach(flags LongRunFlags) {
	v.worker.TryDetach(flags)
}

var _ SlotWorker = FixedSlotWorker{}
type FixedSlotWorker struct {
	internalSlotWorker
}

func (v FixedSlotWorker) OuterCall(*SlotMachine, NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (v FixedSlotWorker) asDetachable() DetachableSlotWorker {
	switch {
	case v.IsZero():
		panic(throw.IllegalState())
	case !v.isFixed:
		panic(throw.IllegalState())
	}
	return DetachableSlotWorker(v)
}

type AttachedSlotWorker struct {
	internalSlotWorker
}

func (v AttachedSlotWorker)	OuterCall(*SlotMachine, NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (v AttachedSlotWorker)	CanWorkOn(*SlotMachine) bool {
	panic(throw.NotImplemented())
}

func (v AttachedSlotWorker) DetachableCall(fn DetachableFunc) (wasDetached bool) {
	if v.isFixed {
		panic(throw.IllegalState())
	}

	if !v.worker.TryStartDetachableCall() {
		panic(throw.IllegalState())
	}
	defer func() {
		wasDetached = v.worker.EndDetachableCall()
	}()

	fn(DetachableSlotWorker{internalSlotWorker{v.worker, false}})
	return
}

func (v AttachedSlotWorker) AsFixedSlotWorker() FixedSlotWorker {
	return FixedSlotWorker{ internalSlotWorker{ v.worker, true}}
}

