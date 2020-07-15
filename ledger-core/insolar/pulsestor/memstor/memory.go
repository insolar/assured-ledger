// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memstor

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

var _ appctl.Accessor = &StorageMem{}

// StorageMem is a memory storage implementation. It saves pulses to memory and allows removal.
type StorageMem struct {
	lock    sync.RWMutex
	storage map[pulse.Number]*memNode
	head    *memNode
	tail    *memNode
}

type memNode struct {
	pulse      appctl.PulseChange
	prev, next *memNode
}

// NewStorageMem creates new memory storage instance.
func NewStorageMem() *StorageMem {
	return &StorageMem{
		storage: make(map[pulse.Number]*memNode),
	}
}

// ForPulseNumber returns pulse for provided Pulse number. If not found, ErrNotFound will be returned.
func (s *StorageMem) ForPulseNumber(ctx context.Context, pn pulse.Number) (appctl.PulseChange, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if node, ok := s.storage[pn]; ok {
		return node.pulse, nil
	}
	return appctl.PulseChange{}, pulsestor.ErrNotFound
}

// Latest returns a latest pulse saved in memory. If not found, ErrNotFound will be returned.
func (s *StorageMem) Latest(ctx context.Context) (pulse appctl.PulseChange, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tail == nil {
		err = pulsestor.ErrNotFound
		return
	}

	return s.tail.pulse, nil
}

// Append appends provided a pulse to current storage. Pulse number should be greater than currently saved for preserving
// pulse consistency. If provided Pulse does not meet the requirements, ErrBadPulse will be returned.
func (s *StorageMem) Append(ctx context.Context, pulse appctl.PulseChange) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	var appendTail = func() {
		oldTail := s.tail
		newTail := &memNode{
			prev:  oldTail,
			pulse: pulse,
		}
		oldTail.next = newTail
		newTail.prev = oldTail
		s.storage[newTail.pulse.PulseNumber] = newTail
		s.tail = newTail
	}
	var appendHead = func() {
		s.tail = &memNode{
			pulse: pulse,
		}
		s.storage[pulse.PulseNumber] = s.tail
		s.head = s.tail
	}

	if s.head == nil {
		appendHead()
		return nil
	}

	if pulse.PulseNumber <= s.tail.pulse.PulseNumber {
		return pulsestor.ErrBadPulse
	}
	appendTail()

	return nil
}

// Shift removes oldest pulse from storage. If the storage is empty, an error will be returned.
func (s *StorageMem) Shift(ctx context.Context, pn pulse.Number) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.head == nil {
		err = throw.New("nothing to shift")
		return
	}

	h := s.head
	for h != nil && h.pulse.PulseNumber <= pn {
		delete(s.storage, h.pulse.PulseNumber)
		h = h.next
	}

	s.head = h
	if s.head == nil {
		s.tail = nil
	} else {
		s.head.prev = nil
	}

	return nil
}

// Forwards calculates steps pulses forwards from provided Pulse. If calculated pulse does not exist, ErrNotFound will
// be returned.
func (s *StorageMem) Forwards(ctx context.Context, pn pulse.Number, steps int) (pulse appctl.PulseChange, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	node, ok := s.storage[pn]
	if !ok {
		err = pulsestor.ErrNotFound
		return
	}

	iterator := node
	for i := 0; i < steps; i++ {
		if iterator.next == nil {
			err = pulsestor.ErrNotFound
			return
		}
		iterator = iterator.next
	}

	return iterator.pulse, nil
}

// Backwards calculates steps pulses backwards from provided pulse. If calculated pulse does not exist, ErrNotFound will
// be returned.
func (s *StorageMem) Backwards(ctx context.Context, pn pulse.Number, steps int) (pulse appctl.PulseChange, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	node, ok := s.storage[pn]
	if !ok {
		err = pulsestor.ErrNotFound
		return
	}

	iterator := node
	for i := 0; i < steps; i++ {
		if iterator.prev == nil {
			err = pulsestor.ErrNotFound
			return
		}
		iterator = iterator.prev
	}

	return iterator.pulse, nil
}
