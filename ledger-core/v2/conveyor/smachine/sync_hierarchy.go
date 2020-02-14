// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import "sync"

func newSemaphoreChild(parent *semaphoreSync, value int, name string) DependencyController {
	if parent == nil {
		panic("illegal value")
	}
	if value <= 0 {
		panic("illegal value")
	}
	//panic("not implemented")
	sema := &hierarchySync{parentCtl: parent}

	parentQueue := &parent.controller.awaiters.queue
	sema.controller.parent = parentQueue
	sema.controller.queue.flags = parentQueue.flags

	sema.controller.workerLimit = value
	sema.controller.Init(name, &parent.mutex, &sema.controller)
	return sema
}

var _ DependencyController = &hierarchySync{}

type hierarchySync struct {
	parentCtl  *semaphoreSync
	controller subQueueController
}

func (p *hierarchySync) CheckState() BoolDecision {
	p.controller.mutex.RLock()
	defer p.controller.mutex.RUnlock()

	if !p.controller.canPassThrough() {
		return false
	}
	return p.parentCtl.checkState()
}

func (p *hierarchySync) CheckDependency(dep SlotDependency) Decision {
	p.controller.mutex.RLock()
	defer p.controller.mutex.RUnlock()

	if entry, ok := dep.(*dependencyQueueEntry); ok {
		return p.checkDependency(entry)
	}
	return Impossible
}

func (p *hierarchySync) checkDependency(entry *dependencyQueueEntry) Decision {
	switch {
	case p.controller.contains(entry):
		return NotPassed
	case p.controller.stackedOn(entry):
		return p.parentCtl.checkDependency(entry)
	}
	return Impossible
}

func (p *hierarchySync) UseDependency(dep SlotDependency, flags SlotDependencyFlags) Decision {
	p.controller.mutex.RLock()
	defer p.controller.mutex.RUnlock()

	if entry, ok := dep.(*dependencyQueueEntry); ok {
		if d := p.checkDependency(entry); d.IsValid() && entry.IsCompatibleWith(flags) {
			return d
		}
	}
	return Impossible
}

func (p *hierarchySync) CreateDependency(holder SlotLink, flags SlotDependencyFlags) (BoolDecision, SlotDependency) {
	p.controller.mutex.Lock()
	defer p.controller.mutex.Unlock()

	if p.controller.canPassThrough() {
		d, entry := p.parentCtl.createDependency(holder, flags)
		entry.stacker = &p.controller.stacker
		p.controller.workerCount++
		return d, entry
	}
	return false, p.controller.queue.AddSlot(holder, flags)
}

func (p *hierarchySync) GetLimit() (limit int, isAdjustable bool) {
	return p.controller.workerLimit, false
}

func (p *hierarchySync) AdjustLimit(limit int, absolute bool) (deps []StepLink, activate bool) {
	panic("illegal state")
}

func (p *hierarchySync) GetCounts() (active, inactive int) {
	p.controller.mutex.RLock()
	defer p.controller.mutex.RUnlock()

	return p.controller.workerCount, p.controller.queue.Count()
}

func (p *hierarchySync) GetName() string {
	return p.controller.GetName()
}

func (p *hierarchySync) EnumQueues(fn EnumQueueFunc) bool {
	p.controller.mutex.RLock()
	defer p.controller.mutex.RUnlock()

	return p.controller.enum(0, fn)
}

var _ DependencyQueueController = &subQueueController{}
var _ dependencyStackController = &subQueueController{}

type subQueueController struct {
	parent      *DependencyQueueHead
	stacker     dependencyStackEntry
	workerLimit int
	workerCount int
	waitingQueueController
}

func (p *subQueueController) Init(name string, mutex *sync.RWMutex, controller DependencyQueueController) {
	p.waitingQueueController.Init(name, mutex, controller)
	p.mutex = mutex
	p.stacker.controller = controller.(dependencyStackController)
}

func (p *subQueueController) canPassThrough() bool {
	return p.workerCount < p.workerLimit
}

func (p *subQueueController) stackedOn(entry *dependencyQueueEntry) bool {
	return entry.stacker == &p.stacker
}

// MUST be under the same lock as the parent
func (p *subQueueController) ReleaseStacked(releasedBy *dependencyQueueEntry, flags SlotDependencyFlags) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.workerCount--
	p.workerCount += p.moveToInactive(p.workerLimit-p.workerCount, p.parent, &p.stacker)
}
