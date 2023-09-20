package smsync

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

func NewExclusive(name string) smachine.SyncLink {
	return NewExclusiveWithFlags(name, 0)
}

func NewExclusiveWithFlags(name string, flags DependencyQueueFlags) smachine.SyncLink {
	ctl := &exclusiveSync{}
	ctl.awaiters.queue.flags = flags
	ctl.awaiters.Init(name, &ctl.mutex, &ctl.awaiters)
	return smachine.NewSyncLink(ctl)
}

type exclusiveSync struct {
	mutex    sync.RWMutex
	awaiters exclusiveQueueController
}

func (p *exclusiveSync) CheckState() smachine.BoolDecision {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return smachine.BoolDecision(p.awaiters.isEmpty())
}

func (p *exclusiveSync) UseDependency(dep smachine.SlotDependency, flags smachine.SlotDependencyFlags) smachine.Decision {
	if entry, ok := dep.(*dependencyQueueEntry); ok {
		p.mutex.RLock()
		defer p.mutex.RUnlock()

		switch {
		case !entry.link.IsValid(): // just to make sure
			return smachine.Impossible
		case !entry.IsCompatibleWith(flags):
			return smachine.Impossible
		case !p.awaiters.contains(entry):
			return smachine.Impossible
		case p.awaiters.isEmptyOrFirst(entry.link):
			return smachine.Passed
		default:
			return smachine.NotPassed
		}
	}
	return smachine.Impossible
}

func (p *exclusiveSync) ReleaseDependency(dep smachine.SlotDependency) (bool, smachine.SlotDependency, []smachine.PostponedDependency, []smachine.StepLink) {
	pd, sl := dep.ReleaseAll()
	return true, nil, pd, sl
}

func (p *exclusiveSync) CreateDependency(holder smachine.SlotLink, flags smachine.SlotDependencyFlags) (smachine.BoolDecision, smachine.SlotDependency) {
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

func (p *exclusiveSync) AdjustLimit(int, bool) (deps []smachine.StepLink, activate bool) {
	panic("illegal state")
}

func (p *exclusiveSync) EnumQueues(fn smachine.EnumQueueFunc) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.awaiters.enum(1, fn)
}

var _ dependencyQueueController = &exclusiveQueueController{}

type exclusiveQueueController struct {
	mutex *sync.RWMutex
	queueControllerTemplate
}

func (p *exclusiveQueueController) Init(name string, mutex *sync.RWMutex, controller dependencyQueueController) {
	p.queueControllerTemplate.Init(name, mutex, controller)
	p.mutex = mutex
}

func (p *exclusiveQueueController) SafeRelease(entry *dependencyQueueEntry, chkAndRemoveFn func() bool) ([]smachine.PostponedDependency, []smachine.StepLink) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// p.queue.First() must happen before chkAndRemoveFn()
	if f := p.queue.First(); f == nil || f != entry {
		chkAndRemoveFn()
		return nil, nil
	}

	if !chkAndRemoveFn() {
		return nil, nil
	}

	switch f, step := p.queue.FirstValid(); {
	case f == nil:
		return nil, nil
	//case f.stacker != nil:
	//	if postponed := f.stacker.popStack(&p.queue, f, step); postponed != nil {
	//		return nil, []PostponedDependency{postponed}, nil
	//	}
	//	fallthrough
	default:
		return nil, []smachine.StepLink{step}
	}
}

func (p *exclusiveQueueController) enum(qID int, fn smachine.EnumQueueFunc) bool {
	item := p.queue.head.QueueNext()
	if item == nil {
		return false
	}

	flags := item.getFlags()
	if fn(qID, item.link, flags) {
		return true
	}
	qID--

	for item = item.QueueNext(); item != nil; item = item.QueueNext() {
		flags := item.getFlags()
		if fn(qID, item.link, flags) {
			return true
		}
	}
	return false
}
