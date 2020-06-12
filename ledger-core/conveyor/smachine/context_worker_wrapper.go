// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ DetachableSlotWorker = fixedWorkerWrapper{}

type fixedWorkerWrapper struct {
	FixedSlotWorker
}

func (w fixedWorkerWrapper) AddNestedCallCount(u uint) {
	panic(throw.Unsupported())
}

func (w fixedWorkerWrapper) NonDetachableCall(fn NonDetachableFunc) (wasExecuted bool) {
	fn(w)
	return true
}

func (w fixedWorkerWrapper) NonDetachableOuterCall(sm *SlotMachine, fn NonDetachableFunc) (wasExecuted bool) {
	return w.OuterCall(sm, fn)
}

func (w fixedWorkerWrapper) DetachableOuterCall(*SlotMachine, DetachableFunc) (wasExecuted, wasDetached bool) {
	return false, false
}

func (w fixedWorkerWrapper) TryDetach(LongRunFlags) {
	panic(throw.Unsupported())
}
