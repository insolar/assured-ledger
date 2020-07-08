// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cabinet

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type BundleCompletedFunc = func([]catalog.DirectoryIndex)

type DropWriter interface {
	WriteBundle(entries []WriteBundleEntry, completedFn BundleCompletedFunc)
}

type EntryWriterFunc = func (catalog.DirectoryIndex, []catalog.StorageLocator) MarshalerTo

type WriteBundleEntry struct {
	Directory catalog.SectionID
	EntryKey  reference.Holder
	Entry     EntryWriterFunc
	Payloads  []SectionPayload
}

type SectionPayload struct {
	Section catalog.SectionID
	Payload MarshalerTo
}

type MarshalerTo interface {
	ProtoSize() int
	MarshalTo([]byte) (int, error)
}
