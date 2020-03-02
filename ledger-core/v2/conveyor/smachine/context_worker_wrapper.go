// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

var _ DetachableSlotWorker = fixedWorkerWrapper{}

type fixedWorkerWrapper struct {
	FixedSlotWorker
}

func (w fixedWorkerWrapper) NonDetachableCall(fn NonDetachableFunc) (wasExecuted bool) {
	fn(w)
	return true
}

func (w fixedWorkerWrapper) NonDetachableOuterCall(*SlotMachine, NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (w fixedWorkerWrapper) TryDetach(flags LongRunFlags) {
	panic("unsupported")
}
