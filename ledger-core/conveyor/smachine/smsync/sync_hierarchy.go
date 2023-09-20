package smsync

import (
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SemaphoreChildFlags uint8

const (
	AllowPartialRelease SemaphoreChildFlags = 1 << iota
	PrioritizePartialAcquire
)

type SemaChildLink struct {
	ctl *hierarchySync
}

func (v SemaChildLink) IsZero() bool {
	return v.ctl == nil
}

func (v SemaChildLink) NewDelta(delta int) smachine.SyncAdjustment {
	return smachine.NewSyncAdjustment(v.ctl, delta, false)
}

func (v SemaChildLink) NewValue(value int) smachine.SyncAdjustment {
	return smachine.NewSyncAdjustment(v.ctl, value, true)
}

func (v SemaChildLink) SyncLink() smachine.SyncLink {
	return smachine.NewSyncLink(v.ctl)
}

func newSemaphoreChild(parent *semaphoreSync, flags SemaphoreChildFlags, value int, isAdjustable bool, name string) *hierarchySync {
	if parent == nil {
		panic(throw.IllegalValue())
	}
	sema := &hierarchySync{}

	sema.isAdjustable = isAdjustable
	sema.controller.parentSync = parent
	parentFlags := parent.controller.awaiters.queue.flags
	sema.controller.queue.flags = parentFlags

	if parentFlags&QueueAllowsPriority == 0 {
		flags &^= PrioritizePartialAcquire
	}

	sema.controller.flags = flags
	sema.controller.workerLimit = value
	sema.controller.Init(name, &parent.mutex, &sema.controller)
	return sema
}

var _ smachine.DependencyController = &hierarchySync{}

type hierarchySync struct {
	controller   subSemaQueueController
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
			if flags == smachine.SyncIgnoreFlags {
				return smachine.NotPassed
			}
			// this is a special case - if caller has released the parent sema, then it has to acquire the parent sema, not this sema
			return smachine.Impossible
		default:
			d, _ := p.controller.parentSync.checkDependencyHere(entry, smachine.SyncIgnoreFlags)
			return d
		}
	}
	return smachine.Impossible
}

func (p *hierarchySync) ReleaseDependency(dep smachine.SlotDependency) (bool, smachine.SlotDependency, []smachine.PostponedDependency, []smachine.StepLink) {
	pd, sl := dep.ReleaseAll()
	return true, nil, pd, sl
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
	return false, p.controller.awaiters.queue.AddSlot(holder, flags, &p.controller.stacker)
}

func (p *hierarchySync) GetLimit() (limit int, isAdjustable bool) {
	return p.controller.workerLimit, p.isAdjustable
}

func (p *hierarchySync) AdjustLimit(limit int, absolute bool) (deps []smachine.StepLink, activate bool) {
	p.controller.awaiters.mutex.Lock()
	defer p.controller.awaiters.mutex.Unlock()

	if !p.isAdjustable {
		panic(throw.IllegalState())
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

	deps, activate = p.controller.adjustLimit(delta, p.controller.getParentAwaitQueue())
	// NB! MUST NOT immediately release these slots as they were added to parent's wait queue
	if n := len(deps); activate && n > 0 {
		p.controller.workerAtParentCount += n
		deps = p.controller.pushParentAwaiters()
	}

	return
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

var _ dependencyStackController = &subSemaQueueController{}

type subSemaQueueController struct {
	parentSync *semaphoreSync
	stacker    dependencyStack
	flags      SemaphoreChildFlags

	workerAtParentCount int

	workingQueueController
}

func (p *subSemaQueueController) getParentAwaitQueue() *dependencyQueueHead {
	return &p.parentSync.controller.awaiters.queue
}

func (p *subSemaQueueController) getParentActiveQueue() *dependencyQueueHead {
	return &p.parentSync.controller.queue
}

func (p *subSemaQueueController) Init(name string, mutex *sync.RWMutex, controller dependencyStackQueueController) {
	p.workingQueueController.Init(name, mutex, controller)
	p.stacker.controller = controller
}

func (p *subSemaQueueController) canPassThrough() bool {
	return p.queue.Count()+p.workerAtParentCount < p.workerLimit
}

func (p *subSemaQueueController) owns(entry *dependencyQueueEntry) bool {
	return entry.stacker == &p.stacker
}

func (p *subSemaQueueController) SafeRelease(_ *dependencyQueueEntry, chkAndRemoveFn func() bool) ([]smachine.PostponedDependency, []smachine.StepLink) {
	p.awaiters.mutex.Lock()
	defer p.awaiters.mutex.Unlock()

	if !chkAndRemoveFn() {
		return nil, nil
	}
	p.updateQueue()
	return nil, nil
}

func (p *subSemaQueueController) updateQueue() {
	p.workerAtParentCount += p.awaiters.moveToInactive(p.workerLimit-p.queue.Count()-p.workerAtParentCount, p.getParentAwaitQueue())
}

//// MUST be under the same lock as the parent
//func (p *subSemaQueueController) ReleaseStacked(*dependencyQueueEntry) {
//	p.mutex.Lock()
//	defer p.mutex.Unlock()
//
//	p.workerCount--
//	p.workerCount += p.moveToInactive(p.workerLimit-p.workerCount, p.parent, &p.stacker)
//}

func (p *subSemaQueueController) containsInStack(q *dependencyQueueHead, entry *dependencyQueueEntry) smachine.Decision {
	switch {
	case q != p.getParentAwaitQueue():
		// wrong parent
	case !p.owns(entry):
		panic(throw.IllegalState())
	case p.containsInAwaiters(entry):
		return smachine.NotPassed
	case p.contains(entry):
		return smachine.Passed
	}
	return smachine.Impossible
}

func (p *subSemaQueueController) afterRelease(q *dependencyQueueHead, entry *dependencyQueueEntry) {
	switch {
	case !p.owns(entry):
		//
	case p.getParentAwaitQueue() == q || p.getParentActiveQueue() == q:
		p.workerAtParentCount--
		p.updateQueue()
	}
}

func (p *subSemaQueueController) tryPartialRelease(entry *dependencyQueueEntry) bool {
	if p.flags&AllowPartialRelease == 0 || !p.owns(entry) {
		return false
	}
	if q := entry.getQueue(); p.getParentAwaitQueue() != q && p.getParentActiveQueue() != q {
		return false
	}
	entry.removeFromQueue()
	p.workerAtParentCount--
	p.queue.AddLast(entry)
	return true
}

func (p *subSemaQueueController) tryPartialAcquire(entry *dependencyQueueEntry, hasActiveCapacity bool) smachine.Decision {
	if p.flags&AllowPartialRelease == 0 || !p.owns(entry) {
		return smachine.Impossible
	}
	if q := entry.getQueue(); !p.isQueue(q) && !p.isQueueOfAwaiters(q) {
		return smachine.Impossible
	}

	entry.removeFromQueue()
	p.workerAtParentCount++

	if hasActiveCapacity {
		p.getParentActiveQueue().AddLast(entry)
		return smachine.Passed
	}

	p.getParentAwaitQueue().addSlotWithPriority(entry, p.flags&PrioritizePartialAcquire != 0)
	return smachine.NotPassed
}

func (p *subSemaQueueController) pushParentAwaiters() []smachine.StepLink {
	return p.parentSync.controller.pullAwaiters()
}
