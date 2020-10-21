// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
	panic(throw.NotImplemented()) // TODO
	// sequence := &nextIterator{curIdx: 0, reader: reader, curEntry: 0}
	//
	// extractor := dataextractor.NewSequenceReader(sequence, v.cfg.Limiter, v.cfg.Output)
	// extractor.SetReader(reader)
	// return extractor.ReadAll()
}

var _ dataextractor.Iterator = &nextIterator{}
// nextIterator goes by a cursor, e.g. reversed index
type nextIterator struct {
	curIdx ledger.DirectoryIndex
	cursor readbundle.DirectoryCursor
}

func (p *nextIterator) Direction() dataextractor.Direction {
	return dataextractor.ToPresent
}

func (p *nextIterator) CurrentEntry() ledger.DirectoryIndex {
	return p.curIdx
}

func (p *nextIterator) ExtraEntries() []ledger.DirectoryIndex {
	return nil
}

func (p *nextIterator) Next(reference.Holder) (bool, error) {
	if p.curIdx == 0 {
		return false, nil
	}

	last := p.curIdx
	p.curIdx = 0

	switch idx, err := p.cursor.NextIndex(last); {
	case err != nil:
		return false, throw.WithDetails(ErrNextEntryNotFound, struct { Index ledger.DirectoryIndex }{ last }, err)
	case idx == 0:
		return false, nil
	default:
		p.curIdx = idx
		return true, nil
	}
}



