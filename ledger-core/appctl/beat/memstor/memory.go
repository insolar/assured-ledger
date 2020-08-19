// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memstor

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

var _ beat.History = &StorageMem{}

// StorageMem is a memory storage implementation. It saves pulses to memory and allows removal.
type StorageMem struct {
	lock    sync.RWMutex
	storage map[pulse.Number]*memNode
	head    *memNode
	tail    *memNode
}

type memNode struct {
	pulse      beat.Beat
	prev, next *memNode
}

// NewStorageMem creates new memory storage instance.
func NewStorageMem() *StorageMem {
	return &StorageMem{
		storage: make(map[pulse.Number]*memNode),
	}
}

// TimeBeat returns pulse for provided Pulse number. If not found, ErrNotFound will be returned.
func (s *StorageMem) TimeBeat(pn pulse.Number) (beat.Beat, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if node, ok := s.storage[pn]; ok {
		return node.pulse, nil
	}
	return beat.Beat{}, throw.WithStack(ErrNotFound)
}

// LatestTimeBeat returns a latest pulse saved in memory. If not found, ErrNotFound will be returned.
func (s *StorageMem) LatestTimeBeat() (beat.Beat, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tail != nil {
		return s.tail.pulse, nil
	}
	return beat.Beat{}, throw.WithStack(ErrNotFound)
}

func (s *StorageMem) EnsureLatestTimeBeat(pulse beat.Beat) error {
	switch latest, err := s.LatestTimeBeat(); {
	case err != nil:
		return err
	case pulse.Data != latest.Data:
		return throw.WithStack(ErrBadPulse)
	}
	return nil
}

func (s *StorageMem) AddExpectedBeat(beat.Beat) error {
	return nil
}

func (s *StorageMem) AddCommittedBeat(pulse beat.Beat) error {
	if !pulse.PulseEpoch.IsTimeEpoch() {
		panic(throw.IllegalValue())
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.head == nil {
		s.tail = &memNode{
			pulse: pulse,
		}
		s.storage[pulse.PulseNumber] = s.tail
		s.head = s.tail
		return nil
	}

	if pulse.PulseNumber <= s.tail.pulse.PulseNumber {
		return throw.WithStack(ErrBadPulse)
	}

	oldTail := s.tail
	newTail := &memNode{
		prev:  oldTail,
		pulse: pulse,
	}
	oldTail.next = newTail
	newTail.prev = oldTail
	s.storage[newTail.pulse.PulseNumber] = newTail
	s.tail = newTail

	return nil
}

// Trim removes oldest pulse from storage. If the storage is empty, an error will be returned.
func (s *StorageMem) Trim(pn pulse.Number) (err error) {
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

