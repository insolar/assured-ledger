// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sworker

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// Very simple implementation of a slot worker. No support for detachments.
func NewAttachableSimpleSlotWorker() *AttachableSimpleSlotWorker {
	return &AttachableSimpleSlotWorker{}
}

var _ smachine.AttachableSlotWorker = &AttachableSimpleSlotWorker{}

type AttachableSimpleSlotWorker struct {
	exclusive atomickit.Uint32
}

func (v *AttachableSimpleSlotWorker) WakeupWorkerOnEvent() {}

func (v *AttachableSimpleSlotWorker) WakeupWorkerOnSignal() {}

func (v *AttachableSimpleSlotWorker) AttachAsNested(m *smachine.SlotMachine, outer smachine.DetachableSlotWorker,
	loopLimit uint32, fn smachine.AttachedFunc) (wasDetached bool) {

	if !v.exclusive.CompareAndSwap(0, 1) {
		panic(throw.IllegalState())
	}
	defer v.exclusive.Store(0)

	w := &SimpleSlotWorker{ internalSlotWorker: internalSlotWorker{
		outerSignal: outer.GetSignalMark(),
		loopLimitFn: outer.CanLoopOrHasSignal,
		loopLimit: int(loopLimit),
		machine: m,
	}}

	fn(w)
	outer.AddNestedCallCount(w.callCount)
	return false
}

func (v *AttachableSimpleSlotWorker) AttachTo(m *smachine.SlotMachine, signal *synckit.SignalVersion,
	loopLimit uint32, fn smachine.AttachedFunc) (wasDetached bool, callCount uint) {

	if !v.exclusive.CompareAndSwap(0, 1) {
		panic(throw.IllegalState())
	}
	defer v.exclusive.Store(0)

	w := &SimpleSlotWorker{ internalSlotWorker: internalSlotWorker{
		outerSignal: signal,
		loopLimit: int(loopLimit),
		machine: m,
	}}

	fn(w)
	return false, w.callCount
}

var _ smachine.AttachedSlotWorker = &SimpleSlotWorker{}

type SimpleSlotWorker struct {
	internalSlotWorker
}

func (p *SimpleSlotWorker) AsFixedSlotWorker() smachine.FixedSlotWorker {
	return smachine.NewFixedSlotWorker(&p.internalSlotWorker)
}

func (p *SimpleSlotWorker) OuterCall(*smachine.SlotMachine, smachine.NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (p *SimpleSlotWorker) DetachableCall(fn smachine.DetachableFunc) (wasDetached bool) {
	if !p.detachable.CompareAndSwap(0, 1) {
		panic(throw.IllegalState())
	}
	defer p.detachable.Store(0)

	p.callCount++
	fn(smachine.NewDetachableSlotWorker(&p.internalSlotWorker))
	return false
}

var _ smachine.DetachableSlotWorkerSupport = &internalSlotWorker{}

type internalSlotWorker struct {
	outerSignal *synckit.SignalVersion
	loopLimitFn smachine.LoopLimiterFunc // NB! MUST correlate with outerSignal
	loopLimit   int
	callCount   uint
	detachable  atomickit.Uint32

	machine *smachine.SlotMachine
}

func (p *internalSlotWorker) AddNestedCallCount(u uint) {
	p.callCount += u
}

func (p *internalSlotWorker) TryDetach(smachine.LongRunFlags) {
	panic(throw.Unsupported())
}

func (p *internalSlotWorker) TryStartNonDetachableCall() bool {
	return p.detachable.CompareAndSwap(1, 2)
}

func (p *internalSlotWorker) EndNonDetachableCall() {
	p.detachable.Store(1)
}

func (p *internalSlotWorker) HasSignal() bool {
	return p.outerSignal != nil && p.outerSignal.HasSignal()
}

func (*internalSlotWorker) IsDetached() bool {
	return false
}

func (p *internalSlotWorker) GetSignalMark() *synckit.SignalVersion {
	return p.outerSignal
}

func (p *internalSlotWorker) CanLoopOrHasSignal(loopCount int) (canLoop, hasSignal bool) {
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

