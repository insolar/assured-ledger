package lmnapp

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/memstor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type addStorageFunc func (pn pulse.Number, rd readbundle.Reader) error

func newStorageBuilderFactory(pageSize int, addFn addStorageFunc) storageFactory {
	switch {
	case addFn == nil:
		panic(throw.IllegalValue())
	case pageSize < memstor.MinStoragePageSize:
		panic(throw.IllegalValue())
	}
	return storageFactory{pageSize, addFn}
}

type storageFactory struct {
	pageSize int
	addFn addStorageFunc
}

func (v storageFactory) CreateSnapshotWriter(pn pulse.Number, maxSection ledger.SectionID) bundle.SnapshotWriter {
	switch {
	case maxSection < ledger.DefaultDataSection:
		panic(throw.IllegalValue())
	case maxSection > ledger.MaxSectionID:
		panic(throw.IllegalValue())
	}
	return memstor.NewMemoryStorageWriter(pn, maxSection, v.pageSize)
}

func (v storageFactory) DepositReadOnlyWriter(w bundle.SnapshotWriter) error {
	msw, ok := w.(*memstor.MemoryStorageWriter)
	switch {
	case !ok:
		return throw.Unsupported()
	case !msw.IsReadOnly():
		return throw.IllegalState()
	}

	r := memstor.NewMemoryStorageReaderFromWriter(msw)
	return v.addFn(msw.PulseNumber(), r)
}
