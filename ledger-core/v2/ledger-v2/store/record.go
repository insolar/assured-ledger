// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package store

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
)

type Record struct {
	ID     insolar.ID
	Record record.Record
}

type RecordStore struct {
	lock    sync.Mutex
	storage map[insolar.ID]record.Record
}

func NewRecordStore() *RecordStore {
	return &RecordStore{
		storage: map[insolar.ID]record.Record{},
	}
}

func (s *RecordStore) Store(recs []*Record) {
	s.lock.Lock()
	for _, r := range recs {
		s.storage[r.ID] = r.Record
	}
	s.lock.Unlock()
}
