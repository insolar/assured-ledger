// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journal

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/journal/predicate"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SubscriberFunc = func(debuglogger.UpdateEvent)

type Dispenser struct {
	lock sync.RWMutex
	subscribers []SubscriberFunc
	predicates  []predicate.Func
	signals     []synckit.ClosableSignalChannel
	freeItems   []uint32
	stopped     bool
}

func (p *Dispenser) SubscribeAll(outFn SubscriberFunc)  {
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

func (p *Dispenser) SubscribeFilter(predicateFn func(debuglogger.UpdateEvent) bool, outFn SubscriberFunc)  {
	switch {
	case outFn == nil:
		panic(throw.IllegalValue())
	case predicateFn == nil:
		p.SubscribeAll(outFn)
	default:
		p.SubscribeAll(func(event debuglogger.UpdateEvent) {
			if predicateFn(event) || event.IsEmpty() {
				outFn(event)
			}
		})
	}
}

func (p *Dispenser) SubscribeFirst(predicateFn func(debuglogger.UpdateEvent) bool, chanLimit int) synckit.SignalChannel {
	if chanLimit <= 0 {
		chanLimit = 1
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	signal := make(synckit.ClosableSignalChannel, chanLimit)

	if n := len(p.freeItems); n > 0 {
		n--
		freeItem := p.freeItems[n]
		p.freeItems = p.freeItems[:n]

		p.predicates[freeItem] = predicateFn
		p.signals[freeItem] = signal
		return signal
	}

	p.predicates = append(p.predicates, predicateFn)
	p.signals = append(p.signals, signal)
	return signal
}

func (p *Dispenser) EventInput(event debuglogger.UpdateEvent) {
	if event.IsEmpty() {
		p.Stop()
		return
	}

	for _, subFn := range p.process(event) {
		subFn(event)
	}
}

func (p *Dispenser) process(event debuglogger.UpdateEvent) []SubscriberFunc {
	p.lock.Lock()
	defer p.lock.Unlock()

	for i, predicateFn := range p.predicates {
		if predicateFn(event) {
			signal := p.signals[i]

			p.signals[i] = nil
			p.predicates[i] = nil
			p.freeItems = append(p.freeItems, uint32(i))

			select {
			case signal <- struct {}{}:
			default:
			}
		}
	}

	return p.subscribers // to avoid cascades of locks
}

func (p *Dispenser) Stop() {

	for _, subFn := range p._stop() {
		subFn(debuglogger.UpdateEvent{})
	}
}

func (p *Dispenser) _stop() []SubscriberFunc {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.stopped {
		return nil
	}
	p.stopped = true

	for i, predicateFn := range p.predicates {
		predicateFn(debuglogger.UpdateEvent{})
		close(p.signals[i])
	}
	p.signals = nil
	p.predicates = nil

	subscribers := p.subscribers
	p.subscribers = nil
	p.freeItems = nil

	return subscribers // to avoid cascades of locks
}

func (p *Dispenser) Merge(dispenser *Dispenser) {
	if dispenser == nil {
		return
	}

	p.lock.Lock()
	if p.stopped {
		p.lock.Unlock()

		dispenser.Stop()
		return
	}
	defer p.lock.Unlock()

	dispenser.lock.Lock()
	defer dispenser.lock.Unlock()

	if dispenser.stopped {
		return
	}
	dispenser.stopped = true

	p.subscribers = append(p.subscribers, dispenser.subscribers...)

	for i, pFn := range dispenser.predicates {
		if pFn == nil {
			continue
		}
		p.predicates = append(p.predicates, pFn)
		p.signals = append(p.signals, dispenser.signals[i])
	}

	dispenser.subscribers = nil
	dispenser.predicates = nil
	dispenser.signals = nil
	dispenser.freeItems = nil
}
