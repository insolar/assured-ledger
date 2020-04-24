// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/store"
)

type DBStore struct {
	sync.RWMutex
	db store.DB
}

func NewDBStore(db store.DB) *DBStore {
	return &DBStore{db: db}
}

func (s *DBStore) All(ctx context.Context, pulse insolar.PulseNumber) []insolar.JetID {
	s.RLock()
	defer s.RUnlock()

	tree := s.get(pulse)
	return tree.LeafIDs()
}

func (s *DBStore) ForID(ctx context.Context, pulse insolar.PulseNumber, recordID insolar.ID) (insolar.JetID, bool) {
	s.RLock()
	defer s.RUnlock()

	tree := s.get(pulse)
	return tree.Find(recordID)
}

// TruncateHead remove all records after lastPulse
func (s *DBStore) TruncateHead(ctx context.Context, from insolar.PulseNumber) error {
	s.Lock()
	defer s.Unlock()

	it := s.db.NewIterator(pulseKey(from), false)
	defer it.Close()

	var hasKeys bool
	for it.Next() {
		hasKeys = true
		key := newPulseKey(it.Key())
		err := s.db.Delete(&key)
		if err != nil {
			return errors.Wrapf(err, "can't delete key: %+v", key)
		}

		inslogger.FromContext(ctx).Debugf("Erased key with pulse number: %s", insolar.PulseNumber(key))
	}

	if !hasKeys {
		inslogger.FromContext(ctx).Debugf("No records. Nothing done. Pulse number: %s", from.String())
	}

	return nil
}

func (s *DBStore) Update(ctx context.Context, pulse insolar.PulseNumber, actual bool, ids ...insolar.JetID) error {
	s.Lock()
	defer s.Unlock()

	tree := s.get(pulse)

	for _, id := range ids {
		tree.Update(id, actual)
	}
	err := s.set(pulse, tree)
	if err != nil {
		return errors.Wrapf(err, "failed to update jets")
	}
	return nil
}

func (s *DBStore) Split(ctx context.Context, pulse insolar.PulseNumber, id insolar.JetID) (insolar.JetID, insolar.JetID, error) {
	s.Lock()
	defer s.Unlock()

	tree := s.get(pulse)
	left, right, err := tree.Split(id)
	if err != nil {
		return insolar.ZeroJetID, insolar.ZeroJetID, err
	}
	err = s.set(pulse, tree)
	if err != nil {
		return insolar.ZeroJetID, insolar.ZeroJetID, err
	}
	return left, right, nil
}
func (s *DBStore) Clone(ctx context.Context, from, to insolar.PulseNumber, keepActual bool) error {
	s.Lock()
	defer s.Unlock()

	tree := s.get(from)
	newTree := tree.Clone(keepActual)
	err := s.set(to, newTree)
	if err != nil {
		return errors.Wrapf(err, "failed to clone jet.Tree")
	}
	return nil
}

type pulseKey insolar.PulseNumber

func (k pulseKey) Scope() store.Scope {
	return store.ScopeJetTree
}

func (k pulseKey) ID() []byte {
	return insolar.PulseNumber(k).Bytes()
}

func newPulseKey(raw []byte) pulseKey {
	key := pulseKey(insolar.NewPulseNumber(raw))
	return key
}

func (s *DBStore) get(pn insolar.PulseNumber) *Tree {
	serializedTree, err := s.db.Get(pulseKey(pn))
	if err != nil {
		return NewTree(pn == insolar.GenesisPulse.PulseNumber)
	}

	recovered := &Tree{}
	err = recovered.Unmarshal(serializedTree)
	if err != nil {
		return nil
	}
	return recovered
}

func (s *DBStore) set(pn insolar.PulseNumber, jt *Tree) error { // nolint: interfacer
	key := pulseKey(pn)

	serialized, err := jt.Marshal()
	if err != nil {
		return errors.Wrap(err, "failed to serialize jet.Tree")
	}

	return s.db.Set(key, serialized)
}
