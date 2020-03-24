// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package vermap

import (
	"sync"
	"sync/atomic"
)

var _ LiveMap = &IncrementalMap{}

type IncrementalMap struct {
	updateVersion uint64 // atomic
	commitVersion uint64 // atomic
	options       Options

	entries sync.Map // map[Key]txEntry
}

func (m *IncrementalMap) getVersion() uint64 {
	return atomic.LoadUint64(&m.commitVersion)
}

func (m *IncrementalMap) ViewNow() ReadMap {
	return &Tx{container: m, viewVersion: m.getVersion(), txMark: &txMark{}, options: m.options}
}

func (m *IncrementalMap) StartUpdate() TxMap {
	return &Tx{container: m, viewVersion: m.getVersion(), txMark: &txMark{}, options: m.options &^ UpdateMask}
}

func (m *IncrementalMap) validateEntry(kv Entry) error {
	if err := m.validateKey(kv.Key); err != nil {
		return err
	}
	return nil
}

func (m *IncrementalMap) validateKey(k Key) error {
	if k.IsEmpty() {
		return ErrEmptyKey
	}
	return nil
}

func (m *IncrementalMap) getAsOf(k Key, version uint64) (Entry, error) {
	if vi, ok := m.entries.Load(k); ok {
		if kv := vi.(txEntry); kv.tx.getVersionNil() <= version {
			return kv.Entry, nil
		}
	}
	return Entry{}, ErrKeyNotFound
}

func (m *IncrementalMap) markAsOf(k Key, version uint64) (*txMark, bool) {
	if vi, ok := m.entries.Load(k); ok {
		if kv := vi.(txEntry); kv.tx.getVersionNil() <= version {
			return kv.tx, true
		}
	}
	return nil, false
}

func (m *IncrementalMap) allocateNextVersion() uint64 {
	for {
		switch nextVersion := atomic.LoadUint64(&m.updateVersion) + 1; {
		case nextVersion == pendingTxVersion:
			panic("version overflow")
		case atomic.CompareAndSwapUint64(&m.updateVersion, nextVersion-1, nextVersion):
			return nextVersion
		}
	}
}

func (m *IncrementalMap) commitTx(t *Tx) error {
	nextVersion := uint64(0)

	if ok, err := func() (bool, error) {
		t.mutex.Lock()
		defer t.mutex.Unlock()

		switch {
		case len(t.pending) == 0:
			return false, nil
		case t.isPending():
			nextVersion = m.allocateNextVersion()
			if t.txMark.setVersion(nextVersion) {
				return true, nil
			}
			// TODO set next version to avoid deadlock
		}
		return false, ErrDiscardedTxn
	}(); !ok || err != nil {
		return err
	}

	for k, v := range t.pending {
		if _, loaded := m.entries.LoadOrStore(k, v); loaded {
			// collision
		}
	}

	for !atomic.CompareAndSwapUint64(&m.updateVersion, nextVersion-1, nextVersion) {
	}

	return nil
}
