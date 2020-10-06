// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type DirectoryMap interface {
	Find(reference.Holder) ledger.StorageLocator
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
		entryMap: make(map[reference.Global]ledger.StorageLocator, n - 1), // zero index is always empty
	}

	for _, pg := range directoryEntries {
		for _, entry := range pg {
			dm.entries = append(dm.entries, entry.Loc)
			if entry.Loc != 0 && !entry.Key.IsEmpty() {
				dm.entryMap[entry.Key] = entry.Loc
			}
		}
	}

	return dm
}

type directoryMap struct {
	entries  []ledger.StorageLocator
	entryMap map[reference.Global]ledger.StorageLocator
}

func (v directoryMap) Find(ref reference.Holder) ledger.StorageLocator {
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
