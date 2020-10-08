// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memstor

import (
	"io"
	"math"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const MinStoragePageSize = directoryEntrySize *16

func NewMemoryStorageWriter(pn pulse.Number, maxSection ledger.SectionID, pageSize int) *MemoryStorageWriter {
	switch {
	case maxSection == 0:
		panic(throw.IllegalValue())
	case maxSection > ledger.MaxSectionID:
		panic(throw.IllegalValue())
	case pageSize < MinStoragePageSize:
		panic(throw.IllegalValue())
	}

	mc := &MemoryStorageWriter{ pn: pn}
	mc.sections = make([]cabinetSection, maxSection + 1)
	dirSize := pageSize / directoryEntrySize

	for i := ledger.SectionID(0); i <= maxSection; i++ {
		s := &mc.sections[i]
		s.sectionID = i
		s.chapters = [][]byte{ make([]byte, 0, pageSize) }

		if s.sectionID <= ledger.DefaultDustSection {
			s.hasDir = true
			s.directory = [][]bundle.DirectoryEntry{make([]bundle.DirectoryEntry, 1, dirSize) } // ordinal==0 is reserved
		}
	}

	return mc
}

var _ bundle.SnapshotWriter = &MemoryStorageWriter{}

type MemoryStorageWriter struct {
	sections []cabinetSection
	state    atomickit.Uint32
	pn       pulse.Number
}

const (
	_ = iota
	stateReadOnly
	stateBroken
)

func (p *MemoryStorageWriter) PulseNumber() pulse.Number {
	return p.pn
}

func (p *MemoryStorageWriter) TakeSnapshot() (bundle.Snapshot, error) {
	if err := p.checkWriteable(); err != nil {
		return nil, err
	}

	return &memorySnapshot{
		storage:  p,
		snapshot: make([]sectionSnapshot, len(p.sections)),
	}, nil
}

func (p *MemoryStorageWriter) IsReadOnly() bool {
	return p.state.Load() == stateReadOnly
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

func (p *MemoryStorageWriter) GetDirectoryEntry(index ledger.DirectoryIndex) bundle.DirectoryEntry {
	switch section := p.getDirtyReadSection(index.SectionID()); {
	case section == nil:
	case !section.hasDirectory():
	default:
		return section.readDirtyEntry(index)
	}
	return bundle.DirectoryEntry{}
}

func (p *MemoryStorageWriter) GetDirectoryEntries(section ledger.SectionID) [][]bundle.DirectoryEntry {
	switch section := p.getDirtyReadSection(section); {
	case section == nil:
	case !section.hasDirectory():
	default:
		return section.directory
	}
	return nil
}

func (p *MemoryStorageWriter) GetDirectoryEntryLocator(index ledger.DirectoryIndex) (ledger.StorageLocator, error) {
	return p.GetDirectoryEntry(index).Loc, nil
}

func (p *MemoryStorageWriter) GetEntryStorage(locator ledger.StorageLocator) (readbundle.Slice, error) {
	switch section := p.getDirtyReadSection(locator.SectionID()); {
	case section == nil:
	case !section.hasDirectory():
	default:
		b := section.readDirtyStorage(locator)
		u, n, err := protokit.DecodeVarintFromBytesWithError(b)
		if err != nil {
			return nil, throw.W(err, "invalid directory entry size", struct { Locator ledger.StorageLocator }{ locator})
		}
		if u == 0 {
			return nil, throw.W(err, "zero length directory entry", struct { Locator ledger.StorageLocator }{ locator})
		}
		u += uint64(n)
		if u > uint64(len(b)) {
			return nil, throw.W(err, "invalid directory entry size", struct { Locator ledger.StorageLocator }{ locator})
		}
		return readbundle.WrapBytes(b[n:u:u]), nil
	}
	return nil, nil
}

func (p *MemoryStorageWriter) GetPayloadStorage(locator ledger.StorageLocator, sz int) (readbundle.Slice, error) {
	if section := p.getDirtyReadSection(locator.SectionID()); sz >= 0 && section != nil {
		if b := section.readDirtyStorage(locator); len(b) >= sz {
			return readbundle.WrapBytes(b[:sz:sz]), nil
		}
	}
	return nil, nil
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
	for i := range p.snapshot {
		cs := p.snapshot[i].section
		if cs == nil || p.snapshot[i].dirCount == 0 {
			continue
		}
		if _, err := cs.commit(p.snapshot[i], true); err != nil {
			return throw.W(err, "pre-commit failed")
		}
	}

	hadChanges := false
	for i := range p.snapshot {
		cs := p.snapshot[i].section
		if cs == nil || p.snapshot[i].dirCount == 0 {
			continue
		}

		switch hasChanges, err := cs.commit(p.snapshot[i], false); {
		case err != nil:
			if hadChanges || hasChanges {
				p.storage.state.Store(stateBroken)
				return throw.W(err, "commit failed, storage is broken")
			}
			return throw.W(err, "commit failed")
		case hasChanges:
			hadChanges = true
		}
	}

	return nil
}

func (p *memorySnapshot) Rollback(chained bool) error {
	switch p.storage.state.Load() {
	case stateBroken:
		return throw.FailHere("rollback impossible, storage is broken")
	case stateReadOnly:
		p.storage.state.Store(stateBroken)
		return throw.FailHere("rollback failed, storage is readonly")
	default:
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
	dirCount uint32
}

func (p *sectionSnapshot) GetNextDirectoryIndex() ledger.DirectoryIndex {
	return p.section.getNextDirectoryIndex(p)
}

func (p *sectionSnapshot) AppendDirectoryEntry(index ledger.DirectoryIndex, entry bundle.DirectoryEntry) error {
	if err := p.section.setDirectoryEntry(index, entry); err != nil {
		return err
	}
	p.dirCount++
	return nil
}

func (p *sectionSnapshot) AllocateEntryStorage(size int) (bundle.PayloadReceptacle, ledger.StorageLocator, error) {
	return p.section.allocatePayloadStorage(p, size, true, 0)
}

func (p *sectionSnapshot) AllocatePayloadStorage(size int, extID ledger.ExtensionID) (bundle.PayloadReceptacle, ledger.StorageLocator, error) {
	return p.section.allocatePayloadStorage(p, size, false, extID)
}

/*********************************/

type cabinetSection struct {
	// set at construction

	sectionID ledger.SectionID
	hasDir     bool // must correlate with directory

	// mutex allows
	mutex     sync.RWMutex
	chapters  [][]byte
	directory [][]bundle.DirectoryEntry
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

func (p *cabinetSection) setDirectoryEntry(index ledger.DirectoryIndex, entry bundle.DirectoryEntry) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	n := len(p.directory)
	switch {
	case n == 0:
		panic(throw.IllegalState())
	case index.SectionID() != p.sectionID:
		panic(throw.IllegalValue())
	case entry.Loc == 0:
		panic(throw.IllegalValue())
	}

	n--
	last := p.directory[n]
	defCap := cap(last)
	ord := ledger.Ordinal(n * defCap + len(last))
	if index.Ordinal() != ord {
		panic(throw.IllegalValue())
	}

	if entry.Key.IsEmpty() {
		return throw.E("invalid key")
	}

	if defCap == len(last) {
		last = make([]bundle.DirectoryEntry, 0, defCap)
		p.directory = append(p.directory, last)
		n++
	}
	p.directory[n] = append(last, entry)
	return nil
}

// allocatePayloadStorage is allowed to reorder payloads
func (p *cabinetSection) allocatePayloadStorage(snap *sectionSnapshot, vSize int, sizePrefix bool, _ ledger.ExtensionID) (bundle.PayloadReceptacle, ledger.StorageLocator, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case vSize < 0:
		panic(throw.IllegalValue())
	case vSize > math.MaxUint32:
		panic(throw.IllegalValue())
	case vSize == 0:
		return nil, 0, nil
	}

	size := vSize
	if sizePrefix {
		size += protokit.SizeVarint32(uint32(vSize))
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

	if !sizePrefix {
		return byteReceptacle(b), loc, nil
	}

	n := protokit.EncodeVarintToBytes(b, uint64(vSize))
	return byteReceptacle(b[n:]), loc, nil
}

func (p *cabinetSection) commit(snapshot sectionSnapshot, testOnly bool) (hasChanges bool, err error) {
	_ = testOnly
	err = p.checkCommit(snapshot)
	return false, err
}

func (p *cabinetSection) checkCommit(snapshot sectionSnapshot) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	entryOrd := snapshot.dirIndex
	for i := snapshot.dirCount; i > 0; i, entryOrd = i-1, entryOrd+1 {
		switch entry := p._getDirEntry(entryOrd); {
		case entry == nil:
			return throw.Impossible()
		case entry.Fil.Link == 0:
			// there is no relative - just skip it
		case entry.Fil.Link > entryOrd:
			return throw.E("wrong head ordinal", struct { Current, Relative ledger.Ordinal }{ entryOrd, entry.Fil.Link })
		case entry.Fil.Link == entryOrd:
			// self-ref
			if entry.Fil.Flags & ledger.FilamentLocalStart == 0 {
				return throw.E("wrong self-relative", struct { Current ledger.Ordinal }{ entryOrd })
			}
		case entry.Fil.Flags & ledger.FilamentLocalStart != 0:
			return throw.E("wrong self-head", struct { Current ledger.Ordinal }{ entryOrd })
		default:
			switch relEntry := p._getDirEntry(entry.Fil.Link); {
			case relEntry == nil:
				return throw.Impossible()
			case relEntry.Fil.Flags & ledger.FilamentLocalStart == 0:
				return throw.E("wrong head type", struct { Current, Relative ledger.Ordinal }{ entryOrd, entry.Fil.Link })
			// case relEntry.Fil.Flags & ledger.FilamentClose != 0:
			// 	if entry.Fil.Flags & ledger.FilamentReopen == 0 {
			// 		return throw.E("closed head", struct { Current, Relative ledger.Ordinal }{ entryOrd, entry.Fil.Link })
			// 	}
			// case entry.Rel.Flags() &^ (ledger.FilamentClose|ledger.FilamentReopen) != 0:
			// 	return throw.E("inconsistent entry relative", struct { Current, Relative ledger.Ordinal }{ entryOrd, entry.Rel.Ordinal() })
			default:
			}
		}
	}

	return nil
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

func (p *cabinetSection) _getDirEntry(index ledger.Ordinal) *bundle.DirectoryEntry {
	n := len(p.directory)
	switch {
	case n == 0:
		panic(throw.IllegalState())
	case index == 0:
		panic(throw.IllegalValue())
	}

	defCap := cap(p.directory[0])

	pgN := int(index) / defCap
	if pgN >= n {
		return nil
	}
	pg := p.directory[pgN]

	idx := int(index) % defCap
	if idx >= len(pg) {
		return nil
	}

	return &pg[idx]
}

func (p *cabinetSection) readDirtyEntry(index ledger.DirectoryIndex) bundle.DirectoryEntry {
	if index.SectionID() != p.sectionID {
		panic(throw.IllegalValue())
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if dirEntry := p._getDirEntry(index.Ordinal()); dirEntry != nil {
		return *dirEntry
	}
	return bundle.DirectoryEntry{}
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

func (b byteReceptacle) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	return int64(n), err
}

/*********************************/

// size of bundle.DirectoryEntry
const directoryEntrySize = reference.GlobalBinarySize + 8 /* StorageLocator */ + 8 /* DirectoryIndexAndFlags */
