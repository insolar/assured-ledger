// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cabinet

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewBundleWriter(snap SnapshotWriter) BundleWriter {
	if snap == nil {
		panic(throw.IllegalValue())
	}
	return &bundleWriter{snap: snap}
}

var _ BundleWriter = &bundleWriter{}

type bundleWriter struct {
	mutex sync.Mutex
	snap  SnapshotWriter

	lastReady synckit.SignalChannel
}

func (p *bundleWriter) WaitWriteBundles(done synckit.SignalChannel) bool {
	p.mutex.Lock()
	prev, next := p._prepareWait(false)
	defer p.mutex.Unlock()

	if next == nil {
		// there is nothing to wait: (1) no writers (2) all done (3) already rolled back
		return true
	}

	select {
	case _, ok := <- prev:
		if ok {
			// propagate status properly as someone can be after this wait operation, hence can be affected by rollback
			next <- struct{}{}
		}
		close(next)
		return true
	case <- done:
		// make sure that commit/rollback status will be propagated properly
		go propagateReady(prev, next)
		return false
	}
}

func propagateReady(prev synckit.SignalChannel, next synckit.ClosableSignalChannel) {
	_, ok := <- prev
	if ok {
		next <- struct{}{}
	}
	close(next)
}

func (p *bundleWriter) _prepareWait(alwaysSetNext bool) (prev synckit.SignalChannel, next synckit.ClosableSignalChannel) {
	prev = p.lastReady
	p.lastReady = nil

	select {
	case <- prev:
		// ignore prev if can read it - it is either ok, or was rolled back completely
		prev = nil
	default:
		// prev hasn't finished yet
	}

	if alwaysSetNext || prev != nil {
		next = make(chan struct{}, 1)
		p.lastReady = next
	}

	return prev, next
}

func (p *bundleWriter) WriteBundle(entries []WriteBundleEntry, completedFn BundleResultFunc) error {
	if completedFn == nil {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	snapshot := p.snap.TakeSnapshot()
	defer func() {
		if snapshot != nil {
			snapshot.Rollback()
		}
	}()

	preparedEntries, err := p._prepareBundle(snapshot, entries)
	if err != nil {
		return err
	}

	prev, next := p._prepareWait(true)
	go p.applyBundleSafe(snapshot, preparedEntries, prev, next, completedFn)

	snapshot = nil
	return nil
}

func (p *bundleWriter) applyBundleSafe(snapshot Snapshot, entries []preparedEntry,
	prev synckit.SignalChannel, next synckit.ClosableSignalChannel, completedFn BundleResultFunc,
) {
	defer close(next) // to be executed as the very last one

	chained := false
	locked  := false
	rollback := true

	defer func() {
		switch {
		case !rollback:
			if locked {
				p.mutex.Unlock()
			}
			return
		case !locked:
			p.mutex.Lock()
		}
		defer p.mutex.Unlock()

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

		select {
		case _, ok := <-prev:
			if !ok {
				chained = true // rollback was made by a previous writer
				return throw.E("chained cancel")
			}
			prev = nil // previous writer is done, no need to wait for it further
		default:
			// don't wait, just check. Also handles prev==nil
		}

		var assignments []ledger.DirectoryIndex
		assignments, err = p.applyBundle(entries)

		if err = snapshot.Completed(); err != nil {
			return err
		}

		if prev != nil {
			if _, ok := <-prev; !ok {
				chained = true // rollback was made by a previous writer and we have to stop
				return throw.E("chained cancel")
			}
		}

		p.mutex.Lock()
		locked = true

		if !completedFn(assignments, nil) {
			return nil // abort
		}

		rollback = false
		if err = snapshot.Commit(); err != nil {
			return err
		}

		p.mutex.Unlock()
		locked = false

		next <- struct{}{} // send ok to next
		return nil
	}()


	if err != nil {
		if !locked {
			p.mutex.Lock()
			locked = true
		}
		completedFn(nil, err)
	}
}

func (p *bundleWriter) applyBundle(entries []preparedEntry) ([]ledger.DirectoryIndex, error) {
	indices := make([]ledger.DirectoryIndex, len(entries))
	for i := range entries {
		indices[i] = entries[i].entryIndex
		for _, pl := range entries[i].payloads {
			if pl.target == nil {
				continue
			}
			if err := pl.target.ApplyMarshalTo(pl.payload); err != nil {
				return nil, err
			}
		}
	}
	return indices, nil
}

func (p *bundleWriter) _prepareBundle(snapshot Snapshot, entries []WriteBundleEntry) ([]preparedEntry, error) {
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

func (p *bundleWriter) prepareRecord(snapshot Snapshot, entry WriteBundleEntry) (preparedEntry, error) {
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
				payload: pl,
				target:  receptacle,
				loc:     loc,
				size:    uint32(size),
			}
		}
	}

	entryPayload := entry.EntryFn(entryIndex, payloadLoc)
	entrySize := entryPayload.ProtoSize()
	receptacle, entryLoc, err := ds.AllocateEntryStorage(entrySize)
	if err != nil {
		return preparedEntry{}, err
	}

	if err := ds.AppendDirectoryEntry(entryIndex, entry.EntryKey, entryLoc); err != nil {
		return preparedEntry{}, err
	}

	preparedPayloads[nPayloads] = preparedPayload{
		payload: entryPayload,
		target:  receptacle,
		loc:     entryLoc,
		size:    uint32(entrySize),
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
	payload MarshalerTo
	target  PayloadReceptacle
	loc     ledger.StorageLocator
	size    uint32
}

