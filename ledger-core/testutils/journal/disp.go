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
	lock        sync.RWMutex
	subscribers []predicate.SubscriberFunc
	freeItems   []uint32
	stopped     bool
}

func (p *Dispenser) EventInput(event debuglogger.UpdateEvent) predicate.SubscriberState {
	if event.IsEmpty() {
		p.Stop()
		return predicate.RemoveSubscriber
	}

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

	p.lock.Lock()
	defer p.lock.Unlock()

	if p.stopped {
		outFn(debuglogger.UpdateEvent{})
		return
	}

	p.subscribers = append(p.subscribers, outFn)
}

func (p *Dispenser) Stop() {
	for _, subFn := range p._stop() {
		subFn(debuglogger.UpdateEvent{})
	}
}

func (p *Dispenser) _stop() []predicate.SubscriberFunc {
	p.lock.Lock()
	defer p.lock.Unlock()

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
	p.lock.Lock()
	defer p.lock.Unlock()
	p.subscribers[i] = nil
	p.freeItems = append(p.freeItems, uint32(i))
}

func (p *Dispenser) getSubscribers() []predicate.SubscriberFunc {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.subscribers // to avoid cascades of locks
}

func (p *Dispenser) Merge(other *Dispenser) {
	if other == nil {
		return
	}

	p.lock.Lock()
	if p.stopped {
		p.lock.Unlock()

		other.Stop()
		return
	}
	defer p.lock.Unlock()

	other.lock.Lock()
	defer other.lock.Unlock()

	if other.stopped {
		return
	}
	other.stopped = true

	p.subscribers = append(p.subscribers, other.subscribers...)

	other.subscribers = nil
	other.freeItems = nil
}
