// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"io"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/cabinet"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewMemoryCabinet(sectionCount, pageSize int) *MemoryCabinet {
	switch {
	case sectionCount < 0:
		panic(throw.IllegalValue())
	case pageSize < directoryEntrySize * 16:
		panic(throw.IllegalValue())
	}

	mc := &MemoryCabinet{}
	sectionCount += 2
	mc.sections = make([]cabinetSection, sectionCount)
	dirSize := pageSize / directoryEntrySize

	for i := 0; i < sectionCount; i++ {
		s := &mc.sections[i]
		s.sectionID = ledger.SectionID(i)
		s.chapters = [][]byte{ make([]byte, 0, pageSize) }
		if s.sectionID <= ledger.DefaultDustSection {
			s.directory = [][]directoryEntry{ make([]directoryEntry, 1, dirSize) } // ordinal==0 is reserved
		}
	}

	return mc
}

var _ cabinet.DropWriter = &MemoryCabinet{}

type MemoryCabinet struct {
	writeMutex sync.Mutex
	sections []cabinetSection

	lastWriterReady chan struct{}
}

func (p *MemoryCabinet) WriteBundle(entries []cabinet.WriteBundleEntry, completedFn cabinet.BundleCompletedFunc) error {
	if completedFn == nil {
		panic(throw.IllegalValue())
	}

	p.writeMutex.Lock()
	defer p.writeMutex.Unlock()

	snapshot := make([]sectionSnapshot, len(p.sections))
	defer p.rollbackBundle(snapshot)

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

	snapshotCopy := snapshot
	go func() {
		assignments, err := p.applyBundleSafe(snapshotCopy, preparedEntries, prev, next)
		completedFn(assignments, err)
	}()

	snapshot = nil
	return nil
}

func (p *MemoryCabinet) applyBundleSafe(snapshot []sectionSnapshot, entries []preparedEntry,
	prev, next synckit.ClosableSignalChannel,
) (assignments []ledger.DirectoryIndex, err error) {

	defer func() {
		p.rollbackBundle(snapshot)
		close(next)
		err = throw.RW(recover(), err, "applyBundle failed")
	}()

	assignments, err = p.applyBundle(entries)

	if prev != nil {
		if _, ok := <- prev; !ok  {
			snapshot = nil // cancel rollback - it was made by a previous writer
			return nil, throw.E("chain cancel")
		}
	}

	next <- struct{}{} // send ok to next
	snapshot = nil // no need to rollback
	return
}

