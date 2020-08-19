// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
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

type SlotWorkerSupport interface {
	SlotWorker
	AddNestedCallCount(uint)
	TryDetach(LongRunFlags)

	TryStartDetachableCall() bool
	EndDetachableCall() (wasDetached bool)

	TryStartNonDetachableCall() bool
	EndNonDetachableCall()
}

type AttachableSlotWorker interface {
	AttachTo(m *SlotMachine, signal *synckit.SignalVersion, loopLimit uint32, fn AttachedFunc) (wasDetached bool, callCount uint)
	AttachAsNested(m *SlotMachine, w DetachableSlotWorker, loopLimit uint32, fn AttachedFunc) (wasDetached bool)
}
