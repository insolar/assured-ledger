// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import "sync"

func NewExclusive(name string) SyncLink {
	return NewExclusiveWithFlags(name, 0)
}

func NewExclusiveWithFlags(name string, flags DependencyQueueFlags) SyncLink {
	ctl := &exclusiveSync{}
	ctl.awaiters.queue.flags = flags
	ctl.awaiters.Init(name, &ctl.mutex, &ctl.awaiters)
	return NewSyncLink(ctl)
}

type exclusiveSync struct {
	mutex    sync.RWMutex
	awaiters exclusiveQueueController
}

func (p *exclusiveSync) CheckState() BoolDecision {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return BoolDecision(p.awaiters.isEmpty())
}

func (p *exclusiveSync) UseDependency(dep SlotDependency, flags SlotDependencyFlags) Decision {
	if entry, ok := dep.(*dependencyQueueEntry); ok {
		p.mutex.RLock()
		defer p.mutex.RUnlock()

		switch {
		case !entry.link.IsValid(): // just to make sure
			return Impossible
		case !entry.IsCompatibleWith(flags):
			return Impossible
		case !p.awaiters.contains(entry):
			return Impossible
		case p.awaiters.isEmptyOrFirst(entry.link):
			return Passed
		default:
			return NotPassed
		}
	}
	return Impossible
}

func (p *exclusiveSync) CreateDependency(holder SlotLink, flags SlotDependencyFlags) (BoolDecision, SlotDependency) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	sd := p.awaiters.queue.addSlotForExclusive(holder, flags)
	if f, _ := p.awaiters.queue.FirstValid(); f == sd {
		return true, sd
	}
	return false, sd
}

func (p *exclusiveSync) GetCounts() (active, inactive int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	n := p.awaiters.queue.Count()
	if n <= 0 {
		return 0, n
	}
	return 1, n - 1
}

func (p *exclusiveSync) GetName() string {
	return p.awaiters.GetName()
}

func (p *exclusiveSync) GetLimit() (limit int, isAdjustable bool) {
	return 1, false
}

func (p *exclusiveSync) AdjustLimit(int, bool) (deps []StepLink, activate bool) {
	panic("illegal state")
}

func (p *exclusiveSync) EnumQueues(fn EnumQueueFunc) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.awaiters.enum(1, fn)
}

var _ DependencyQueueController = &exclusiveQueueController{}

type exclusiveQueueController struct {
	mutex *sync.RWMutex
	queueControllerTemplate
}

func (p *exclusiveQueueController) Init(name string, mutex *sync.RWMutex, controller DependencyQueueController) {
	p.queueControllerTemplate.Init(name, mutex, controller)
	p.mutex = mutex
}

func (p *exclusiveQueueController) Release(link SlotLink, _ SlotDependencyFlags, chkAndRemoveFn func() bool) ([]PostponedDependency, []StepLink) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if f := p.queue.First(); f == nil || f.link != link {
		chkAndRemoveFn()
		return nil, nil
	}

	if !chkAndRemoveFn() {
		return nil, nil
	}

	switch f, step := p.queue.FirstValid(); {
	case f == nil:
		return nil, nil
	case f.stacker != nil:
		if postponed := f.stacker.PushStack(f, step); postponed != nil {
			return []PostponedDependency{postponed}, nil
		}
		fallthrough
	default:
		return nil, []StepLink{step}
	}
}

func (p *exclusiveQueueController) enum(qId int, fn EnumQueueFunc) bool {
	item := p.queue.head.QueueNext()
	if item == nil {
		return false
	}

	flags := item.getFlags()
	if fn(qId, item.link, flags) {
		return true
	}
	qId--

	for item = item.QueueNext(); item != nil; item = item.QueueNext() {
		flags := item.getFlags()
		if fn(qId, item.link, flags) {
			return true
		}
	}
	return false
}
