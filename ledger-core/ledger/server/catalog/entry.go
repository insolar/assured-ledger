// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package catalog

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type Entry = rms.CatalogEntry
type Ordinal uint32
type ExtensionID uint32

const (
	SameAsBodyExtensionID ExtensionID = 0
)

type SectionEntry struct {
	ExtensionID
	Data []byte
}

type WriteEntry struct {
	Entry
	Body       SectionEntry
	Payload    SectionEntry
	Extensions []SectionEntry
}

type WriteBundle struct {
	Entries    []WriteEntry
	CallbackFn func(WrittenBundle)
}

type WrittenEntry struct {
	EntryLoc   StorageLocator
	BodyLoc    StorageLocator
	PayloadLoc StorageLocator
	ExtLocs    []StorageLocator
}

type WrittenBundle struct {
	First   Ordinal
	Entries []WrittenEntry
}

