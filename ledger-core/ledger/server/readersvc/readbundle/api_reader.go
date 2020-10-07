// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package readbundle

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type ReadSlice interface {
	longbits.FixedReader
}

type Reader interface {
	FindDirectoryEntry(ledger.SectionID, reference.Holder) (ledger.StorageLocator, error)

	// GetDirectoryEntry returns key and entry's locator for the given index. Invalid index will return (nil, 0)
	GetDirectoryEntry(ledger.DirectoryIndex) (ledger.StorageLocator, error)
	// GetEntryStorage returns start of byte slice for the given locator.
	GetEntryStorage(ledger.StorageLocator) (ReadSlice, error)
	// GetPayloadStorage returns start of byte slice for the given locator.
	GetPayloadStorage(ledger.StorageLocator, int) (ReadSlice, error)

	// // GetDirectoryEntries returns all known entries.
	// GetDirectoryEntries(ledger.SectionID) [][]DirectoryEntry
}

