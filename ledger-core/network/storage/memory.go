// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package storage

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/network/node"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

const entriesCount = 10

// NewMemoryStorage constructor creates MemoryStorage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		limit: entriesCount,
	}
}

type MemoryStorage struct {
	lock            sync.RWMutex
	limit           int
	entries         []pulse.Number
	snapshotEntries map[pulse.Number]*node.Snapshot
}

// Truncate deletes all entries except Count
func (m *MemoryStorage) Truncate(count int) {
	switch {
	case len(m.entries) <= count:
		return
	case count == 0:
		m.snapshotEntries = nil
		m.entries = nil
		return
	}

	removeN := len(m.entries)-count
	for _, pn := range m.entries[:removeN] {
		delete(m.snapshotEntries, pn)
	}
	copy(m.entries, m.entries[removeN:])
	m.entries = m.entries[:count]
}

func (m *MemoryStorage) Append(snapshot *node.Snapshot) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	pn := snapshot.GetPulse()

	if m.snapshotEntries == nil {
		m.snapshotEntries = make(map[pulse.Number]*node.Snapshot)
	}

	if _, ok := m.snapshotEntries[pn]; !ok {
		m.entries = append(m.entries, pn)
	}
	m.snapshotEntries[pn] = snapshot

	m.Truncate(m.limit)

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