func (p *MemoryCabinet) applyBundle(entries []preparedEntry) ([]ledger.DirectoryIndex, error) {
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

func (p *MemoryCabinet) _prepareBundle(snapshot []sectionSnapshot, entries []cabinet.WriteBundleEntry) ([]preparedEntry, error) {
	preparedEntries := make([]preparedEntry, len(entries))

	for i := range entries {
		var err error
		preparedEntries[i], err = p.prepareRecord(snapshot, entries[i])
		if err != nil {
			return nil, err
		}
	}
	return preparedEntries, nil
}

func (p *MemoryCabinet) prepareRecord(snapshot []sectionSnapshot, entry cabinet.WriteBundleEntry) (preparedEntry, error) {
	ds, err := p.getSection(entry.Directory, true)
	if err != nil {
		return preparedEntry{}, err
	}

	entryIndex := ds.getNextDirectoryIndex(snapshot)

	nPayloads := len(entry.Payloads)
	var payloadLoc []ledger.StorageLocator
	preparedPayloads := make([]preparedPayload, nPayloads + 1)

	if nPayloads > 0 {
		payloadLoc = make([]ledger.StorageLocator, nPayloads)
		for j := range entry.Payloads {
			ps, err := p.getSection(entry.Payloads[j].Section, false)
			if err != nil {
				return preparedEntry{}, err
			}

			pl := entry.Payloads[j].Payload
			if pl == nil {
				continue
			}

			size := pl.ProtoSize()
			receptacle, loc, err := ps.allocatePayloadStorage(snapshot, size, entry.Payloads[j].Extension)
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
	receptacle, entryLoc, err := ds.allocateEntryData(snapshot, entrySize)
	if err != nil {
		return preparedEntry{}, err
	}

	if err := ds.setDirectoryEntry(entryIndex, entry.EntryKey, entryLoc); err != nil {
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

func (p *MemoryCabinet) getSection(sectionID ledger.SectionID, directory bool) (*cabinetSection, error) {
	if int(sectionID) >= len(p.sections) {
		return nil, throw.E("unknown section", struct { ledger.SectionID }{ sectionID })
	}
	s := &p.sections[sectionID]
	if directory && !s.hasDirectory() {
		return nil, throw.E("unknown directory section", struct { ledger.SectionID }{ sectionID })
	}
	return s, nil
}

func (p *MemoryCabinet) rollbackBundle(snapshot []sectionSnapshot) {
	if len(snapshot) == 0 {
		return
	}

	p.writeMutex.Lock()
	defer p.writeMutex.Unlock()

	for i := range snapshot {
		s, _ := p.getSection(ledger.SectionID(i), false)
		s.rollback(snapshot[i])
	}
}

type sectionSnapshot struct {
	dirIndex ledger.Ordinal
	chapter  ledger.ChapterID
	lastOfs  uint32
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

type cabinetSection struct {
	sectionID    ledger.SectionID

	chapters  [][]byte
	directory [][]directoryEntry
}

func (p *cabinetSection) hasDirectory() bool {
	return len(p.directory) > 0
}

func (p *cabinetSection) getNextDirectoryIndex(snapshot []sectionSnapshot) ledger.DirectoryIndex {
	n := len(p.directory)
	if n == 0 {
		panic(throw.IllegalState())
	}

	n--
	last := p.directory[n]
	ord := ledger.Ordinal(n * cap(last) + len(last))
	ss := &snapshot[p.sectionID]
	if ss.dirIndex == 0 {
		ss.dirIndex = ord
	}
	return ledger.NewDirectoryIndex(p.sectionID, ord)
}

func (p *cabinetSection) setDirectoryEntry(index ledger.DirectoryIndex, key reference.Holder, loc ledger.StorageLocator) error {
	n := len(p.directory)
	switch {
	case n == 0:
		panic(throw.IllegalState())
	case index.SectionID() != p.sectionID:
		panic(throw.IllegalValue())
	case loc == 0:
		panic(throw.IllegalValue())
	}

	n--
	last := p.directory[n]
	defCap := cap(last)
	ord := ledger.Ordinal(n * defCap + len(last))
	if index.Ordinal() != ord {
		panic(throw.IllegalValue())
	}

	k := reference.Copy(key)
	if k.IsEmpty() {
		return throw.E("invalid key")
	}

	if defCap == len(last) {
		last = make([]directoryEntry, 0, defCap)
		p.directory = append(p.directory, last)
	}
	last = append(last, directoryEntry{
		key:      k,
		entryLoc: loc,
	})
	return nil
}

// allocatePayloadStorage can reorder payloads
func (p *cabinetSection) allocatePayloadStorage(snapshot []sectionSnapshot, size int, _ ledger.ExtensionID) (MarshalToReceptacle, ledger.StorageLocator, error) {
	switch {
	case size < 0:
		panic(throw.IllegalValue())
	case size == 0:
		return nil, 0, nil
	}

	chapterID := ledger.ChapterID(len(p.chapters))
	if chapterID == 0 {
		panic(throw.IllegalState())
	}
	chapter := p.chapters[chapterID - 1]
	lastOfs := len(chapter)

	ss := &snapshot[p.sectionID]
	if ss.chapter == 0 {
		ss.chapter = chapterID
		ss.lastOfs = uint32(lastOfs)
	}

	if n := cap(chapter) - lastOfs; n < size {
		defCap := cap(p.chapters[0])
		if size >= defCap - 32 {
			defCap = size
		}
		chapter = make([]byte, 0, defCap)
		p.chapters = append(p.chapters, chapter)
		chapterID++
		lastOfs = 0
	}

	b := chapter[lastOfs:lastOfs + size]
	loc := ledger.NewLocator(p.sectionID, chapterID, uint32(lastOfs))
	return byteReceptacle(b), loc, nil
}

func (p *cabinetSection) allocateEntryData(snapshot []sectionSnapshot, size int) (MarshalToReceptacle, ledger.StorageLocator, error) {
	return p.allocatePayloadStorage(snapshot, size, 0)
}

func (p *cabinetSection) rollback(snapshot sectionSnapshot) {
	if snapshot.dirIndex > 0 {
		defCap := cap(p.directory[0])
		page := int(snapshot.dirIndex) / defCap
		ofs := int(snapshot.dirIndex) % defCap
		p.directory = p.directory[:page]
		p.directory[page - 1] = p.directory[page - 1][:ofs]
	}

	if snapshot.chapter > 0 {
		p.chapters = p.chapters[:snapshot.chapter]
		p.chapters[snapshot.chapter - 1] = p.chapters[snapshot.chapter - 1][:snapshot.lastOfs]
	}
}

type MarshalToReceptacle interface {
	ApplyMarshalTo(cabinet.MarshalerTo) error
}

type byteReceptacle []byte

func (b byteReceptacle) ApplyMarshalTo(to cabinet.MarshalerTo) error {
	switch n, err := to.MarshalTo(b); {
	case err != nil:
		return err
	case n != len(b):
		return io.ErrShortWrite
	}
	return nil
}

const directoryEntrySize = reference.GlobalBinarySize + 8

type directoryEntry struct {
	key      reference.Global
	entryLoc ledger.StorageLocator
}
