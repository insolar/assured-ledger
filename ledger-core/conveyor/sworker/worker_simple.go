// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sworker

import (
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

// Very simple implementation of a slot worker. No support for detachments.
func NewAttachableSimpleSlotWorker() *AttachableSimpleSlotWorker {
	return &AttachableSimpleSlotWorker{}
}

var _ smachine.AttachableSlotWorker = &AttachableSimpleSlotWorker{}

type AttachableSimpleSlotWorker struct {
	exclusive uint32
}

func (v *AttachableSimpleSlotWorker) WakeupWorkerOnEvent() {
}

func (v *AttachableSimpleSlotWorker) WakeupWorkerOnSignal() {
}

func (v *AttachableSimpleSlotWorker) AttachAsNested(m *smachine.SlotMachine, outer smachine.DetachableSlotWorker,
	loopLimit uint32, fn smachine.AttachedFunc) (wasDetached bool) {

	if !atomic.CompareAndSwapUint32(&v.exclusive, 0, 1) {
		panic("is attached")
	}
	defer atomic.StoreUint32(&v.exclusive, 0)

	w := &SimpleSlotWorker{outerSignal: outer.GetSignalMark(), loopLimitFn: outer.CanLoopOrHasSignal,
		machine: m, loopLimit: int(loopLimit)}

	w.init()
	fn(w)
	return false
}

func (v *AttachableSimpleSlotWorker) AttachTo(m *smachine.SlotMachine, signal *synckit.SignalVersion,
	loopLimit uint32, fn smachine.AttachedFunc) (wasDetached bool) {

	if !atomic.CompareAndSwapUint32(&v.exclusive, 0, 1) {
		panic("is attached")
	}
	defer atomic.StoreUint32(&v.exclusive, 0)

	w := &SimpleSlotWorker{outerSignal: signal, machine: m, loopLimit: int(loopLimit)}

	w.init()
	fn(w)
	return false
}

var _ smachine.FixedSlotWorker = &SimpleSlotWorker{}

type SimpleSlotWorker struct {
	outerSignal *synckit.SignalVersion
	loopLimitFn smachine.LoopLimiterFunc // NB! MUST correlate with outerSignal
	loopLimit   int

	machine *smachine.SlotMachine

	dsw DetachableSimpleSlotWorker
	nsw NonDetachableSimpleSlotWorker
}

func (p *SimpleSlotWorker) init() {
	p.dsw.SimpleSlotWorker = p
	p.nsw.SimpleSlotWorker = p
}

func (p *SimpleSlotWorker) HasSignal() bool {
	return p.outerSignal != nil && p.outerSignal.HasSignal()
}

func (*SimpleSlotWorker) IsDetached() bool {
	return false
}

func (p *SimpleSlotWorker) GetSignalMark() *synckit.SignalVersion {
	return p.outerSignal
}

func (p *SimpleSlotWorker) CanLoopOrHasSignal(loopCount int) (canLoop, hasSignal bool) {
	switch {
	case p.loopLimitFn != nil:
		canLoop, hasSignal = p.loopLimitFn(loopCount)
		if loopCount >= p.loopLimit {
			canLoop = false
		}
		return canLoop, hasSignal

	case p.outerSignal.HasSignal():
		return false, true
	default:
		return loopCount < p.loopLimit, false
	}
}

func (p *SimpleSlotWorker) OuterCall(*smachine.SlotMachine, smachine.NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (p *SimpleSlotWorker) DetachableCall(fn smachine.DetachableFunc) (wasDetached bool) {
	fn(&p.dsw)
	return false
}

var _ smachine.DetachableSlotWorker = &DetachableSimpleSlotWorker{}

type DetachableSimpleSlotWorker struct {
	*SimpleSlotWorker
}

func (p *DetachableSimpleSlotWorker) TryDetach(smachine.LongRunFlags) {
	panic("unsupported")
}

func (p *DetachableSimpleSlotWorker) NonDetachableOuterCall(*smachine.SlotMachine, smachine.NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (p *DetachableSimpleSlotWorker) DetachableOuterCall(*smachine.SlotMachine, smachine.DetachableFunc) (wasExecuted, wasDetached bool) {
	return false, false
}

func (p *DetachableSimpleSlotWorker) NonDetachableCall(fn smachine.NonDetachableFunc) (wasExecuted bool) {
	fn(&NonDetachableSimpleSlotWorker{p.SimpleSlotWorker})
	return true
}

type NonDetachableSimpleSlotWorker struct {
	*SimpleSlotWorker
}

func (p *NonDetachableSimpleSlotWorker) DetachableCall(fn smachine.DetachableFunc) (wasDetached bool) {
	panic("not allowed") // this method shouldn't be accessible through interface
}
