// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smsync

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"sync"
	"sync/atomic"
	"unsafe"
)

type dependencyQueueController interface {
	GetName() string

	HasToReleaseOn(link smachine.SlotLink, flags smachine.SlotDependencyFlags, dblCheckFn func() bool) bool
	//SafeRelease(link smachine.SlotLink, flags smachine.SlotDependencyFlags, chkAndRemoveFn func() bool) ([]smachine.PostponedDependency, []smachine.StepLink)
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

func (p *queueControllerTemplate) enum(qId int, fn smachine.EnumQueueFunc) bool {
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

type dependencyQueueHead struct {
	controller dependencyQueueController
	head       dependencyQueueEntry
	count      int
	flags      DependencyQueueFlags
}

func (p *dependencyQueueHead) AddSlot(link smachine.SlotLink, flags smachine.SlotDependencyFlags, stacker *dependencyStack) *dependencyQueueEntry {
	entry := &dependencyQueueEntry{link: link, slotFlags: flags, stacker: stacker}
	p.addSlotWithPriority(entry)
	return entry
}

func (p *dependencyQueueHead) addSlotForExclusive(link smachine.SlotLink, flags smachine.SlotDependencyFlags) *dependencyQueueEntry {
	entry := &dependencyQueueEntry{link: link, slotFlags: flags}
	p._addSlot(entry, func() *dependencyQueueEntry {
		if first := p.First(); first != nil {
			return first.QueueNext()
		}
		return nil
	})
	return entry
}

func (p *dependencyQueueHead) addSlotWithPriority(entry *dependencyQueueEntry) {
	p._addSlot(entry, p.First)
}

func (p *dependencyQueueHead) _addSlot(entry *dependencyQueueEntry, firstFn func() *dependencyQueueEntry) {
	switch {
	case !entry.link.IsValid():
		panic("illegal value")
	case entry.slotFlags&smachine.SyncIgnoreFlags != 0:
		panic("illegal value")
	case p.flags&QueueAllowsPriority == 0:
		entry.slotFlags &^= smachine.SyncPriorityMask
	}

	if entry.slotFlags&smachine.SyncPriorityMask != 0 {
		if check := firstFn(); check != nil {
			f := check.getFlags()
			if f.HasLessPriorityThan(entry.slotFlags) {
				p._addBefore(check, entry)
				return
			}
		}
	}

	p.AddLast(entry)
	return
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
			done = queue == p.getQueue()
			return done
		})
		if done {
			return result
		}
	}
}

//func (p *dependencyQueueEntry) SafeRelease(c smachine.DependencyController) (sd smachine.SlotDependency, pd []smachine.PostponedDependency, sl []smachine.StepLink) {
//	if c == nil {
//		panic(throw.IllegalValue())
//	}
//	var pd0 smachine.PostponedDependency
//	pd, sl = p._release(func(queue *dependencyQueueHead) bool {
//		if queue != p.getQueue() {
//			return false
//		}
//		if p.isInQueue() {
//			p.removeFromQueue()
//			pd0, sd = p.stacker.popStack(queue, p)
//		}
//		return true
//	})
//	if pd0 != nil {
//		pd = append(pd, pd0)
//	}
//	return
//}
//
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
			done = chkAndRemoveFn(queue)
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
