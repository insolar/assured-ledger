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

	w := internalSlotWorker{
		outerSignal: outer.GetSignalMark(),
		loopLimitFn: outer.CanLoopOrHasSignal,
		loopLimit: int(loopLimit),
		machine: m,
	}

	fn(smachine.NewAttachedSlotWorker(&w))
	outer.AddNestedCallCount(w.callCount)
	return false
}

func (v *AttachableSimpleSlotWorker) AttachTo(m *smachine.SlotMachine, signal *synckit.SignalVersion,
	loopLimit uint32, fn smachine.AttachedFunc) (wasDetached bool, callCount uint) {

	if !v.exclusive.CompareAndSwap(0, 1) {
		panic(throw.IllegalState())
	}
	defer v.exclusive.Store(0)

	w := internalSlotWorker{
		outerSignal: signal,
		loopLimit: int(loopLimit),
		machine: m,
	}

	fn(smachine.NewAttachedSlotWorker(&w))
	return false, w.callCount
}


const (
	workerStateAttached = iota
	workerStateDetachable
	workerStateFixedInDetachable
)

var _ smachine.SlotWorkerSupport = &internalSlotWorker{}

type internalSlotWorker struct {
	outerSignal *synckit.SignalVersion
	loopLimitFn smachine.LoopLimiterFunc // NB! MUST correlate with outerSignal
	loopLimit   int
	callCount   uint
	state       atomickit.Uint32

	machine *smachine.SlotMachine
}

func (p *internalSlotWorker) TryStartDetachableCall() bool {
	if p.state.CompareAndSwap(workerStateAttached, workerStateDetachable) {
		p.callCount++
		return true
	}
	return false
}

func (p *internalSlotWorker) EndDetachableCall() (wasDetached bool) {
	if !p.state.CompareAndSwap(workerStateDetachable, workerStateAttached) {
		panic(throw.Impossible())
	}
	return false
}

func (p *internalSlotWorker) AddNestedCallCount(u uint) {
	p.callCount += u
}

func (p *internalSlotWorker) TryDetach(smachine.LongRunFlags) {
	panic(throw.Unsupported())
}

func (p *internalSlotWorker) TryStartNonDetachableCall() bool {
	return p.state.CompareAndSwap(workerStateDetachable, workerStateFixedInDetachable)
}

func (p *internalSlotWorker) EndNonDetachableCall() {
	if !p.state.CompareAndSwap(workerStateFixedInDetachable, workerStateDetachable) {
		panic(throw.Impossible())
	}
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

