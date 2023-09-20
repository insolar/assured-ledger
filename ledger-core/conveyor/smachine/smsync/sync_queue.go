package smsync

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type dependencyQueueController interface {
	GetName() string

	HasToReleaseOn(link smachine.SlotLink, flags smachine.SlotDependencyFlags, dblCheckFn func() bool) bool
	SafeRelease(entry *dependencyQueueEntry, chkAndRemoveFn func() bool) ([]smachine.PostponedDependency, []smachine.StepLink)
}

type queueControllerTemplate struct {
	name  string
	queue dependencyQueueHead
}

func (p *queueControllerTemplate) Init(name string, _ *sync.RWMutex, controller dependencyQueueController) {
	if p.queue.controller != nil {
		panic("illegal state")
	}
	p.name = name
	p.queue.controller = controller
}

func (p *queueControllerTemplate) isEmpty() bool {
	return p.queue.IsEmpty()
}

func (p *queueControllerTemplate) isEmptyOrFirst(link smachine.SlotLink) bool {
	f := p.queue.First()
	return f == nil || f.link == link
}

func (p *queueControllerTemplate) contains(entry *dependencyQueueEntry) bool {
	return p.isQueue(entry.getQueue())
}

func (p *queueControllerTemplate) isQueue(q *dependencyQueueHead) bool {
	return q == &p.queue
}

func (p *queueControllerTemplate) HasToReleaseOn(_ smachine.SlotLink, _ smachine.SlotDependencyFlags, dblCheckFn func() bool) bool {
	dblCheckFn()
	return false
}

func (p *queueControllerTemplate) GetName() string {
	return p.name
}

func (p *queueControllerTemplate) enum(qID int, fn smachine.EnumQueueFunc) bool {
	for item := p.queue.head.QueueNext(); item != nil; item = item.QueueNext() {
		flags := item.getFlags()
		if fn(qID, item.link, flags) {
			return true
		}
	}
	return false
}

type DependencyQueueFlags uint8

const (
	QueueAllowsPriority DependencyQueueFlags = 1 << iota
)

type dependencyQueueHead struct {
	controller dependencyQueueController
	head       dependencyQueueEntry
	boostHead  *dependencyQueueEntry
	count      int
	flags      DependencyQueueFlags
}

func (p *dependencyQueueHead) AddSlot(link smachine.SlotLink, flags smachine.SlotDependencyFlags, stacker *dependencyStack) *dependencyQueueEntry {
	entry := &dependencyQueueEntry{link: link, slotFlags: flags, stacker: stacker}
	p.addSlotWithPriority(entry, false)
	return entry
}

func (p *dependencyQueueHead) addSlotForExclusive(link smachine.SlotLink, flags smachine.SlotDependencyFlags) *dependencyQueueEntry {
	entry := &dependencyQueueEntry{link: link, slotFlags: flags}
	p._addSlot(entry, true, false)
	return entry
}

func (p *dependencyQueueHead) addSlotWithPriority(entry *dependencyQueueEntry, addFirst bool) {
	p._addSlot(entry, false, addFirst)
}

func (p *dependencyQueueHead) _addSlot(entry *dependencyQueueEntry, skipFirst, addFirst bool) {
	switch {
	case !entry.link.IsValid():
		panic("illegal value")
	case entry.slotFlags&smachine.SyncIgnoreFlags == smachine.SyncIgnoreFlags:
		panic("illegal value")

	case p.flags&QueueAllowsPriority == 0:
		entry.slotFlags &^= smachine.SyncPriorityMask
		// p.boostHead is not needed when there are no priorities
	case entry.slotFlags&smachine.SyncPriorityHigh != 0 || addFirst:
		if first := p.First(); first != nil {
			if skipFirst {
				if first = first.QueueNext(); first == nil {
					break
				}
			}
			p._addBefore(first, entry)
			return
		}
	case entry.slotFlags&smachine.SyncPriorityBoosted != 0:
		if first := p.boostHead; first != nil {
			if skipFirst && first == p.head.nextInQueue {
				break
			}
			p._addBefore(first, entry)
			return
		}
	default: // no priority
		if p.boostHead == nil {
			p.boostHead = entry
		}
	}
	p.AddLast(entry)
}

func (p *dependencyQueueHead) _addBefore(position, entry *dependencyQueueEntry) {
	p.initEmpty() // TODO PLAT-27 move to controller's Init
	entry.ensureNotInQueue()

	position._addQueuePrev(entry, entry)
	entry.setQueue(p)
	p.count++
}

func (p *dependencyQueueHead) AddFirst(entry *dependencyQueueEntry) {
	p._addBefore(p.head.nextInQueue, entry)
}

func (p *dependencyQueueHead) AddLast(entry *dependencyQueueEntry) {
	p._addBefore(&p.head, entry)
}

func (p *dependencyQueueHead) Count() int {
	return p.count
}

func (p *dependencyQueueHead) FirstValid() (*dependencyQueueEntry, smachine.StepLink) {
	for {
		f := p.head.QueueNext()
		if f == nil {
			return f, smachine.StepLink{}
		}
		if step, ok := f.link.GetStepLink(); ok {
			return f, step
		}
		f.removeFromQueue()
	}
}

func (p *dependencyQueueHead) First() *dependencyQueueEntry {
	return p.head.QueueNext()
}

func (p *dependencyQueueHead) Last() *dependencyQueueEntry {
	return p.head.QueuePrev()
}

func (p *dependencyQueueHead) IsZero() bool {
	return p.head.nextInQueue == nil
}

func (p *dependencyQueueHead) IsEmpty() bool {
	return p.head.nextInQueue == nil || p.head.nextInQueue.isQueueHead()
}

