package smsync

import "github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"

// methods of this interfaces can be protected by mutex
type dependencyStackController interface {
	containsInStack(q *dependencyQueueHead, entry *dependencyQueueEntry) smachine.Decision
	afterRelease(q *dependencyQueueHead, entry *dependencyQueueEntry)
	tryPartialRelease(entry *dependencyQueueEntry) bool
	tryPartialAcquire(entry *dependencyQueueEntry, hasActiveCapacity bool) smachine.Decision
}

type dependencyStack struct {
	controller dependencyStackController
}

func (p *dependencyStack) contains(q *dependencyQueueHead, entry *dependencyQueueEntry) smachine.Decision {
	if p == nil || p.controller == nil {
		return smachine.Impossible
	}
	return p.controller.containsInStack(q, entry)
}

func (p *dependencyStack) updateAfterRelease(entry *dependencyQueueEntry, q *dependencyQueueHead) {
	if p == nil || p.controller == nil {
		return
	}
	p.controller.afterRelease(q, entry)
}

func (p *dependencyStack) tryPartialRelease(entry *dependencyQueueEntry) bool {
	if p == nil || p.controller == nil {
		return false
	}
	return p.controller.tryPartialRelease(entry)
}

func (p *dependencyStack) tryPartialAcquire(entry *dependencyQueueEntry, hasActiveCapacity bool) smachine.Decision {
	if p == nil || p.controller == nil {
		return smachine.NotPassed
	}
	return p.controller.tryPartialAcquire(entry, hasActiveCapacity)
}
