package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewSequenceReader(iter Iterator, lim Limiter, out Output) *SequenceReader {
	switch {
	case iter == nil:
		panic(throw.IllegalValue())
	case lim == nil:
		panic(throw.IllegalValue())
	case out == nil:
		panic(throw.IllegalValue())
	}

	return &SequenceReader{iter: iter, lim: lim, out: out}
}

type SequenceReader struct {
	iter   Iterator
	lim    Limiter
	out    Output
	reader readbundle.BasicReader

	recPrevRef reference.Holder
	consumedSz int
	hasNoMore  bool
}

func (p *SequenceReader) SetReader(reader readbundle.BasicReader) {
	switch {
	case reader == nil:
		panic(throw.IllegalValue())
	case p.reader != nil:
		panic(throw.IllegalState())
	}
	p.reader = reader
}

func (p *SequenceReader) SetPrevRef(ref reference.Holder) {
	if p.reader != nil {
		panic(throw.IllegalState())
	}
	p.recPrevRef = ref
}

func (p *SequenceReader) HasMore() bool {
	return !p.hasNoMore
}

func (p *SequenceReader) ReadAll() error {
	for p.HasMore() {
		if _, err := p.ReadNext(); err != nil {
			return err
		}
	}

	return nil
}

func (p *SequenceReader) ReadNext() (bool, error) {
	if p.hasNoMore {
		return false, nil
	}
	p.hasNoMore = true
	if p.reader == nil {
		panic(throw.IllegalState())
	}

	switch ok, err := p.iter.Next(p.recPrevRef); {
	case err != nil:
		return false, err
	case !ok:
		return false, nil
	}

	idx := p.iter.CurrentEntry()
	if idx == 0 {
		return false, throw.IllegalState() // TODO error
	}

	entry, err := p.readEntry(idx)
	if err != nil {
		return false, err
	}
	p.lim.Next(p.consumedSz, entry.EntryData.RecordRef.Get())
	p.consumedSz = 0

	switch {
	case !p.lim.CanRead():
		return false, err
	case !p.lim.IsSkipped(entry.EntryData.RecordType):
		if err = p.processEntryContent(&entry); err != nil {
			return false, err
		}

		for _, extraIdx := range p.iter.ExtraEntries() {
			extra, err := p.readEntry(extraIdx)
			if err != nil {
				return true, err
			}
			if err = p.processEntryContent(&extra); err != nil {
				return true, err
			}
		}
	}

	p.recPrevRef = entry.EntryData.PrevRef.Get()
	p.hasNoMore = false
	return true, nil
}

func (p *SequenceReader) readEntry(idx ledger.DirectoryIndex) (entry catalog.Entry, _ error) {
	loc, err := p.reader.GetDirectoryEntryLocator(idx)
	switch {
	case err != nil:
		return entry, err // TODO details
	case loc == 0:
		return entry, throw.IllegalState() // TODO details
	}

	var slice readbundle.Slice
	slice, err = p.reader.GetEntryStorage(loc)
	switch {
	case err != nil:
		return entry, err // TODO details
	case slice == nil:
		return entry, throw.IllegalState() // TODO details
	}

	if err = readbundle.UnmarshalToFunc(slice, entry.Unmarshal); err != nil {
		return entry, err
	}
	return entry, nil
}

func (p *SequenceReader) processEntryContent(entry *catalog.Entry) error {
	if err := p.out.BeginEntry(entry, p.lim.CanReadExcerpt()); err != nil {
		return err
	}

	bodySize, payloadSize := entry.GetBodyAndPayloadSizes()
	if bodySize > 0 && p.lim.CanReadBody() {
		slice, err := p.reader.GetPayloadStorage(entry.BodyLoc, bodySize)
		if err != nil {
			return err
		}
		if err = p.out.AddBody(slice); err != nil {
			return err
		}
	}

	if payloadSize > 0 && p.lim.CanReadPayload() {
		slice, err := p.reader.GetPayloadStorage(entry.PayloadLoc, payloadSize)
		if err != nil {
			return err
		}
		if err = p.out.AddPayload(slice); err != nil {
			return err
		}
	}

	if p.lim.CanReadExtensions() {
		for _, ext := range entry.ExtensionLoc.Ext {
			if !p.lim.CanReadExtension(ext.ExtensionID) {
				continue
			}

			slice, err := p.reader.GetPayloadStorage(ext.PayloadLoc, int(ext.PayloadSize))
			if err != nil {
				return err
			}
			if err = p.out.AddExtension(ext.ExtensionID, slice); err != nil {
				return err
			}
		}
	}

	switch consumedSize, err := p.out.EndEntry(); {
	case err != nil:
		return err
	case consumedSize > 0:
		p.consumedSz += consumedSize
	}
	return nil
}
