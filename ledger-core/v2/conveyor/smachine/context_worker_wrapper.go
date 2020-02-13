// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

var _ DetachableSlotWorker = migrateWorkerWrapper{}

type migrateWorkerWrapper struct {
	FixedSlotWorker
}

func (w migrateWorkerWrapper) NonDetachableCall(fn NonDetachableFunc) (wasExecuted bool) {
	fn(w)
	return true
}

func (w migrateWorkerWrapper) NonDetachableOuterCall(*SlotMachine, NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (w migrateWorkerWrapper) TryDetach(flags LongRunFlags) {
	panic("unsupported")
}
