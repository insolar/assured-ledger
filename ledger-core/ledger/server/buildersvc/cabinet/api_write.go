// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cabinet

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

type BundleResultFunc = func([]ledger.DirectoryIndex, error) bool

type DropWriter interface {
	WriteBundle([]WriteBundleEntry, BundleResultFunc) error
	WaitWriteBundles(synckit.SignalChannel) bool
}

type EntryWriterFunc = func (ledger.DirectoryIndex, []ledger.StorageLocator) MarshalerTo

type WriteBundleEntry struct {
	EntryKey  reference.Holder
	EntryFn   EntryWriterFunc
	Payloads  []SectionPayload
	Directory ledger.SectionID
}

type SectionPayload struct {
	Payload   MarshalerTo
	Extension ledger.ExtensionID
	Section   ledger.SectionID
}

type MarshalerTo interface {
	ProtoSize() int
	MarshalTo([]byte) (int, error)
}
