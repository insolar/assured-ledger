// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smsync

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"math"
	"sync"
)

type SemaphoreChildFlags uint8

const (
	CanReleaseParent SemaphoreChildFlags = 1 << iota
	ParentReacquireBoost
)

func newSemaphoreChild(parent *semaphoreSync, flags SemaphoreChildFlags, value int, name string) smachine.DependencyController {
	if parent == nil {
		panic("illegal value")
	}
	if value <= 0 {
		panic("illegal value")
	}
	//panic("not implemented")
	sema := &hierarchySync{}

	sema.controller.parentSync = parent
	sema.controller.queue.flags = parent.controller.awaiters.queue.flags

	sema.controller.workerLimit = value
	sema.controller.Init(name, &parent.mutex, &sema.controller)
	return sema
}

var _ smachine.DependencyController = &hierarchySync{}

type hierarchySync struct {
	controller   subSemaphoreController
	isAdjustable bool
}

func (p *hierarchySync) CheckState() smachine.BoolDecision {
	p.controller.awaiters.mutex.RLock()
	defer p.controller.awaiters.mutex.RUnlock()

	if !p.controller.canPassThrough() {
		return false
	}
	return p.controller.parentSync.checkState()
}

func (p *hierarchySync) UseDependency(dep smachine.SlotDependency, flags smachine.SlotDependencyFlags) smachine.Decision {
	if entry, ok := dep.(*dependencyQueueEntry); ok {
		p.controller.awaiters.mutex.RLock()
		defer p.controller.awaiters.mutex.RUnlock()

		switch {
		case !entry.link.IsValid(): // just to make sure
			return smachine.Impossible
		case !entry.IsCompatibleWith(flags):
			return smachine.Impossible
		case !p.controller.owns(entry):
			return smachine.Impossible
		case p.controller.containsInAwaiters(entry):
			return smachine.NotPassed
		case p.controller.contains(entry):
			return smachine.Impossible // this is a special case - if caller has released the parent sema, then it has to acquire the parent sema
		default:
			d, _ := p.controller.parentSync.checkDependencyHere(entry, smachine.SyncIgnoreFlags)
			return d
		}
	}
	return smachine.Impossible
}

func (p *hierarchySync) ReleaseDependency(dep smachine.SlotDependency) (smachine.SlotDependency, []smachine.PostponedDependency, []smachine.StepLink) {
	pd, sl := dep.ReleaseAll()
	return nil, pd, sl
}

func (p *hierarchySync) CreateDependency(holder smachine.SlotLink, flags smachine.SlotDependencyFlags) (smachine.BoolDecision, smachine.SlotDependency) {
	p.controller.awaiters.mutex.Lock()
	defer p.controller.awaiters.mutex.Unlock()

	if p.controller.canPassThrough() {
		d, entry := p.controller.parentSync.createDependency(holder, flags)
		entry.stacker = &p.controller.stacker
		p.controller.workerAtParentCount++
		return d, entry
	}
	return false, p.controller.queue.AddSlot(holder, flags, nil)
}

func (p *hierarchySync) GetLimit() (limit int, isAdjustable bool) {
	return p.controller.workerLimit, isAdjustable
}

func (p *hierarchySync) AdjustLimit(limit int, absolute bool) (deps []smachine.StepLink, activate bool) {
	p.controller.awaiters.mutex.Lock()
	defer p.controller.awaiters.mutex.Unlock()

	if !p.isAdjustable {
		panic("illegal state")
	}

	if ok, newLimit := applyWrappedAdjustment(p.controller.workerLimit, limit, math.MinInt32, math.MaxInt32, absolute); ok {
		limit = newLimit
	} else {
		return nil, false
	}

	delta := limit - p.controller.workerLimit
	p.controller.workerLimit = limit
	if delta < 0 {
		// can't revoke from an active
		return nil, false
	}
	return p.controller.adjustLimit(delta, p.controller.getParentAwaitQueue())
}

func (p *hierarchySync) GetCounts() (active, inactive int) {
	p.controller.awaiters.mutex.RLock()
	defer p.controller.awaiters.mutex.RUnlock()

	return p.controller.workerAtParentCount + p.controller.queue.Count(), p.controller.awaiters.queue.Count()
}

func (p *hierarchySync) GetName() string {
	return p.controller.GetName()
}

func (p *hierarchySync) EnumQueues(fn smachine.EnumQueueFunc) bool {
	p.controller.awaiters.mutex.RLock()
	defer p.controller.awaiters.mutex.RUnlock()

	return p.controller.enum(1, fn)
}

type dependencyStackQueueController interface {
	dependencyQueueController
	dependencyStackController
}

var _ dependencyStackController = &subSemaphoreController{}

type subSemaphoreController struct {
	parentSync *semaphoreSync
	stacker    dependencyStack

	workerAtParentCount int

	workingQueueController
}

func (p *subSemaphoreController) getParentAwaitQueue() *dependencyQueueHead {
	return &p.parentSync.controller.awaiters.queue
}

func (p *subSemaphoreController) getParentActiveQueue() *dependencyQueueHead {
	return &p.parentSync.controller.queue
}

func (p *subSemaphoreController) Init(name string, mutex *sync.RWMutex, controller dependencyStackQueueController) {
	p.workingQueueController.Init(name, mutex, controller)
	p.stacker.controller = controller
}

func (p *subSemaphoreController) canPassThrough() bool {
	return p.queue.Count()+p.workerAtParentCount < p.workerLimit
}

func (p *subSemaphoreController) owns(entry *dependencyQueueEntry) bool {
	return entry.stacker == &p.stacker
}

func (p *subSemaphoreController) SafeRelease(_ *dependencyQueueEntry, chkAndRemoveFn func() bool) ([]smachine.PostponedDependency, []smachine.StepLink) {
	p.awaiters.mutex.Lock()
	defer p.awaiters.mutex.Unlock()

	if !chkAndRemoveFn() {
		return nil, nil
	}

	// TODO how to detect release of a queue item by parent? pass slot dependency!?
	//	p.workerAtParentCount--
	pd, sl := p.awaiters.moveToActive(p.workerLimit-p.queue.Count()-p.workerAtParentCount, p.getParentAwaitQueue())
	p.workerAtParentCount += len(sl)
	return pd, sl
}

//// MUST be under the same lock as the parent
//func (p *subSemaphoreController) ReleaseStacked(*dependencyQueueEntry) {
//	p.mutex.Lock()
//	defer p.mutex.Unlock()
//
//	p.workerCount--
//	p.workerCount += p.moveToInactive(p.workerLimit-p.workerCount, p.parent, &p.stacker)
//}

func (p *subSemaphoreController) containsInStack(q *dependencyQueueHead, entry *dependencyQueueEntry) smachine.Decision {
	switch {
	case !p.owns(entry):
		//
	case p.containsInAwaiters(entry):
		return smachine.NotPassed
	case p.contains(entry):
		return smachine.Passed
	}
	return smachine.Impossible
}

func (p *subSemaphoreController) popStack(q *dependencyQueueHead, entry *dependencyQueueEntry) (smachine.PostponedDependency, *dependencyQueueEntry) {
	panic("implement me")
	//p.queue.AddLast(entry)
}
