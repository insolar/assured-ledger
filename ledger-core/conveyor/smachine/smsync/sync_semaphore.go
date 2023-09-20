package smsync

import (
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// Semaphore allows Acquire() call to pass through for a number of workers within the limit.
func NewFixedSemaphore(limit int, name string) smachine.SyncLink {
	return NewFixedSemaphoreWithFlags(limit, name, 0)
}

func NewFixedSemaphoreWithFlags(limit int, name string, flags DependencyQueueFlags) smachine.SyncLink {
	if limit < 0 {
		panic("illegal value")
	}
	switch limit {
	case 0:
		return NewInfiniteLock(name)
	case 1:
		return NewExclusiveWithFlags(name, flags)
	default:
		return smachine.NewSyncLink(newSemaphore(limit, false, name, flags))
	}
}

// Semaphore allows Acquire() call to pass through for a number of workers within the limit.
// Negative and zero values are not passable.
// The limit can be changed with adjustments. Delta adjustments are capped by min/max int, no overflows.
func NewSemaphore(initialValue int, name string) SemaphoreLink {
	return NewSemaphoreWithFlags(initialValue, name, 0)
}

func NewSemaphoreWithFlags(initialValue int, name string, flags DependencyQueueFlags) SemaphoreLink {
	return SemaphoreLink{newSemaphore(initialValue, true, name, flags)}
}

type SemaphoreLink struct {
	ctl *semaphoreSync
}

func (v SemaphoreLink) IsZero() bool {
	return v.ctl == nil
}

func (v SemaphoreLink) NewDelta(delta int) smachine.SyncAdjustment {
	return smachine.NewSyncAdjustment(v.ctl, delta, false)
}

func (v SemaphoreLink) NewValue(value int) smachine.SyncAdjustment {
	return smachine.NewSyncAdjustment(v.ctl, value, true)
}

func (v SemaphoreLink) NewFixedChild(childValue int, name string) smachine.SyncLink {
	if childValue <= 0 {
		return NewInfiniteLock(name)
	}
	return v.NewChildExt(false, childValue, name, 0).SyncLink()
}

func (v SemaphoreLink) NewChild(childValue int, name string) SemaChildLink {
	return v.NewChildExt(true, childValue, name, 0)
}

func (v SemaphoreLink) NewChildExt(isAdjustable bool, childValue int, name string, flags SemaphoreChildFlags) SemaChildLink {
	if !isAdjustable && childValue <= 0 {
		panic(throw.IllegalValue())
	}
	return SemaChildLink{newSemaphoreChild(v.ctl, flags, childValue, isAdjustable, name)}
}

func (v SemaphoreLink) SyncLink() smachine.SyncLink {
	return smachine.NewSyncLink(v.ctl)
}

func (v SemaphoreLink) PartialLink() smachine.SyncLink {
	return smachine.NewSyncLink(semaPartial{v.ctl})
}

func newSemaphore(initialLimit int, isAdjustable bool, name string, flags DependencyQueueFlags) *semaphoreSync {
	ctl := &semaphoreSync{isAdjustable: true}
	ctl.controller.awaiters.queue.flags = flags
	ctl.controller.Init(name, &ctl.mutex, &ctl.controller)

	deps, _ := ctl.AdjustLimit(initialLimit, false)
	if len(deps) != 0 {
		panic("illegal state")
	}
	ctl.isAdjustable = isAdjustable
	return ctl
}

type semaphoreSync struct {
	mutex        sync.RWMutex
	controller   workingQueueController
	isAdjustable bool
}

func (p *semaphoreSync) CheckState() smachine.BoolDecision {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.checkState()
}

func (p *semaphoreSync) checkState() smachine.BoolDecision {
	return smachine.BoolDecision(p.controller.canPassThrough())
}

func (p *semaphoreSync) checkDependency(entry *dependencyQueueEntry, flags smachine.SlotDependencyFlags) (d smachine.Decision, passedStack bool) {
	d, notFound := p.checkDependencyHere(entry, flags)
	if notFound {
		d = entry.stacker.contains(&p.controller.awaiters.queue, entry)
		if d == smachine.Passed {
			return smachine.NotPassed, true
		}
		return d, false
	}
	return d, false

}

func (p *semaphoreSync) checkDependencyHere(entry *dependencyQueueEntry, flags smachine.SlotDependencyFlags) (d smachine.Decision, notFound bool) {
	switch {
	case !entry.link.IsValid(): // just to make sure
		//
	case !entry.IsCompatibleWith(flags):
		//
	case p.controller.contains(entry):
		return smachine.Passed, false
	case p.controller.containsInAwaiters(entry):
		return smachine.NotPassed, false
	default:
		return smachine.Impossible, true
	}
	return smachine.Impossible, false
}

func (p *semaphoreSync) UseDependency(dep smachine.SlotDependency, flags smachine.SlotDependencyFlags) smachine.Decision {
	if entry, ok := dep.(*dependencyQueueEntry); ok {
		if d, stackPassed := func() (smachine.Decision, bool) {
			p.mutex.RLock()
			defer p.mutex.RUnlock()
			return p.checkDependency(entry, flags)
		}(); flags == smachine.SyncIgnoreFlags || !stackPassed {
			return d
		}

		p.mutex.Lock()
		defer p.mutex.Unlock()

		return entry.stacker.tryPartialAcquire(entry, p.controller.canPassThrough())
	}
	return smachine.Impossible
}

func (p *semaphoreSync) ReleaseDependency(dep smachine.SlotDependency) (bool, smachine.SlotDependency, []smachine.PostponedDependency, []smachine.StepLink) {
	if entry, ok := dep.(*dependencyQueueEntry); ok && entry.stacker != nil {
		if func() bool {
			p.mutex.Lock()
			defer p.mutex.Unlock()

			return entry.stacker.tryPartialRelease(entry)
		}() {
			pd, sl := p.controller.updateQueue()
			return true, dep, pd, sl
		}
		return false, dep, nil, nil
	}
	pd, sl := dep.ReleaseAll()
	return true, nil, pd, sl
}

func (p *semaphoreSync) createDependency(holder smachine.SlotLink, flags smachine.SlotDependencyFlags) (smachine.BoolDecision, *dependencyQueueEntry) {
	if p.controller.canPassThrough() {
		return true, p.controller.queue.AddSlot(holder, flags, nil)
	}
	return false, p.controller.awaiters.queue.AddSlot(holder, flags, nil)
}

func (p *semaphoreSync) CreateDependency(holder smachine.SlotLink, flags smachine.SlotDependencyFlags) (smachine.BoolDecision, smachine.SlotDependency) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.createDependency(holder, flags)
}

