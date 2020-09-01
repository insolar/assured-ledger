// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"io"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewMemoryStorageWriter(maxSection ledger.SectionID, pageSize int) *MemoryStorageWriter {
	switch {
	case maxSection == 0:
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
			s.hasDir = true
			s.directory = [][]directoryEntry{make([]directoryEntry, 1, dirSize) } // ordinal==0 is reserved
		}
	}

	return mc
}

var _ bundle.SnapshotWriter = &MemoryStorageWriter{}

type MemoryStorageWriter struct {
	sections []cabinetSection
	state    atomickit.Uint32
}

const (
	_ = iota
	stateReadOnly
	stateBroken
)

func (p *MemoryStorageWriter) TakeSnapshot() (bundle.Snapshot, error) {
	if err := p.checkWriteable(); err != nil {
		return nil, err
	}

	return &memorySnapshot{
		storage:  p,
		snapshot: make([]sectionSnapshot, len(p.sections)),
	}, nil
}

func (p *MemoryStorageWriter) MarkReadOnly() error {
	if p.state.CompareAndSwap(0, stateReadOnly) {
		return nil
	}
	return p.checkWriteable()
}

func (p *MemoryStorageWriter) checkWriteable() error {
	switch p.state.Load() {
	case 0:
		return nil
	case stateReadOnly:
		return throw.FailHere("readonly")
	case stateBroken:
		return throw.FailHere("broken")
	default:
		return throw.Impossible()
	}
}

func (p *MemoryStorageWriter) checkReadable() error {
	switch p.state.Load() {
	case 0, stateReadOnly:
		return nil
	case stateBroken:
		return throw.FailHere("broken")
	default:
		return throw.Impossible()
	}
}

func (p *MemoryStorageWriter) DirtyReader() bundle.DirtyReader {
	return p
}

func (p *MemoryStorageWriter) getDirtyReadSection(sectionID ledger.SectionID) *cabinetSection {
	if err := p.checkReadable(); err != nil {
		return nil
	}
	if int(sectionID) >= len(p.sections) {
		return nil
	}
	s := &p.sections[sectionID]
	if s.chapters == nil { // uninitialized/skipped section
		return nil
	}
	return s
}

func (p *MemoryStorageWriter) GetDirectoryEntry(index ledger.DirectoryIndex) (reference.Holder, ledger.StorageLocator) {
	switch section := p.getDirtyReadSection(index.SectionID()); {
	case section == nil:
	case !section.hasDirectory():
	default:
		return section.readDirtyEntry(index)
	}
	return nil, 0
}

func (p *MemoryStorageWriter) GetEntryStorage(locator ledger.StorageLocator) []byte {
	switch section := p.getDirtyReadSection(locator.SectionID()); {
	case section == nil:
	case !section.hasDirectory():
	default:
		return section.readDirtyStorage(locator)
	}
	return nil
}

func (p *MemoryStorageWriter) GetPayloadStorage(locator ledger.StorageLocator) []byte {
	if section := p.getDirtyReadSection(locator.SectionID()); section != nil {
		return section.readDirtyStorage(locator)
	}
	return nil
}


/*********************************/

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

func (p *memorySnapshot) Rollback(chained bool) error {
	broken := false
	for i := range p.snapshot {
		cs := p.snapshot[i].section
		if cs == nil {
			continue
		}
		if !cs.rollback(p.snapshot[i], chained) {
			broken = true
		}
	}

	if !broken {
		return nil
	}

	p.storage.state.Store(stateBroken)
	return throw.FailHere("rollback failed, storage is broken")
}

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
	if err := p.storage.checkWriteable(); err != nil {
		return nil, err
	}

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

/*********************************/

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

/*********************************/

type cabinetSection struct {
	// set at construction

	sectionID ledger.SectionID
	hasDir     bool // must correlate with directory

	// mutex allows
	mutex     sync.RWMutex
	chapters  [][]byte
	directory [][]directoryEntry
}

func (p *cabinetSection) hasDirectory() bool {
	return p.hasDir
}

func (p *cabinetSection) getNextDirectoryIndex(snap *sectionSnapshot) ledger.DirectoryIndex {
	p.mutex.Lock()
	defer p.mutex.Unlock()

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
	p.mutex.Lock()
	defer p.mutex.Unlock()

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

// allocatePayloadStorage is allowed to reorder payloads
func (p *cabinetSection) allocatePayloadStorage(snap *sectionSnapshot, size int, _ ledger.ExtensionID) (bundle.PayloadReceptacle, ledger.StorageLocator, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

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

func (p *cabinetSection) rollback(snapshot sectionSnapshot, chained bool) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ok := true
	if !p._rollbackDir(int(snapshot.dirIndex)) && !chained {
		ok = false
	}

	if !p._rollbackChap(int(snapshot.chapter), snapshot.lastOfs) && !chained {
		ok = false
	}
	return ok
}

func (p *cabinetSection) _rollbackChap(c int, ofs uint32) bool {
	switch {
	case c == 0:
		return true
	case c > len(p.chapters):
		return false
	}
	p.chapters = p.chapters[:c]
	c--
	chapter := p.chapters[c]

	if int(ofs) > len(chapter) {
		return false
	}
	p.chapters[c] = chapter[:ofs]
	return true
}

func (p *cabinetSection) _rollbackDir(dirIndex int) bool {
	if dirIndex == 0 {
		return true
	}
	defCap := cap(p.directory[0])

	page := dirIndex / defCap
	if page >= len(p.directory) {
		return false
	}

	p.directory = p.directory[:page+1]
	dirPage := p.directory[page]

	ofs := dirIndex % defCap
	if ofs > len(dirPage) {
		return false
	}
	p.directory[page] = dirPage[:ofs]
	return true
}

func (p *cabinetSection) readDirtyEntry(index ledger.DirectoryIndex) (reference.Holder, ledger.StorageLocator) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	n := len(p.directory)
	switch {
	case n == 0:
		panic(throw.IllegalState())
	case index.SectionID() != p.sectionID:
		panic(throw.IllegalValue())
	}

	defCap := cap(p.directory[0])

	pgN := int(index.Ordinal()) / defCap
	if pgN >= n {
		return nil, 0
	}
	pg := p.directory[pgN]

	idx := int(index.Ordinal()) % defCap
	if idx >= len(pg) {
		return nil, 0
	}
	pgE := &pg[idx]

	return pgE.key, pgE.entryLoc
}

func (p *cabinetSection) readDirtyStorage(locator ledger.StorageLocator) []byte {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	chapterID := locator.ChapterID()
	if chapterID == 0 || int(chapterID) > len(p.chapters) {
		return nil
	}

	chapter := p.chapters[chapterID - 1]
	chapterOfs := locator.ChapterOffset()
	if int(chapterOfs) >= len(chapter) {
		return nil
	}

	return chapter[chapterOfs:]
}

/*********************************/

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

/*********************************/

const directoryEntrySize = reference.GlobalBinarySize + 8

type directoryEntry struct {
	key      reference.Global
	entryLoc ledger.StorageLocator
}
