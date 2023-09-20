package memstor

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/ctlsection"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewMemoryStorageReaderFromWriter(w *MemoryStorageWriter) MemoryStorageReader {
	if !w.IsReadOnly() {
		panic(throw.IllegalValue())
	}

	r := MemoryStorageReader{
		sections: make([]cabinetReadSection, len(w.sections)),
	}

	for i := range w.sections {
		section := &w.sections[i]

		var directory DirectoryMap
		if section.hasDirectory() {
			directory = newInMemoryDirectoryMap(section.directory)
		}

		r.sections[i] = cabinetReadSection{
			sectionID: section.sectionID,
			chapters: section.chapters,
			directory: directory,
		}
	}
	return r
}

type cabinetReadSection struct {
	chapters  [][]byte
	directory DirectoryMap
	sectionID ledger.SectionID
}

func (v cabinetReadSection) getStorage(locator ledger.StorageLocator) ([]byte, error) {
	chapterID := locator.ChapterID()
	switch {
	case chapterID == 0:
		return nil, throw.E("zero chapter")
	case int(chapterID) > len(v.chapters):
		return nil, throw.E("chapter is out of bound")
	}

	chapter := v.chapters[chapterID - 1]
	chapterOfs := locator.ChapterOffset()
	if int(chapterOfs) >= len(chapter) {
		return nil, throw.E("chapter offset is out of bound")
	}

	return chapter[chapterOfs:], nil
}

var _ readbundle.Reader = MemoryStorageReader{}

type MemoryStorageReader struct {
	sections []cabinetReadSection
}

func (v MemoryStorageReader) getSection(id ledger.SectionID, hasDirectory bool) (cabinetReadSection, error) {
	if int(id) < len(v.sections) {
		switch s := v.sections[id]; {
		case s.chapters == nil:
			//
		case hasDirectory && s.directory == nil:
			return cabinetReadSection{}, throw.E("section is has no directory") // TODO err
		default:
			return s, nil
		}
	}
	return cabinetReadSection{}, throw.E("section is unknown")
}

func (v MemoryStorageReader) FindDirectoryEntry(id ledger.SectionID, ref reference.Holder) (ledger.Ordinal, error) {
	section, err := v.getSection(id, true)
	if err != nil {
		return 0, throw.WithDetails(err, struct{ SectionID ledger.SectionID }{id})
	}

	ord := section.directory.FindOrdinal(ref)
	return ord, nil
}

func (v MemoryStorageReader) FindDirectoryEntryLocator(id ledger.SectionID, ref reference.Holder) (ledger.StorageLocator, error) {
	section, err := v.getSection(id, true)
	if err != nil {
		return 0, throw.WithDetails(err, struct{ SectionID ledger.SectionID }{id})
	}

	loc := section.directory.FindLocator(ref)
	return loc, nil
}

func (v MemoryStorageReader) GetDirectoryEntryLocator(index ledger.DirectoryIndex) (ledger.StorageLocator, error) {
	if index == 0 {
		return 0, nil
	}
	section, err := v.getSection(index.SectionID(), true)
	if err != nil {
		return 0, throw.WithDetails(err, struct{ Index ledger.DirectoryIndex }{index})
	}

	loc := section.directory.GetByOrdinal(index.Ordinal())
	return loc, nil
}

func (v MemoryStorageReader) GetEntryStorage(locator ledger.StorageLocator) (readbundle.Slice, error) {
	if locator == 0 {
		return nil, nil
	}
	section, err := v.getSection(locator.SectionID(), true)
	if err == nil {
		var b []byte
		if b, err = section.getStorage(locator); err == nil {
			switch size, ofs, err2 := protokit.DecodeVarintFromBytesWithError(b); {
			case err2 != nil:
				err = throw.W(err2, "directory entry size is unavailable")
			case size > math.MaxInt64:
				err = throw.E("directory entry size is invalid")
			default:
				return v.readData(b, ofs, int(size))
			}
		}
	}
	return nil, throw.WithDetails(err, struct { Locator ledger.StorageLocator }{locator})
}

