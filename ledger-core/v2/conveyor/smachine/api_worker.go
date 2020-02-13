// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

type AttachedFunc func(AttachedSlotWorker)
type DetachableFunc func(DetachableSlotWorker)
type NonDetachableFunc func(FixedSlotWorker)

type SlotWorker interface {
	HasSignal() bool
	IsDetached() bool
	GetSignalMark() *synckit.SignalVersion
	CanLoopOrHasSignal(loopCount int) (canLoop, hasSignal bool)
}

type LoopLimiterFunc func(loopCount int) (canLoop, hasSignal bool)
type DetachableSlotWorker interface {
	SlotWorker

	// provides a temporary protection from detach
	NonDetachableCall(NonDetachableFunc) (wasExecuted bool)
	NonDetachableOuterCall(*SlotMachine, NonDetachableFunc) (wasExecuted bool)

	TryDetach(flags LongRunFlags)
	//NestedAttachTo(m *SlotMachine, loopLimit uint32, fn AttachedFunc) (wasDetached bool)
}

type FixedSlotWorker interface {
	SlotWorker
	OuterCall(*SlotMachine, NonDetachableFunc) (wasExecuted bool)
	//CanWorkOn(*SlotMachine) bool
}

type AttachedSlotWorker interface {
	FixedSlotWorker
	DetachableCall(DetachableFunc) (wasDetached bool)
}

type AttachableSlotWorker interface {
	AttachTo(m *SlotMachine, signal *synckit.SignalVersion, loopLimit uint32, fn AttachedFunc) (wasDetached bool)
	AttachAsNested(m *SlotMachine, w DetachableSlotWorker, loopLimit uint32, fn AttachedFunc) (wasDetached bool)
}
