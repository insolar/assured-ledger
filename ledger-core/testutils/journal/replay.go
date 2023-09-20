package journal

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewReplay(parent *Dispenser, size int) *Replay {
	switch {
	case parent == nil:
		panic(throw.IllegalValue())
	case size <= 0:
		panic(throw.IllegalValue())
	}
	return &Replay{
		parent: parent,
		events: make([]debuglogger.UpdateEvent, 0, size),
	}
}

type Replay struct {
	parent    *Dispenser
	lock      sync.RWMutex
	events    []debuglogger.UpdateEvent
	overflown bool
	discard   bool
}

func (p *Replay) EventInput(event debuglogger.UpdateEvent) predicate.SubscriberState {
	p.lock.Lock()
	defer p.lock.Unlock()

	switch {
	case p.overflown:
		return predicate.RemoveSubscriber
	case len(p.events) == cap(p.events):
		p.overflown = true
		if p.discard {
			p.events = nil
		}
		return predicate.RemoveSubscriber
	}
	p.events = append(p.events, event)
	return predicate.RetainSubscriber
}

func (p *Replay) SetDiscardOnOverflow() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.overflown {
		p.events = nil
		return
	}
	p.discard = true
}

func (p *Replay) DiscardNow() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.overflown = true
	p.discard = true
	p.events = nil
}

func (p *Replay) getEventsOrSubscribe(lastLen int, outFn predicate.SubscriberFunc) ([]debuglogger.UpdateEvent, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	switch {
	case lastLen < len(p.events):
		return p.events[lastLen:], p.overflown
	case p.overflown:
		return nil, true
	default:
		p.parent.Subscribe(outFn)
		return nil, false
	}
}

func (p *Replay) ReplayAndSubscribe(outFn predicate.SubscriberFunc) {
	if outFn == nil {
		panic(throw.IllegalValue())
	}

	lastPos := 0
	for {
		chunk, overflown := p.getEventsOrSubscribe(lastPos, outFn)
		switch {
		case overflown:
			panic(throw.IllegalState())
		case len(chunk) == 0:
			return
		}
		for _, event := range chunk {
			if outFn(event) == predicate.RemoveSubscriber {
				return
			}
		}

		lastPos += len(chunk)
	}
}

func (p *Replay) TryReplayThenSubscribe(outFn predicate.SubscriberFunc) bool {
	if outFn == nil {
		panic(throw.IllegalValue())
	}

	lastPos := 0
	for {
		chunk, overflown := p.getEventsOrSubscribe(lastPos, outFn)
		for _, event := range chunk {
			if outFn(event) == predicate.RemoveSubscriber {
				return true
			}
		}

		switch {
		case overflown:
			p.parent.Subscribe(outFn)
			return false
		case len(chunk) == 0:
			return true
		}

		lastPos += len(chunk)
	}
}

func (p *Replay) Parent() *Dispenser {
	return p.parent
}

func (p *Replay) Count() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.events)
}

func (p *Replay) State() (count int, isOverflown bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.events), p.overflown
}

func (p *Replay) Get(i int) debuglogger.UpdateEvent {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if i < len(p.events) {
		return p.events[i]
	}
	return debuglogger.UpdateEvent{}
}
