// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type DependencyQueueController interface {
	GetName() string

	HasToReleaseOn(link SlotLink, flags SlotDependencyFlags, dblCheckFn func() bool) bool
	Release(link SlotLink, flags SlotDependencyFlags, chkAndRemoveFn func() bool) ([]PostponedDependency, []StepLink)
}

type queueControllerTemplate struct {
	name  string
	queue DependencyQueueHead
}

func (p *queueControllerTemplate) Init(name string, _ *sync.RWMutex, controller DependencyQueueController) {
	if p.queue.controller != nil {
		panic("illegal state")
	}
	p.name = name
	p.queue.controller = controller
}

func (p *queueControllerTemplate) isEmpty() bool {
	return p.queue.IsEmpty()
}

func (p *queueControllerTemplate) isEmptyOrFirst(link SlotLink) bool {
	f := p.queue.First()
	return f == nil || f.link == link
}

func (p *queueControllerTemplate) contains(entry *dependencyQueueEntry) bool {
	return entry.getQueue() == &p.queue
}

func (p *queueControllerTemplate) HasToReleaseOn(_ SlotLink, _ SlotDependencyFlags, dblCheckFn func() bool) bool {
	dblCheckFn()
	return false
}

func (p *queueControllerTemplate) GetName() string {
	return p.name
}

func (p *queueControllerTemplate) enum(qId int, fn EnumQueueFunc) bool {
	for item := p.queue.head.QueueNext(); item != nil; item = item.QueueNext() {
		flags := item.getFlags()
		if fn(qId, item.link, flags) {
			return true
		}
	}
	return false
}

type DependencyQueueFlags uint8

const (
	QueueAllowsPriority DependencyQueueFlags = 1 << iota
)

type DependencyQueueHead struct {
	controller DependencyQueueController
	head       dependencyQueueEntry
	count      int
	flags      DependencyQueueFlags
}

func (p *DependencyQueueHead) AddSlot(link SlotLink, flags SlotDependencyFlags, stacker *dependencyStack) *dependencyQueueEntry {
	return p._addSlot(link, flags, stacker, p.First)
}

func (p *DependencyQueueHead) addSlotForExclusive(link SlotLink, flags SlotDependencyFlags) *dependencyQueueEntry {
	return p._addSlot(link, flags, nil, func() *dependencyQueueEntry {
		if first := p.First(); first != nil {
			return first.QueueNext()
		}
		return nil
	})
}

func (p *DependencyQueueHead) _addSlot(link SlotLink, flags SlotDependencyFlags, stacker *dependencyStack,
	firstFn func() *dependencyQueueEntry,
) *dependencyQueueEntry {
	switch {
	case !link.IsValid():
		panic("illegal value")
	case flags&syncIgnoreFlags != 0:
		panic("illegal value")
	case p.flags&QueueAllowsPriority == 0:
		flags &^= syncPriorityMask
	}

	entry := &dependencyQueueEntry{link: link, slotFlags: flags}

	if flags&syncPriorityMask != 0 {
		if check := firstFn(); check != nil {
			f := check.getFlags()
			if f.hasLessPriorityThan(flags) {
				p._addBefore(check, entry)
				return entry
			}
		}
	}

	p.AddLast(entry)
	return entry
}

func (p *DependencyQueueHead) _addBefore(position, entry *dependencyQueueEntry) {
	p.initEmpty() // TODO PLAT-27 move to controller's Init
	entry.ensureNotInQueue()

	position._addQueuePrev(entry, entry)
	entry.setQueue(p)
	p.count++
}

func (p *DependencyQueueHead) AddFirst(entry *dependencyQueueEntry) {
	p._addBefore(p.head.nextInQueue, entry)
}

func (p *DependencyQueueHead) AddLast(entry *dependencyQueueEntry) {
	p._addBefore(&p.head, entry)
}

func (p *DependencyQueueHead) Count() int {
	return p.count
}

