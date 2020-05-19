// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package storage

import (
	"context"
	"sync"

	pulse2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

const entriesCount = 10

// NewMemoryStorage constructor creates MemoryStorage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		entries:         make([]pulse2.Pulse, 0),
		snapshotEntries: make(map[pulse.Number]*node.Snapshot),
	}
}

type MemoryStorage struct {
	lock            sync.RWMutex
	entries         []pulse2.Pulse
	snapshotEntries map[pulse.Number]*node.Snapshot
}

// truncate deletes all entries except Count
func (m *MemoryStorage) truncate(count int) {
	if len(m.entries) <= count {
		return
	}

	truncatePulses := m.entries[:len(m.entries)-count]
	m.entries = m.entries[len(truncatePulses):]
	for _, p := range truncatePulses {
		delete(m.snapshotEntries, p.PulseNumber)
	}
}

func (m *MemoryStorage) AppendPulse(ctx context.Context, pulse pulse2.Pulse) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.entries = append(m.entries, pulse)
	m.truncate(entriesCount)
	return nil
}

func (m *MemoryStorage) GetPulse(ctx context.Context, number pulse.Number) (pulse2.Pulse, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, p := range m.entries {
		if p.PulseNumber == number {
			return p, nil
		}
	}

	return *pulse2.GenesisPulse, ErrNotFound
}

func (m *MemoryStorage) GetLatestPulse(ctx context.Context) (pulse2.Pulse, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if len(m.entries) == 0 {
		return *pulse2.GenesisPulse, ErrNotFound
	}
	return m.entries[len(m.entries)-1], nil
}

func (m *MemoryStorage) Append(pulse pulse.Number, snapshot *node.Snapshot) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.snapshotEntries[pulse] = snapshot
	return nil
}

func (m *MemoryStorage) ForPulseNumber(pulse pulse.Number) (*node.Snapshot, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if s, ok := m.snapshotEntries[pulse]; ok {
		return s, nil
	}
	return nil, ErrNotFound
}