func (v MemoryStorageReader) GetPayloadStorage(locator ledger.StorageLocator, size int) (readbundle.Slice, error) {
	switch b, err := v.getPayloadStorage(locator, size); {
	case err != nil:
		return nil, err
	case b == nil:
		return nil, nil
	default:
		return v.readData(b, 0, size)
	}
}

func (v MemoryStorageReader) getPayloadStorage(locator ledger.StorageLocator, size int) ([]byte, error) {
	switch {
	case size < 0:
		panic(throw.IllegalValue())
	case locator == 0:
		return nil, nil
	case size == 0:
		panic(throw.IllegalValue())
	}

	switch section, err := v.getSection(locator.SectionID(), false); {
	case err != nil:
		return nil, err
	default:
		b, err := section.getStorage(locator)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}

func (v MemoryStorageReader) checkSlice(b []byte, ofs, size int) ([]byte, error) {
	switch end := ofs + size; {
	case end > len(b):
		return nil, throw.E("invalid data slice", struct{ Offset, Length int }{ofs, size})
	default:
		return b[ofs:end:end], nil
	}
}

func (v MemoryStorageReader) readData(b []byte, ofs, size int) (readbundle.Slice, error) {
	switch bb, err := v.checkSlice(b, ofs, size); {
	case err != nil:
		return nil, err
	default:
		return longbits.WrapBytes(bb), nil
	}
}

var ErrCtlRecNotFound = throw.E("control record not found")

func (v MemoryStorageReader) readCtlRecLoc(ref reference.Holder) (readbundle.Slice, error) {
	loc, err := v.FindDirectoryEntryLocator(ledger.ControlSection, ref)
	switch {
	case err != nil:
	case loc == 0:
		err = ErrCtlRecNotFound
	default:
		var slice readbundle.Slice
		switch slice, err = v.GetEntryStorage(loc); {
		case err != nil:
		case slice == nil:
			err = ErrCtlRecNotFound
		default:
			return slice, nil
		}
	}
	return nil, err
}

func (v MemoryStorageReader) readSectionSummary(sectionID ledger.SectionID) (rec rms.RCtlSectionSummary, _ error) {
	if sectionID == ledger.ControlSection {
		panic(throw.Unsupported())
	}

	slice, err := v.readCtlRecLoc(ctlsection.SectionCtlRecordRef(sectionID, rec.GetDefaultPolymorphID()))
	if err != nil {
		return rec, err
	}
	err = readbundle.UnmarshalTo(slice, &rec)
	return rec, err
}

func (v MemoryStorageReader) FinderOfNext(sectionID ledger.SectionID) readbundle.DirectoryIndexFinder {
	rec, err := v.readSectionSummary(sectionID)
	if err == nil {
		var list ctlsection.OrdinalListMapper
		list, err = v.readOrdinalListFinder(rec.RecToNextLoc, int(rec.RecToNextSize))
		if err == nil {
			return ordinalMapperFinder{"recToNext", sectionID, list}
		}
	}

	panic(err)
}

func (v MemoryStorageReader) FinderOfFirst(ledger.SectionID) readbundle.DirectoryIndexFinder {
	panic(throw.NotImplemented()) // TODO FinderOfFirst
}

func (v MemoryStorageReader) FinderOfLast(ledger.SectionID) readbundle.DirectoryIndexFinder {
	panic(throw.NotImplemented()) // TODO FinderOfLast
}

var ErrInvalidOrdinalIndex = throw.E("ordinal index has wrong size")

func (v MemoryStorageReader) readOrdinalListFinder(listLoc rms.StorageLocator, listSize int) (ctlsection.OrdinalListMapper, error) {
	if listLoc == 0 || listSize == 0 {
		return nil, nil
	}

	b0, err := v.getPayloadStorage(listLoc, listSize)
	if err != nil {
		return nil, err
	}
	if b0, err = v.checkSlice(b0, 0, listSize); err != nil {
		return nil, err
	}

	list := ctlsection.OrdinalListMapper(b0)
	switch n := list.Len(); {
	case n == 0:
		return nil, nil
	case list.Len() < 0:
		return nil, ErrInvalidOrdinalIndex
	}
	return list, nil
}