func (p *semaphoreSync) GetLimit() (limit int, isAdjustable bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.controller.workerLimit, p.isAdjustable
}

func (p *semaphoreSync) AdjustLimit(limit int, absolute bool) ([]smachine.StepLink, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

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
	return p.controller.adjustLimit(delta, &p.controller.queue)
}

func (p *semaphoreSync) GetCounts() (active, inactive int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.controller.queue.Count(), p.controller.awaiters.queue.Count()
}

func (p *semaphoreSync) GetName() string {
	return p.controller.GetName()
}

func (p *semaphoreSync) EnumQueues(fn smachine.EnumQueueFunc) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.controller.enum(1, fn)
}

type waitingQueueController struct {
	mutex *sync.RWMutex
	queueControllerTemplate
}

func (p *waitingQueueController) Init(name string, mutex *sync.RWMutex, controller dependencyQueueController) {
	p.queueControllerTemplate.Init(name, mutex, controller)
	p.mutex = mutex
}

func (p *waitingQueueController) IsOpen(smachine.SlotDependency) bool {
	return false
}

func (p *waitingQueueController) SafeRelease(entry *dependencyQueueEntry, chkAndRemoveFn func() bool) ([]smachine.PostponedDependency, []smachine.StepLink) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	q := entry.queue
	if !chkAndRemoveFn() {
		return nil, nil
	}
	entry.stacker.updateAfterRelease(entry, q)

	return nil, nil
}

