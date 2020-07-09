// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/cabinet"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewCabinetWriter(cab StorageCabinetWriter) *CabinetWriter {
	if cab == nil {
		panic(throw.IllegalValue())
	}
	return &CabinetWriter{ cab: cab}
}

type Snapshot interface {
	// Prepared indicates that all allocations are made, but data were not written yet. Called once per Snapshot.
	// Implementation should NOT delay return from Prepared().
	// It is guaranteed that there will be no new TakeSnapshot() call until Prepared() is invoked on the last one.
	// Not concurrent with StorageCabinetWriter.TakeSnapshot().
	Prepared() error
	// Completed indicates that all data were written into the allocations made. Called once per Snapshot.
	// Implementation can delay return from Completed() e.g. to do multiple copies of data.
	// Concurrent with StorageCabinetWriter.TakeSnapshot().
	Completed() error
	// Commit is invoked after Completed().
	// Is guaranteed to be invoked in the same sequence as snapshots were made by TakeSnapshot().
	// Not concurrent with StorageCabinetWriter.TakeSnapshot().
	Commit() error
	// Rollback is invoked on any errors before Commit().
	// Not concurrent with StorageCabinetWriter.TakeSnapshot().
	Rollback()
	// ChainedRollback is invoked before Commit() when previous snapshot was rolled back.
	// Is guaranteed to be invoked in the same sequence as snapshots were made by TakeSnapshot().
	// Not concurrent with StorageCabinetWriter.TakeSnapshot().
	ChainedRollback()

	// GetPayloadSection returns PayloadSection for the given id or error.
	// Happens between TakeSnapshot() and Prepared().
	// Not concurrent with StorageCabinetWriter.TakeSnapshot().
	GetPayloadSection(ledger.SectionID) (PayloadSection, error)
	// GetDirectorySection returns DirectorySection for the given id or error
	// Happens between TakeSnapshot() and Prepared().
	// Not concurrent with StorageCabinetWriter.TakeSnapshot().
	GetDirectorySection(ledger.SectionID) (DirectorySection, error)
}

type StorageCabinetWriter interface {
	// TakeSnapshot starts a new write operation and remembers a rollback point.
	// Not concurrent at StorageCabinetWriter, but can be called before commit of previous snapshot(s).
	TakeSnapshot() Snapshot
}

type MarshalToReceptacle interface {
	ApplyMarshalTo(cabinet.MarshalerTo) error
}

type PayloadSection interface {
	AllocatePayloadStorage(size int, extID ledger.ExtensionID) (MarshalToReceptacle, ledger.StorageLocator, error)
}

type DirectorySection interface {
	GetNextDirectoryIndex() ledger.DirectoryIndex
	SetDirectoryEntry(index ledger.DirectoryIndex, key reference.Holder, loc ledger.StorageLocator) error
	AllocateEntryStorage(size int) (MarshalToReceptacle, ledger.StorageLocator, error)
}

var _ cabinet.DropWriter = &CabinetWriter{}

type CabinetWriter struct {
	writeMutex sync.Mutex
	cab        StorageCabinetWriter

	lastWriterReady chan struct{}
}

func (p *CabinetWriter) WaitWriteBundles(done synckit.SignalChannel) bool {
	p.writeMutex.Lock()
	last := p.lastWriterReady
	p.writeMutex.Unlock()
	if last == nil {
		return true
	}

	select {
	case <- last:
		return true
	case <- done:
		return false
	}
}

func (p *CabinetWriter) WriteBundle(entries []cabinet.WriteBundleEntry, completedFn cabinet.BundleResultFunc) error {
	if completedFn == nil {
		panic(throw.IllegalValue())
	}

	p.writeMutex.Lock()
	defer p.writeMutex.Unlock()

	snapshot := p.cab.TakeSnapshot()
	defer func() {
		if snapshot != nil {
			snapshot.Rollback()
		}
	}()

	preparedEntries, err := p._prepareBundle(snapshot, entries)
	if err != nil {
		return err
	}

	prev := p.lastWriterReady
	next := make(chan struct{}, 1)
	p.lastWriterReady = next

	select {
	case <- prev:
		// ignore prev if can read it - it is either ok, or was rolled back completely
		prev = nil
	default:
		// prev hasn't finished yet
	}

	go p.applyBundleSafe(snapshot, preparedEntries, prev, next, completedFn)

	snapshot = nil
	return nil
}

