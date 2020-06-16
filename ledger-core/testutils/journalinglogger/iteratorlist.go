// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journalinglogger

import (
	"sync"
)

type JournalIteratorList struct {
	lock   sync.RWMutex
	lastID int
	list   map[int]*JournalIterator
}

func (l *JournalIteratorList) nextID() int {
	l.lastID++
	return l.lastID
}

func (l *JournalIteratorList) createIterator(logger *JournalingLogger) *JournalIterator {
	l.lock.Lock()
	defer l.lock.Unlock()

	id := l.nextID()

	it := &JournalIterator{
		id:       id,
		await:    make(chan struct{}, 1),
		position: 0,
		logger:   logger,
	}

	l.list[id] = it

	return it
}

func (l *JournalIteratorList) Finish(it *JournalIterator) {
	l.lock.Lock()
	defer l.lock.Unlock()

	delete(l.list, it.id)
}

func (l *JournalIteratorList) Stop() {
	for _, it := range l.list {
		it.Stop()
	}
}

func (l *JournalIteratorList) Notify() {
	l.lock.RLock()
	defer l.lock.RUnlock()

	for _, it := range l.list {
		select {
		case it.await <- struct{}{}:
		default:
		}
	}
}
