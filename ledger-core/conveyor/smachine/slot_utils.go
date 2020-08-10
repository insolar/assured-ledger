// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

func (s *Slot) activateSlot(worker FixedSlotWorker) {
	s.machine.updateSlotQueue(s, worker, activateSlot)
}

func (p SlotLink) activateSlot(worker FixedSlotWorker) {
	if p.IsValid() {
		p.s.activateSlot(worker)
	}
}

func (p StepLink) activateSlotStep(worker FixedSlotWorker) {
	if p.IsAtStep() {
		p.s.activateSlot(worker)
	}
}

func (p StepLink) activateSlotStepWithSlotLink(_ SlotLink, worker FixedSlotWorker) {
	p.activateSlotStep(worker)
}

func (p SlotLink) activateOnNonBusy(waitOn SlotLink, worker DetachableSlotWorker) bool {
	switch {
	case !p.IsValid():
		// requester is dead, don't wait anymore and don't wake it up
		return true
	case worker.IsZero():
		// too many retries - have to wake up the requester
	case waitOn.isValidAndBusy():
		// someone got it already, this callback should be added back to the queue
		return false
	}

	// lets try to wake up with synchronous operations
	m := p.getActiveMachine()
	switch {
	case m == nil:
		// requester is dead, don't wait anymore and don't wake it up
		return true
	case worker.IsZero():
		//
	case !waitOn.isMachine(m):
		if worker.NonDetachableOuterCall(m, p.s.activateSlot) {
			return true
		}
	case worker.NonDetachableCall(p.s.activateSlot):
		return true
	}

	// last resort - try to wake up via asynchronous
	m.syncQueue.AddAsyncUpdate(p, SlotLink.activateSlot)
	return true
}

func buildShadowMigrator(localInjects []interface{}, defFn ShadowMigrateFunc) ShadowMigrateFunc {
	count := len(localInjects)
	if defFn != nil {
		count++
	}
	shadowMigrates := make([]ShadowMigrateFunc, 0, count)

	for _, v := range localInjects {
		if smFn, ok := v.(ShadowMigrator); ok {
			shadowMigrates = append(shadowMigrates, smFn.ShadowMigrate)
		}
	}

	switch {
	case len(shadowMigrates) == 0:
		return defFn
	case defFn != nil:
		shadowMigrates = append(shadowMigrates, defFn)
	}
	if len(shadowMigrates)+1 < cap(shadowMigrates) { // allow only a minimal oversize
		shadowMigrates = append([]ShadowMigrateFunc(nil), shadowMigrates...)
	}

	return func(start, delta uint32) {
		for _, fn := range shadowMigrates {
			fn(start, delta)
		}
	}
}

func (s *Slot) _releaseDependency(controller DependencyController) ([]StepLink, bool) {
	if controller == nil {
		panic(throw.IllegalValue())
	}
	dep := s.dependency
	s.dependency = nil
	wasReleased, replace, postponed, released := controller.ReleaseDependency(dep)
	s.dependency = replace

	released = PostponedList(postponed).PostponedActivate(released)
	return released, wasReleased
}

func (s *Slot) _releaseAllDependency() []StepLink {
	dep := s.dependency
	s.dependency = nil

	postponed, released := dep.ReleaseAll()
	released = PostponedList(postponed).PostponedActivate(released)
	return released
}

var _ PostponedDependency = &PostponedList{}

type PostponedList []PostponedDependency

func (p PostponedList) PostponedActivate(appendTo []StepLink) []StepLink {
	for _, d := range p {
		appendTo = d.PostponedActivate(appendTo)
	}
	return appendTo
}