func (p *CabinetWriter) applyBundleSafe(snapshot Snapshot, entries []preparedEntry,
	prev, next synckit.ClosableSignalChannel, completedFn cabinet.BundleResultFunc,
) {
	defer close(next) // to be executed as the very last one

	chained := false
	locked  := false
	rollback := true

	defer func() {
		switch {
		case !rollback:
			if locked {
				p.writeMutex.Unlock()
			}
			return
		case !locked:
			p.writeMutex.Lock()
		}
		defer p.writeMutex.Unlock()

		if chained {
			snapshot.ChainedRollback()
		} else {
			snapshot.Rollback()
		}
	}()

	err := func() (err error) {
		defer func() {
			err = throw.RW(recover(), err, "applyBundle failed")
		}()

		if prev != nil {
			select {
			case _, ok := <-prev:
				if !ok {
					chained = true // rollback was made by a previous writer
					return throw.E("chained cancel")
				}
				prev = nil // previous writer is ready
			default:
			}
		}

		var assignments []ledger.DirectoryIndex
		assignments, err = p.applyBundle(entries)

		if err = snapshot.Completed(); err != nil {
			return err
		}

		if prev != nil {
			if _, ok := <-prev; !ok {
				chained = true // rollback was made by a previous writer
				return throw.E("chained cancel")
			}
		}

		p.writeMutex.Lock()
		locked = true

		if !completedFn(assignments, nil) {
			return nil // abort
		}

		rollback = false
		if err = snapshot.Commit(); err != nil {
			return err
		}

		p.writeMutex.Unlock()
		locked = false

		next <- struct{}{} // send ok to next
		return nil
	}()


	if err != nil {
		if !locked {
			p.writeMutex.Lock()
			locked = true
		}
		completedFn(nil, err)
	}
}

func (p *CabinetWriter) applyBundle(entries []preparedEntry) ([]ledger.DirectoryIndex, error) {
	indices := make([]ledger.DirectoryIndex, len(entries))
	for i := range entries {
		indices[i] = entries[i].entryIndex
		for _, pl := range entries[i].payloads {
			if pl.Target == nil {
				continue
			}
			if err := pl.Target.ApplyMarshalTo(pl.Payload); err != nil {
				return nil, err
			}
		}
	}
	return indices, nil
}

func (p *CabinetWriter) _prepareBundle(snapshot Snapshot, entries []cabinet.WriteBundleEntry) ([]preparedEntry, error) {
	preparedEntries := make([]preparedEntry, len(entries))

	for i := range entries {
		var err error
		preparedEntries[i], err = p.prepareRecord(snapshot, entries[i])
		if err != nil {
			return nil, err
		}
	}
	if err := snapshot.Prepared(); err != nil {
		return nil, err
	}
	return preparedEntries, nil
}

func (p *CabinetWriter) prepareRecord(snapshot Snapshot, entry cabinet.WriteBundleEntry) (preparedEntry, error) {
	ds, err := snapshot.GetDirectorySection(entry.Directory)
	if err != nil {
		return preparedEntry{}, err
	}

	entryIndex := ds.GetNextDirectoryIndex()

	nPayloads := len(entry.Payloads)
	var payloadLoc []ledger.StorageLocator
	preparedPayloads := make([]preparedPayload, nPayloads + 1)

	if nPayloads > 0 {
		payloadLoc = make([]ledger.StorageLocator, nPayloads)
		for j := range entry.Payloads {
			ps, err := snapshot.GetPayloadSection(entry.Payloads[j].Section)
			if err != nil {
				return preparedEntry{}, err
			}

			pl := entry.Payloads[j].Payload
			if pl == nil {
				continue
			}

			size := pl.ProtoSize()
			receptacle, loc, err := ps.AllocatePayloadStorage(size, entry.Payloads[j].Extension)
			if err != nil {
				return preparedEntry{}, err
			}

			payloadLoc[j] = loc
			preparedPayloads[j] = preparedPayload{
				Payload: pl,
				Target:  receptacle,
				Loc:     loc,
				Size:    uint32(size),
			}
		}
	}

	entryPayload := entry.EntryFn(entryIndex, payloadLoc)
	entrySize := entryPayload.ProtoSize()
	receptacle, entryLoc, err := ds.AllocateEntryStorage(entrySize)
	if err != nil {
		return preparedEntry{}, err
	}

	if err := ds.SetDirectoryEntry(entryIndex, entry.EntryKey, entryLoc); err != nil {
		return preparedEntry{}, err
	}

	preparedPayloads[nPayloads] = preparedPayload{
		Payload: entryPayload,
		Target:  receptacle,
		Loc:     entryLoc,
		Size:    uint32(entrySize),
	}

	return preparedEntry{
		entryIndex: entryIndex,
		entryKey:   entry.EntryKey,
		payloads:   preparedPayloads,
	}, nil
}

type preparedEntry struct {
	entryIndex ledger.DirectoryIndex
	entryKey   reference.Holder
	payloads   []preparedPayload
}

type preparedPayload struct {
	Payload cabinet.MarshalerTo
	Target  MarshalToReceptacle
	Loc     ledger.StorageLocator
	Size    uint32
}

