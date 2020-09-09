// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bundle

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

type ResultFunc = func([]ledger.DirectoryIndex, error) bool

type Writer interface {
	// WriteBundle verifies the bundle and starts writing of it. Can return after validation, but before completion of write.
	// All bundle entries must be written atomically.
	// Callback (ResultFunc) is invoked either on error, or after completion of write of all copies, but before commit.
	// Rollback is initiated either on an error or by returning (false) from (ResultFunc).
	// Implementation MUST ensure on successful writes the same sequence of calls to (ResultFunc) as were for WriteBundle.
	WriteBundle(Writeable, ResultFunc) error

	// WaitWriteBundles will wait for all bundles invoked before WaitWriteBundles to be completed.
	// Arg (synckit.SignalChannel) can be used to interrupt waiting.
	WaitWriteBundles(cancel synckit.SignalChannel, fn func(cancelled bool)) bool

	WaitWriteBundlesAsync(cancel synckit.SignalChannel, fn func(cancelled bool))

	// MarkReadOnly applies read only status to the underlying storage. It may affect bundles being written.
	MarkReadOnly() error
}

type Writeable interface {
	PrepareWrite(Snapshot) error
	ApplyWrite() ([]ledger.DirectoryIndex, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle.Writeable -s _mock.go -g
