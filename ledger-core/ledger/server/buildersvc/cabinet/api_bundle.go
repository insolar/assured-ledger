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

type BundleWriter interface {
	// WriteBundle verifies the bundle and starts writing of it. Can return after validation, but before completion of write.
	// All bundle entries must be written atomically.
	// Callback (BundleResultFunc) is invoked either on error, or after completion of write of all copies, but before commit.
	// Rollback is initiated either on an error or by returning (false) from (BundleResultFunc).
	// Implementation MUST ensure on successful writes the same sequence of calls to (BundleResultFunc) as were for WriteBundle.
	WriteBundle([]WriteBundleEntry, BundleResultFunc) error

	// WaitWriteBundles will wait for all bundles invoked before WaitWriteBundles to be completed.
	// Arg (synckit.SignalChannel) can be used to interrupt waiting.
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
