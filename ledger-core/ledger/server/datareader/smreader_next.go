package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var ErrNextEntryNotFound = throw.E("Next entry not found, possible corruption of reverse index")

type nextReader struct {
	cfg    dataextractor.Config
}

func (v nextReader) ReadData(reader readbundle.Reader) error {
	sectionID := ledger.DefaultEntrySection

	switch {
	case v.cfg.Selector.Direction.IsToPast():
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

	nextFinder := reader.FinderOfNext(sectionID)
	if nextFinder == nil {
		panic(throw.Impossible())
	}

	startRef := v.cfg.Selector.StartRef
	ord, err := reader.FindDirectoryEntry(sectionID, startRef)

	if err != nil || ord == 0 {
		_, _ = reader.FindDirectoryEntry(sectionID, startRef)
		return throw.WithDetails(ErrStartRefNotFound, struct { PrevRef reference.Global } { reference.Copy(startRef) }, err)
	}

	sequence := &nextIterator{firstIdx: ledger.NewDirectoryIndex(sectionID, ord), finder: nextFinder}

	extractor := dataextractor.NewSequenceReader(sequence, v.cfg.Limiter, v.cfg.Output)
	extractor.SetReader(reader)
	return extractor.ReadAll()
}

var _ dataextractor.Iterator = &nextIterator{}
// nextIterator goes by a cursor, e.g. reversed index
type nextIterator struct {
	currIdx  ledger.DirectoryIndex
	firstIdx ledger.DirectoryIndex
	finder   readbundle.DirectoryIndexFinder
}

func (p *nextIterator) Direction() dataextractor.Direction {
	return dataextractor.ToPresent
}

func (p *nextIterator) CurrentEntry() ledger.DirectoryIndex {
	return p.currIdx
}

func (p *nextIterator) ExtraEntries() []ledger.DirectoryIndex {
	return nil
}

func (p *nextIterator) Next(reference.Holder) (bool, error) {
	switch {
	case p.currIdx != 0:
	case p.firstIdx == 0:
		return false, nil
	default:
		p.currIdx = p.firstIdx
		p.firstIdx = 0
		return true, nil
	}

	last := p.currIdx
	p.currIdx = 0

	switch idx, err := p.finder.LookupByIndex(last); {
	case err != nil:
		return false, throw.WithDetails(ErrNextEntryNotFound, struct { Index ledger.DirectoryIndex }{ last }, err)
	case idx == 0:
		return false, nil
	default:
		p.currIdx = idx
		return true, nil
	}
}