func (p *waitingQueueController) moveToInactive(n int, q *dependencyQueueHead) int {
	if n <= 0 {
		return 0
	}

	count := 0
	for n > 0 {
		if f, _ := p.queue.FirstValid(); f == nil {
			break
		} else {
			f.removeFromQueue()
			q.AddLast(f)
			count++
			n--
		}
	}
	return count
}

func (p *waitingQueueController) moveToActive(n int, q *dependencyQueueHead) ([]smachine.PostponedDependency, []smachine.StepLink) {
	if n <= 0 {
		return nil, nil
	}

	links := make([]smachine.StepLink, 0, n)
	for n > 0 {
		if f, step := p.queue.FirstValid(); f == nil {
			break
		} else {
			f.removeFromQueue()
			q.AddLast(f)
			links = append(links, step)
			n--
		}
	}
	return nil, links
}

type workingQueueController struct {
	queueControllerTemplate
	workerLimit int
	awaiters    waitingQueueController
}

func (p *workingQueueController) Init(name string, mutex *sync.RWMutex, controller dependencyQueueController) {
	p.queueControllerTemplate.Init(name, mutex, controller)
	p.awaiters.Init(name, mutex, &p.awaiters)
}

func (p *workingQueueController) canPassThrough() bool {
	return p.queue.Count() < p.workerLimit
}

func (p *workingQueueController) SafeRelease(entry *dependencyQueueEntry, chkAndRemoveFn func() bool) ([]smachine.PostponedDependency, []smachine.StepLink) {
	p.awaiters.mutex.Lock()
	defer p.awaiters.mutex.Unlock()

	q := entry.queue
	if !chkAndRemoveFn() {
		return nil, nil
	}
	entry.stacker.updateAfterRelease(entry, q)
	return p.updateQueue()
}

func (p *workingQueueController) updateQueue() ([]smachine.PostponedDependency, []smachine.StepLink) {
	return p.awaiters.moveToActive(p.workerLimit-p.queue.Count(), &p.queue)
}
func (p *workingQueueController) containsInAwaiters(entry *dependencyQueueEntry) bool {
	return p.awaiters.contains(entry)
}

func (p *workingQueueController) isQueueOfAwaiters(q *dependencyQueueHead) bool {
	return p.awaiters.isQueue(q)
}

func (p *workingQueueController) enum(qID int, fn smachine.EnumQueueFunc) bool {
	if p.queueControllerTemplate.enum(qID, fn) {
		return true
	}
	return p.awaiters.enum(qID-1, fn)
}

func (p *workingQueueController) adjustLimit(delta int, activeQ *dependencyQueueHead) ([]smachine.StepLink, bool) {
	if delta > 0 {
		links := make([]smachine.StepLink, 0, delta)
		p.awaiters.queue.CutHeadOut(func(entry *dependencyQueueEntry) bool {
			if step, ok := entry.link.GetStepLink(); ok {
				activeQ.AddLast(entry)
				links = append(links, step)
				return len(links) < delta
			}
			return true
		})
		return links, true
	}

	delta = -delta
	links := make([]smachine.StepLink, 0, delta)

	// sequence is reversed!
	p.queue.CutTailOut(func(entry *dependencyQueueEntry) bool {
		if step, ok := entry.link.GetStepLink(); ok {
			p.awaiters.queue.AddFirst(entry)
			links = append(links, step)
			return len(links) < delta
		}
		return true
	})
	return links, false
}

// pullAwaiters is for hierarchy children to activate awaiters after addition
func (p *workingQueueController) pullAwaiters() []smachine.StepLink {
	delta := p.workerLimit - p.queue.Count()
	if delta <= 0 {
		return nil
	}
	deps, activate := p.adjustLimit(delta, &p.queue)
	if !activate {
		panic(throw.Impossible())
	}
	return deps
}
