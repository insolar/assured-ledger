// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cabinet

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type BundleCompletedFunc = func([]ledger.DirectoryIndex) bool

type DropWriter interface {
	WriteBundle(entries []WriteBundleEntry, completedFn BundleCompletedFunc)
}

type EntryWriterFunc = func (ledger.DirectoryIndex, []ledger.StorageLocator) MarshalerTo

type WriteBundleEntry struct {
	EntryKey  reference.Holder
	EntryFn   EntryWriterFunc
	Payloads  []SectionPayload
	Directory ledger.SectionID
}

type SectionPayload struct {
	Payload MarshalerTo
	Extension ledger.ExtensionID
	Section ledger.SectionID
}

type MarshalerTo interface {
	ProtoSize() int
	MarshalTo([]byte) (int, error)
}