func (p *dependencyQueueHead) initEmpty() {
	if p.head.queue == nil {
		p.head.nextInQueue = &p.head
		p.head.prevInQueue = &p.head
		p.head.queue = p
	}
}

func (p *dependencyQueueHead) CutHeadOut(fn func(*dependencyQueueEntry) bool) {
	for {
		entry := p.First()
		if entry == nil {
			return
		}
		entry.removeFromQueue()

		if !fn(entry) {
			return
		}
	}
}

func (p *dependencyQueueHead) CutTailOut(fn func(*dependencyQueueEntry) bool) {
	for {
		entry := p.Last()
		if entry == nil {
			return
		}
		entry.removeFromQueue()

		if !fn(entry) {
			return
		}
	}
}

func (p *dependencyQueueHead) FlushAllAsLinks() []smachine.StepLink {
	if p.count == 0 {
		return nil
	}

	deps := make([]smachine.StepLink, 0, p.count)
	for {
		entry := p.First()
		if entry == nil {
			break
		}
		entry.removeFromQueue()

		if step, ok := entry.link.GetStepLink(); ok {
			deps = append(deps, step)
		}
	}
	return deps
}

var _ smachine.SlotDependency = &dependencyQueueEntry{}

type dependencyQueueEntry struct {
	queue       *dependencyQueueHead  // atomic read when outside queue's lock
	nextInQueue *dependencyQueueEntry // access under queue's lock
	prevInQueue *dependencyQueueEntry // access under queue's lock
	stacker     *dependencyStack      // immutable

	link      smachine.SlotLink            // immutable
	slotFlags smachine.SlotDependencyFlags // immutable
}

func (p *dependencyQueueEntry) getFlags() smachine.SlotDependencyFlags {
	return p.slotFlags
}

func (p *dependencyQueueEntry) IsReleaseOnStepping() bool {
	return p.slotFlags&smachine.SyncForOneStep != 0 || p.IsReleaseOnWorking()
}

func (p *dependencyQueueEntry) IsReleaseOnWorking() bool {
	for done := false; ; {
		queue := p.getQueue()
		if queue == nil {
			return true
		}
		result := queue.controller.HasToReleaseOn(p.link, p.slotFlags, func() bool {
			// nolint
			done = queue == p.getQueue()
			// nolint
			return done
		})
		if done {
			return result
		}
	}
}

func (p *dependencyQueueEntry) ReleaseAll() ([]smachine.PostponedDependency, []smachine.StepLink) {
	return p._release(func(queue *dependencyQueueHead) bool {
		if queue != p.getQueue() {
			return false
		}
		if p.isInQueue() {
			p.removeFromQueue()
		}
		return true
	})
}

func (p *dependencyQueueEntry) _release(chkAndRemoveFn func(*dependencyQueueHead) bool) ([]smachine.PostponedDependency, []smachine.StepLink) {
	for done := false; ; {
		queue := p.getQueue()
		if queue == nil {
			return nil, nil
		}
		pd, sl := queue.controller.SafeRelease(p, func() bool {
			// nolint
			done = chkAndRemoveFn(queue)
			// nolint
			return done
		})
		if done {
			return pd, sl
		}
	}
}

func (p *dependencyQueueEntry) _addQueuePrev(chainHead, chainTail *dependencyQueueEntry) {
	p.ensureInQueue()

	prev := p.prevInQueue

	chainHead.prevInQueue = prev
	chainTail.nextInQueue = p

	p.prevInQueue = chainTail
	prev.nextInQueue = chainHead
}

func (p *dependencyQueueEntry) QueueNext() *dependencyQueueEntry {
	next := p.nextInQueue
	if next == nil || next.isQueueHead() {
		return nil
	}
	return next
}

func (p *dependencyQueueEntry) QueuePrev() *dependencyQueueEntry {
	prev := p.prevInQueue
	if prev == nil || prev.isQueueHead() {
		return nil
	}
	return prev
}

func (p *dependencyQueueEntry) removeFromQueue() {
	if p.isQueueHead() {
		panic("illegal state")
	}
	p.ensureInQueue()

	if p.queue.boostHead == p {
		p.queue.boostHead = p.QueueNext()
	}

	next := p.nextInQueue
	prev := p.prevInQueue

	next.prevInQueue = prev
	prev.nextInQueue = next

	p.queue.count--
	p.setQueue(nil)
	p.nextInQueue = nil
	p.prevInQueue = nil
}

func (p *dependencyQueueEntry) isQueueHead() bool {
	return p == &p.queue.head
}

func (p *dependencyQueueEntry) ensureNotInQueue() {
	if p.isInQueue() {
		panic("illegal state")
	}
}

func (p *dependencyQueueEntry) ensureInQueue() {
	if !p.isInQueue() {
		panic("illegal state")
	}
}

func (p *dependencyQueueEntry) isInQueue() bool {
	return p.queue != nil || p.nextInQueue != nil || p.prevInQueue != nil
}

func (p *dependencyQueueEntry) getQueue() *dependencyQueueHead {
	return (*dependencyQueueHead)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&p.queue))))
}

func (p *dependencyQueueEntry) setQueue(head *dependencyQueueHead) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&p.queue)), unsafe.Pointer(head))
}

func (p *dependencyQueueEntry) IsCompatibleWith(requiredFlags smachine.SlotDependencyFlags) bool {
	switch {
	case requiredFlags == smachine.SyncIgnoreFlags:
		return true
	case requiredFlags&smachine.SyncPriorityMask == 0:
		// break
	case p.queue.flags&QueueAllowsPriority == 0:
		requiredFlags &^= smachine.SyncPriorityMask
	}
	f := p.getFlags()
	return f.IsCompatibleWith(requiredFlags)
}
