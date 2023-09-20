package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var ErrStartRefNotFound = throw.E("StartRef not found")
var ErrPrevRefNotFound = throw.E("PrevRef not found")

type prevReader struct {
	cfg    dataextractor.Config
}

func (v prevReader) ReadData(reader readbundle.Reader) error {
	sectionID := ledger.DefaultEntrySection

	switch {
	case !v.cfg.Selector.Direction.IsToPast():
		panic(throw.IllegalState())
	case reference.IsEmpty(v.cfg.Selector.StartRef):
		panic(throw.NotImplemented())
	case !reference.IsEmpty(v.cfg.Selector.StopRef):
		panic(throw.NotImplemented())
	case !reference.IsEmpty(v.cfg.Selector.RootRef):
		panic(throw.NotImplemented())
	case !reference.IsEmpty(v.cfg.Selector.ReasonRef):
		panic(throw.NotImplemented())
	}

	startRef := v.cfg.Selector.StartRef
	ord, err := reader.FindDirectoryEntry(sectionID, startRef)

	if err != nil || ord == 0 {
		_, _ = reader.FindDirectoryEntry(sectionID, startRef)
		return throw.WithDetails(ErrStartRefNotFound, struct { PrevRef reference.Global } { reference.Copy(startRef) }, err)
	}

	sequence := &prevIterator{reader: reader, curEntry: ledger.NewDirectoryIndex(sectionID, ord)}

	extractor := dataextractor.NewSequenceReader(sequence, v.cfg.Limiter, v.cfg.Output)
	extractor.SetPrevRef(v.cfg.Selector.StartRef)
	extractor.SetReader(reader)
	return extractor.ReadAll()
}

var _ dataextractor.Iterator = &prevIterator{}
// prevIterator goes by PrevRef
type prevIterator struct {
	reader   readbundle.Reader
	curEntry ledger.DirectoryIndex
}

func (p *prevIterator) Direction() dataextractor.Direction {
	return dataextractor.ToPast
}

func (p *prevIterator) CurrentEntry() ledger.DirectoryIndex {
	return p.curEntry
}

func (p *prevIterator) ExtraEntries() []ledger.DirectoryIndex {
	return nil
}

func (p *prevIterator) Next(prevRef reference.Holder) (bool, error) {
	last := p.curEntry
	p.curEntry = 0

	if reference.IsEmpty(prevRef) {
		return false, nil
	}

	switch ord, err := p.reader.FindDirectoryEntry(last.SectionID(), prevRef); {
	case err != nil:
		return false, err
	case ord == 0:
		return false, throw.WithDetails(ErrPrevRefNotFound, struct { PrevRef reference.Global } { reference.Copy(prevRef) })
	default:
		p.curEntry = ledger.NewDirectoryIndex(last.SectionID(), ord)
		return true, nil
	}
}



