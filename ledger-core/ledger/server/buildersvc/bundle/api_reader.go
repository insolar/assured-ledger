// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bundle

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
)

// DirtyReader provides dirty read access to SnapshotWriter.
// Use with EXTREME caution.
type DirtyReader interface {
	// GetDirectoryEntry returns key and entry's locator for the given index. Invalid index will return (nil, 0)
	// RACE! Caller MUST ensure that the data to be read was fully written.
	GetDirectoryEntry(ledger.DirectoryIndex) DirectoryEntry
	// GetEntryStorage returns start of byte slice for the given locator. Invalid locator will return nil.
	// RACE! Caller MUST ensure that the data to be read was fully written.
	// WARNING! Caller MUST NOT change the byte slice.
	GetEntryStorage(ledger.StorageLocator) []byte
	// GetPayloadStorage returns start of byte slice for the given locator. Invalid locator will return nil.
	// RACE! Caller MUST ensure that the data to be read was fully written.
	// WARNING! Caller MUST NOT change the byte slice.
	// WARNING! Implementation MAY return a byte slice LONGER than the actual content.
	GetPayloadStorage(ledger.StorageLocator) []byte

	// GetDirectoryEntries returns all known entries.
	// RACE! Caller MUST ensure that the data to be read was fully written.
	// WARNING! Caller MUST NOT change the slices.
	GetDirectoryEntries(ledger.SectionID) [][]DirectoryEntry
}
