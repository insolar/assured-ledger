package memstor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type DirectoryMap interface {
	FindLocator(reference.Holder) ledger.StorageLocator
	FindOrdinal(reference.Holder) ledger.Ordinal
	GetByOrdinal(ledger.Ordinal) ledger.StorageLocator
}

func newInMemoryDirectoryMap(directoryEntries [][]bundle.DirectoryEntry) DirectoryMap {
	n := len(directoryEntries)
	switch {
	case n > 1:
		n --
		n = n * len(directoryEntries[0]) + len(directoryEntries[n])
	case n == 1:
		n = len(directoryEntries[0])
	}

	if n == 0 {
		return directoryMap{}
	}

	dm := directoryMap{
		entries: make([]ledger.StorageLocator, 0, n),
		entryMap: make(map[reference.Global]ledger.Ordinal, n - 1), // zero index is always empty
	}

	ord := ledger.Ordinal(0)
	for _, pg := range directoryEntries {
		for _, entry := range pg {
			dm.entries = append(dm.entries, entry.Loc)
			if entry.Loc != 0 && !entry.Key.IsEmpty() {
				dm.entryMap[entry.Key] = ord
			}
			ord++
		}
	}

	return dm
}

type directoryMap struct {
	entries  []ledger.StorageLocator
	entryMap map[reference.Global]ledger.Ordinal
}

func (v directoryMap) FindLocator(ref reference.Holder) ledger.StorageLocator {
	return v.GetByOrdinal(v.FindOrdinal(ref))
}

func (v directoryMap) FindOrdinal(ref reference.Holder) ledger.Ordinal {
	if reference.IsEmpty(ref) {
		return 0
	}
	return v.entryMap[reference.Copy(ref)]
}

func (v directoryMap) GetByOrdinal(ord ledger.Ordinal) ledger.StorageLocator {
	if int(ord) >= len(v.entries) {
		return 0
	}
	return v.entries[ord]
}
