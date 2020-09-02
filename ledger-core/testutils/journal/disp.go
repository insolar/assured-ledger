// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journal

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Dispenser struct {
	changeLock  sync.RWMutex
	eventLock   sync.RWMutex

	subscribers []predicate.SubscriberFunc
	freeItems   []uint32
	stopped     bool
}

func (p *Dispenser) EventInput(event debuglogger.UpdateEvent) predicate.SubscriberState {
	if event.IsEmpty() {
		p.Stop()
		return predicate.RemoveSubscriber
	}

	p.eventLock.RLock()
	defer p.eventLock.RUnlock()

	for i, subFn := range p.getSubscribers() {
		switch {
		case subFn == nil:
			continue
		case subFn(event) != predicate.RetainSubscriber:
			p.removeSubscriber(i)
		}
	}
	return predicate.RetainSubscriber
}

func (p *Dispenser) Subscribe(outFn predicate.SubscriberFunc) {
	if outFn == nil {
		panic(throw.IllegalValue())
	}

	p.changeLock.Lock()
	defer p.changeLock.Unlock()

	if p.stopped {
		outFn(debuglogger.UpdateEvent{})
		return
	}

	p.subscribers = append(p.subscribers, outFn)
}

// EnsureSubscription should be used after Subscribe to guarantee that there is no unprocessed events.
// LOCK! Will cause deadlock when invoked inside an event handler.
func (p *Dispenser) EnsureSubscription() {
	p.eventLock.Lock()
	p.eventLock.Unlock()
}

func (p *Dispenser) Stop() {
	for _, subFn := range p._stop() {
		subFn(debuglogger.UpdateEvent{})
	}
}

func (p *Dispenser) _stop() []predicate.SubscriberFunc {
	p.changeLock.Lock()
	defer p.changeLock.Unlock()

	if p.stopped {
		return nil
	}
	p.stopped = true

	subscribers := p.subscribers
	p.subscribers = nil
	p.freeItems = nil

	return subscribers // to avoid cascades of locks
}

func (p *Dispenser) removeSubscriber(i int) {
	p.changeLock.Lock()
	defer p.changeLock.Unlock()
	p.subscribers[i] = nil
	p.freeItems = append(p.freeItems, uint32(i))
}

func (p *Dispenser) getSubscribers() []predicate.SubscriberFunc {
	p.changeLock.RLock()
	defer p.changeLock.RUnlock()
	return p.subscribers // to avoid cascades of locks
}

func (p *Dispenser) Merge(other *Dispenser) {
	if other == nil {
		return
	}

	p.changeLock.Lock()
	if p.stopped {
		p.changeLock.Unlock()

		other.Stop()
		return
	}
	defer p.changeLock.Unlock()

	other.changeLock.Lock()
	defer other.changeLock.Unlock()

	if other.stopped {
		return
	}
	other.stopped = true

	p.subscribers = append(p.subscribers, other.subscribers...)

	other.subscribers = nil
	other.freeItems = nil
}
