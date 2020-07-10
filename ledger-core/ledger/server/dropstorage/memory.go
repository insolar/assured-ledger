// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewMemoryStorageWriter(maxSection ledger.SectionID, pageSize int) *MemoryStorageWriter {
	switch {
	case maxSection <= 0:
		panic(throw.IllegalValue())
	case maxSection > ledger.MaxSectionID:
		panic(throw.IllegalValue())
	case pageSize < directoryEntrySize*16:
		panic(throw.IllegalValue())
	}

	mc := &MemoryStorageWriter{}
	mc.sections = make([]cabinetSection, maxSection + 1)
	dirSize := pageSize / directoryEntrySize

	for i := ledger.SectionID(0); i <= maxSection; i++ {
		s := &mc.sections[i]
		s.sectionID = i
		if i == ledger.ControlSection {
			continue // not (yet) supported for memory storage
		}
		s.chapters = [][]byte{ make([]byte, 0, pageSize) }

		if s.sectionID <= ledger.DefaultDustSection {
			s.directory = [][]directoryEntry{make([]directoryEntry, 1, dirSize) } // ordinal==0 is reserved
		}
	}

	return mc
}

var _ bundle.SnapshotWriter = &MemoryStorageWriter{}

type MemoryStorageWriter struct {
	sections []cabinetSection
}

func (p *MemoryStorageWriter) TakeSnapshot() bundle.Snapshot {
	return &memorySnapshot{
		storage:  p,
		snapshot: make([]sectionSnapshot, len(p.sections)),
	}
}

func (p *MemoryStorageWriter) getTotalChapterSize() int {
	n := 0
	for _, s := range p.sections {
		for j := range s.chapters {
			n += len(s.chapters[j])
		}
	}
	return n
}

type memorySnapshot struct {
	storage  *MemoryStorageWriter
	snapshot []sectionSnapshot
}

func (p *memorySnapshot) Prepared() error {
	return nil
}

func (p *memorySnapshot) Completed() error {
	return nil
}

func (p *memorySnapshot) Commit() error {
	return nil
}

func (p *memorySnapshot) Rollback() {
	for i := range p.snapshot {
		cs := p.snapshot[i].section
		if cs != nil {
			cs.rollback(p.snapshot[i])
		}
	}
}

func (p *memorySnapshot) ChainedRollback() {}

func (p *memorySnapshot) GetPayloadSection(id ledger.SectionID) (bundle.PayloadSection, error) {
	cs, err := p.getSection(id, false)
	if cs == nil {
		return nil, err
	}
	return cs, err
}

func (p *memorySnapshot) GetDirectorySection(id ledger.SectionID) (bundle.DirectorySection, error) {
	cs, err := p.getSection(id, true)
	if cs == nil {
		return nil, err
	}
	return cs, err
}

func (p *memorySnapshot) getSection(sectionID ledger.SectionID, directory bool) (*sectionSnapshot, error) {
	if int(sectionID) >= len(p.snapshot) {
		return nil, throw.E("unknown section", struct { ledger.SectionID }{ sectionID })
	}
	s := &p.snapshot[sectionID]
	if s.section == nil {
		s.section = &p.storage.sections[sectionID]
	}

	switch {
	case s.section.chapters == nil:
		return nil, throw.E("unknown section", struct { ledger.SectionID }{ sectionID })
	case directory && !s.section.hasDirectory():
		return nil, throw.E("unknown directory section", struct { ledger.SectionID }{ sectionID })
	}
	return s, nil
}

type sectionSnapshot struct {
	section  *cabinetSection
	dirIndex ledger.Ordinal
	chapter  ledger.ChapterID
	lastOfs  uint32
}

func (p *sectionSnapshot) GetNextDirectoryIndex() ledger.DirectoryIndex {
	return p.section.getNextDirectoryIndex(p)
}

func (p *sectionSnapshot) AppendDirectoryEntry(index ledger.DirectoryIndex, key reference.Holder, loc ledger.StorageLocator) error {
	return p.section.setDirectoryEntry(index, key, loc)
}

func (p *sectionSnapshot) AllocateEntryStorage(size int) (bundle.PayloadReceptacle, ledger.StorageLocator, error) {
	return p.section.allocatePayloadStorage(p, size, 0)
}

func (p *sectionSnapshot) AllocatePayloadStorage(size int, extID ledger.ExtensionID) (bundle.PayloadReceptacle, ledger.StorageLocator, error) {
	return p.section.allocatePayloadStorage(p, size, extID)
}

type cabinetSection struct {
	sectionID    ledger.SectionID

	chapters  [][]byte
	directory [][]directoryEntry
}

func (p *cabinetSection) hasDirectory() bool {
	return len(p.directory) > 0
}

func (p *cabinetSection) getNextDirectoryIndex(snap *sectionSnapshot) ledger.DirectoryIndex {
	n := len(p.directory)
	if n == 0 {
		panic(throw.IllegalState())
	}

	n--
	last := p.directory[n]
	ord := ledger.Ordinal(n * cap(last) + len(last))
	if snap.dirIndex == 0 {
		snap.dirIndex = ord
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
		n++
	}
	p.directory[n] = append(last, directoryEntry{
		key:      k,
		entryLoc: loc,
	})
	return nil
}

// allocatePayloadStorage can reorder payloads
func (p *cabinetSection) allocatePayloadStorage(snap *sectionSnapshot, size int, _ ledger.ExtensionID) (bundle.PayloadReceptacle, ledger.StorageLocator, error) {
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

	if snap.chapter == 0 {
		snap.chapter = chapterID
		snap.lastOfs = uint32(lastOfs)
	}

	var b []byte
	if n := cap(chapter) - lastOfs; n < size {
		defCap := cap(p.chapters[0])
		if size >= defCap - 32 {
			defCap = size
		}
		chapter = make([]byte, size, defCap)
		p.chapters = append(p.chapters, chapter)
		b = chapter[:size:size] // protects from race on overflow
		chapterID++
		lastOfs = 0
	} else {
		end := lastOfs + size
		p.chapters[chapterID - 1] = chapter[:end]
		b = chapter[lastOfs:end:end] // protects from race on overflow
	}

	loc := ledger.NewLocator(p.sectionID, chapterID, uint32(lastOfs))
	return byteReceptacle(b), loc, nil
}

func (p *cabinetSection) rollback(snapshot sectionSnapshot) {
	if snapshot.dirIndex > 0 {
		defCap := cap(p.directory[0])
		page := int(snapshot.dirIndex) / defCap
		ofs := int(snapshot.dirIndex) % defCap
		p.directory = p.directory[:page + 1]
		p.directory[page] = p.directory[page][:ofs]
	}

	if snapshot.chapter > 0 {
		c := snapshot.chapter
		p.chapters = p.chapters[:c]
		c--
		p.chapters[c] = p.chapters[c][:snapshot.lastOfs]
	}
}

type byteReceptacle []byte

func (b byteReceptacle) ApplyMarshalTo(to bundle.MarshalerTo) error {
	switch n, err := to.MarshalTo(b); {
	case err != nil:
		return err
	case n < len(b):
		return io.ErrShortWrite
	}
	return nil
}

func (b byteReceptacle) ApplyFixedReader(r longbits.FixedReader) error {
	if r.CopyTo(b) < len(b) {
		return io.ErrShortWrite
	}
	return nil
}

const directoryEntrySize = reference.GlobalBinarySize + 8

type directoryEntry struct {
	key      reference.Global
	entryLoc ledger.StorageLocator
}
