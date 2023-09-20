package smsync

import (
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

// ConditionalBool allows Acquire() call to pass through when current value is >0
func NewConditional(initial int, name string) ConditionalLink {
	ctl := &conditionalSync{}
	ctl.controller.Init(name, &ctl.mutex, &ctl.controller)
	deps, _ := ctl.AdjustLimit(initial, false)
	if len(deps) != 0 {
		panic("illegal state")
	}
	return ConditionalLink{ctl}
}

type ConditionalLink struct {
	ctl *conditionalSync
}

func (v ConditionalLink) IsZero() bool {
	return v.ctl == nil
}

// Creates an adjustment that alters the conditional's value when the adjustment is applied with SynchronizationContext.ApplyAdjustment()
// Can be applied multiple times.
func (v ConditionalLink) NewDelta(delta int) smachine.SyncAdjustment {
	return smachine.NewSyncAdjustment(v.ctl, delta, false)
}

// Creates an adjustment that sets the given value when applied with SynchronizationContext.ApplyAdjustment()
// Can be applied multiple times.
func (v ConditionalLink) NewValue(value int) smachine.SyncAdjustment {
	return smachine.NewSyncAdjustment(v.ctl, value, true)
}

func (v ConditionalLink) SyncLink() smachine.SyncLink {
	return smachine.NewSyncLink(v.ctl)
}

type conditionalSync struct {
	mutex      sync.RWMutex
	controller holdingQueueController
}

func (p *conditionalSync) CheckState() smachine.BoolDecision {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return smachine.BoolDecision(p.controller.canPassThrough())
}

func (p *conditionalSync) UseDependency(dep smachine.SlotDependency, flags smachine.SlotDependencyFlags) smachine.Decision {
	if entry, ok := dep.(*dependencyQueueEntry); ok {
		p.mutex.RLock()
		defer p.mutex.RUnlock()

		switch {
		case !entry.link.IsValid(): // just to make sure
			return smachine.Impossible
		case !entry.IsCompatibleWith(flags):
			return smachine.Impossible
		case !p.controller.contains(entry):
			return smachine.Impossible
		case p.controller.canPassThrough():
			return smachine.Passed
		default:
			return smachine.NotPassed
		}
	}
	return smachine.Impossible
}

func (p *conditionalSync) ReleaseDependency(dep smachine.SlotDependency) (bool, smachine.SlotDependency, []smachine.PostponedDependency, []smachine.StepLink) {
	pd, sl := dep.ReleaseAll()
	return true, nil, pd, sl
}

func (p *conditionalSync) CreateDependency(holder smachine.SlotLink, flags smachine.SlotDependencyFlags) (smachine.BoolDecision, smachine.SlotDependency) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.controller.canPassThrough() {
		return true, nil
	}
	return false, p.controller.queue.AddSlot(holder, flags, nil)
}

func (p *conditionalSync) GetLimit() (limit int, isAdjustable bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.controller.state, true
}

func (p *conditionalSync) AdjustLimit(limit int, absolute bool) ([]smachine.StepLink, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if ok, newState := applyWrappedAdjustment(p.controller.state, limit, math.MinInt32, math.MaxInt32, absolute); ok {
		return p.setLimit(newState)
	}
	return nil, false
}

func (p *conditionalSync) setLimit(limit int) ([]smachine.StepLink, bool) {
	p.controller.state = limit
	if !p.controller.canPassThrough() {
		return nil, false
	}
	return p.controller.queue.FlushAllAsLinks(), true
}

func (p *conditionalSync) GetCounts() (active, inactive int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return -1, p.controller.queue.Count()
}

func (p *conditionalSync) GetName() string {
	return p.controller.GetName()
}

func (p *conditionalSync) EnumQueues(fn smachine.EnumQueueFunc) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.controller.enum(0, fn)
}

type holdingQueueController struct {
	mutex *sync.RWMutex
	queueControllerTemplate
	state int
}

func (p *holdingQueueController) Init(name string, mutex *sync.RWMutex, controller dependencyQueueController) {
	p.queueControllerTemplate.Init(name, mutex, controller)
	p.mutex = mutex
}

func (p *holdingQueueController) canPassThrough() bool {
	return p.state > 0
}

func (p *holdingQueueController) IsOpen(smachine.SlotDependency) bool {
	return false // is still in queue ...
}

func (p *holdingQueueController) SafeRelease(_ *dependencyQueueEntry, chkAndRemoveFn func() bool) ([]smachine.PostponedDependency, []smachine.StepLink) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	chkAndRemoveFn()
	if p.canPassThrough() && p.queue.Count() > 0 {
		panic("illegal state")
	}
	return nil, nil
}

func (p *holdingQueueController) HasToReleaseOn(_ smachine.SlotLink, _ smachine.SlotDependencyFlags, dblCheckFn func() bool) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return dblCheckFn() && p.canPassThrough()
}

func applyWrappedAdjustment(current, adjustment, min, max int, absolute bool) (bool, int) {
	if absolute {
		if current == adjustment {
			return false, current
		}
		if adjustment < min {
			return true, min
		}
		if adjustment > max {
			return true, max
		}
		return true, adjustment
	}

	if adjustment == 0 {
		return false, current
	}
	if adjustment < 0 {
		adjustment += current
		if adjustment < min || adjustment > current /* overflow */ {
			return true, min
		}
		return true, adjustment
	}

	adjustment += current
	if adjustment > max || adjustment < current /* overflow */ {
		return true, max
	}
	return true, adjustment
}
