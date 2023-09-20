package memstor

import (
	"sync"

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
	snapshotEntries map[pulse.Number]*Snapshot
	last            *Snapshot
}

func (m *MemoryStorage) Last() *Snapshot {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.last
}

// Truncate deletes all entries except Count
func (m *MemoryStorage) truncate(count int) {
	switch {
	case len(m.entries) <= count:
		return
	case count == 0:
		m.last = nil
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

func (m *MemoryStorage) Append(snapshot *Snapshot) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	pn := snapshot.GetPulseNumber()

	if m.snapshotEntries == nil {
		m.snapshotEntries = make(map[pulse.Number]*Snapshot)
	}

	if _, ok := m.snapshotEntries[pn]; !ok {
		m.entries = append(m.entries, pn)
	}
	m.snapshotEntries[pn] = snapshot
	m.last = snapshot

	m.truncate(m.limit)

	return nil
}

func (m *MemoryStorage) ForPulseNumber(pulse pulse.Number) (*Snapshot, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if s, ok := m.snapshotEntries[pulse]; ok {
		return s, nil
	}
	return nil, ErrNotFound
}