func (p *DependencyQueueHead) FirstValid() (*dependencyQueueEntry, StepLink) {
	for {
		f := p.head.QueueNext()
		if f == nil {
			return f, StepLink{}
		}
		if step, ok := f.link.GetStepLink(); ok {
			return f, step
		}
		f.removeFromQueue()
	}
}

func (p *DependencyQueueHead) First() *dependencyQueueEntry {
	return p.head.QueueNext()
}

func (p *DependencyQueueHead) Last() *dependencyQueueEntry {
	return p.head.QueuePrev()
}

func (p *DependencyQueueHead) IsZero() bool {
	return p.head.nextInQueue == nil
}

func (p *DependencyQueueHead) IsEmpty() bool {
	return p.head.nextInQueue == nil || p.head.nextInQueue.isQueueHead()
}

func (p *DependencyQueueHead) initEmpty() {
	if p.head.queue == nil {
		p.head.nextInQueue = &p.head
		p.head.prevInQueue = &p.head
		p.head.queue = p
	}
}

func (p *DependencyQueueHead) CutHeadOut(fn func(*dependencyQueueEntry) bool) {
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

func (p *DependencyQueueHead) CutTailOut(fn func(*dependencyQueueEntry) bool) {
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

func (p *DependencyQueueHead) FlushAllAsLinks() []StepLink {
	if p.count == 0 {
		return nil
	}

	deps := make([]StepLink, 0, p.count)
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

var _ SlotDependency = &dependencyQueueEntry{}

type dependencyQueueEntry struct {
	queue       *DependencyQueueHead  // atomic read when outside queue's lock
	nextInQueue *dependencyQueueEntry // access under queue's lock
	prevInQueue *dependencyQueueEntry // access under queue's lock
	stacker     *dependencyStack      // immutable

	link      SlotLink            // immutable
	slotFlags SlotDependencyFlags // immutable
}

func (p *dependencyQueueEntry) getFlags() SlotDependencyFlags {
	return p.slotFlags
}

func (p *dependencyQueueEntry) IsReleaseOnStepping() bool {
	return p.slotFlags&syncForOneStep != 0 || p.IsReleaseOnWorking()
}

func (p *dependencyQueueEntry) IsReleaseOnWorking() bool {
	for done := false; ; {
		queue := p.getQueue()
		if queue == nil {
			return true
		}
		result := queue.controller.HasToReleaseOn(p.link, p.slotFlags, func() bool {
			done = queue == p.getQueue()
			return done
		})
		if done {
			return result
		}
	}
}

func (p *dependencyQueueEntry) Release() (SlotDependency, []PostponedDependency, []StepLink) {
	d, s := p.ReleaseAll()
	return nil, d, s
}

func (p *dependencyQueueEntry) ReleaseAll() ([]PostponedDependency, []StepLink) {
	for done := false; ; {
		queue := p.getQueue()
		if queue == nil {
			return nil, nil
		}
		d, s := queue.controller.Release(p.link, p.slotFlags, func() bool {
			if queue == p.getQueue() {
				done = true
				if p.isInQueue() {
					p.removeFromQueue()
				}
			}
			return done
		})
		if done {
			return d, s
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

func (p *dependencyQueueEntry) getQueue() *DependencyQueueHead {
	return (*DependencyQueueHead)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&p.queue))))
}

func (p *dependencyQueueEntry) setQueue(head *DependencyQueueHead) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&p.queue)), unsafe.Pointer(head))
}

func (p *dependencyQueueEntry) IsCompatibleWith(requiredFlags SlotDependencyFlags) bool {
	switch {
	case requiredFlags == syncIgnoreFlags:
		return true
	case requiredFlags&syncPriorityMask == 0:
		// break
	case p.queue.flags&QueueAllowsPriority == 0:
		requiredFlags &^= syncPriorityMask
	}
	f := p.getFlags()
	return f.isCompatibleWith(requiredFlags)
}
