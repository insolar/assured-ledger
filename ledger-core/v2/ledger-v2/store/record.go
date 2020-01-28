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
