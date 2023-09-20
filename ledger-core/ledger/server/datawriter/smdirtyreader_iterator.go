package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ dataextractor.Iterator = &dirtyIterator{}
type dirtyIterator struct {
	direction dataextractor.Direction
	entries   []ledger.DirectoryIndex
	index     int
}

func (p *dirtyIterator) Direction() dataextractor.Direction {
	return p.direction
}

func (p *dirtyIterator) CurrentEntry() ledger.DirectoryIndex {
	switch {
	case p.index <= 0:
		panic(throw.IllegalState())
	case p.index > len(p.entries):
		panic(throw.IllegalState())
	default:
		return p.entries[p.index - 1]
	}
}

func (p *dirtyIterator) ExtraEntries() []ledger.DirectoryIndex {
	return nil
}

func (p *dirtyIterator) Next(reference.Holder) (bool, error) {
	switch n := len(p.entries); {
	case p.index < n:
		p.index++
		return true, nil
	case p.index == n:
		p.index++
	}
	return false, nil
}

func (p *dirtyIterator) addExpected(entryIndex ledger.DirectoryIndex) {
	if p.index != 0 {
		panic(throw.IllegalState())
	}
	p.entries = append(p.entries, entryIndex)
}

