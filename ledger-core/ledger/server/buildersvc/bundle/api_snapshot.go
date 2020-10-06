// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bundle

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

// SnapshotWriter is an interfaces to support a storage with simplified transactional paradigm for bundled writer.
// Writer can write data opportunistically and has to rollback transactions in the same sequence as bundles were written.
// Concurrency control for SnapshotWriter MUST be done by implementation of BundleWriter.
type SnapshotWriter interface {
	// TakeSnapshot starts a new write operation and remembers a rollback point.
	// Not concurrent at SnapshotWriter, but can be called before commit of previous snapshot(s).
	TakeSnapshot() (Snapshot, error)
	MarkReadOnly() error

	DirtyReader() DirtyReader
}

// Snapshot represents a written bundle / transaction.
type Snapshot interface {
	// Prepared indicates that all allocations are made, but data were not written yet. Called once per Snapshot.
	// Implementation should NOT delay return from Prepared().
	// It is guaranteed that there will be no new TakeSnapshot() call until Prepared() is invoked on the last one.
	// Not concurrent with SnapshotWriter.TakeSnapshot().
	Prepared() error
	// Completed indicates that all data were written into the allocations made. Called once per Snapshot after Prepared().
	// Implementation can delay return from Completed() e.g. to do multiple copies of data.
	// Concurrent.
	Completed() error
	// Commit is invoked after Completed().
	// Is guaranteed to be invoked in the same sequence as snapshots were taken by TakeSnapshot().
	// Not concurrent with SnapshotWriter.TakeSnapshot().
	Commit() error
	// Rollback is invoked on any errors, including errors of Commit().
	// Is guaranteed to be invoked in the same sequence as snapshots were taken by TakeSnapshot().
	// Arg (chained) indicates that rollback was initiated by an earlier snapshot.
	// Not concurrent with SnapshotWriter.TakeSnapshot().
	// WARNING! Rollback's panic will be ignored.
	Rollback(chained bool) error

	// GetPayloadSection returns PayloadSection for the given id or error.
	// Happens between TakeSnapshot() and Prepared().
	// Not concurrent with SnapshotWriter.TakeSnapshot().
	GetPayloadSection(ledger.SectionID) (PayloadSection, error)

	// GetDirectorySection returns DirectorySection for the given id or error
	// Happens between TakeSnapshot() and Prepared().
	// Not concurrent with SnapshotWriter.TakeSnapshot().
	GetDirectorySection(ledger.SectionID) (DirectorySection, error)
}

type PayloadReceptacle interface {
	io.WriterTo
	ApplyMarshalTo(MarshalerTo) error
	ApplyFixedReader(longbits.FixedReader) error
}

// MarshalerTo is interface to write data into storage
// Implementation may also use io.WriterTo behavior.
type MarshalerTo interface {
	// optional io.WriterTo
	ProtoSize() int
	MarshalTo([]byte) (int, error)
}

type PayloadSection interface {
	// AllocatePayloadStorage is called to allocate a storage area for any payload of a record.
	AllocatePayloadStorage(size int, extID ledger.ExtensionID) (PayloadReceptacle, ledger.StorageLocator, error)
}

type DirectoryEntry struct {
	Key   reference.Global
	Loc   ledger.StorageLocator
	// Fil is a temporary field and it is not preserved
	Fil   FilamentInfo
}

type FilamentInfo struct {
	Link  ledger.Ordinal
	JetID jet.ID
	Flags ledger.DirectoryEntryFlags
}

func (v DirectoryEntry) IsZero() bool {
	return v.Loc == 0
}

type DirectorySection interface {
	// GetNextDirectoryIndex provides an index for the next to-be-added directory entry.
	GetNextDirectoryIndex() ledger.DirectoryIndex
	// AppendDirectoryEntry adds a new item to the directory at the given index.
	AppendDirectoryEntry(index ledger.DirectoryIndex, entry DirectoryEntry) error

	// AllocateEntryStorage MUST be called as the last per record and MUST guarantee that the allocated storage is
	// located after any other data of the same record within the same section.
	AllocateEntryStorage(size int) (PayloadReceptacle, ledger.StorageLocator, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle.SnapshotWriter -s _mock.go -g
//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle.Snapshot -s _mock.go -g
//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle.DirectorySection -s _mock.go -g
//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle.PayloadSection -s _mock.go -g
//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle.PayloadReceptacle -s _mock.go -g
