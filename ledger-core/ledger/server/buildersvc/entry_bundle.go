// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ bundle.Writeable = &entryWriter{}

type entryWriter struct {
	dropOrder *atomickit.Uint32
	dropBase  ledger.Ordinal

	entries   []draftEntry
	prepared  []preparedEntry
}

func (p *entryWriter) PrepareWrite(snapshot bundle.Snapshot) error {
	n := uint32(len(p.entries))
	if n == 0 {
		panic(throw.IllegalState())
	}

	p.dropBase = ledger.Ordinal(p.dropOrder.Add(n) - n) + 1 // ordinal starts with 1

	preparedEntries := make([]preparedEntry, n)

	for i := range p.entries {
		var err error
		preparedEntries[i], err = p.prepareRecord(snapshot, &p.entries[i], p.dropBase + ledger.Ordinal(i))
		if err != nil {
			return err
		}
	}
	p.prepared = preparedEntries
	p.entries = nil
	return nil
}

func (p *entryWriter) ApplyRollback() {
	if p.dropBase > 0 {
		p.dropOrder.SetLesser(uint32(p.dropBase - 1))
	}
}

func (p *entryWriter) ApplyWrite() ([]ledger.DirectoryIndex, error) {
	if len(p.prepared) == 0 {
		panic(throw.IllegalState())
	}

	entries := p.prepared
	p.prepared = nil

	indices := make([]ledger.DirectoryIndex, len(entries))
	for i := range entries {
		indices[i] = entries[i].entryIndex
		for _, pl := range entries[i].payloads {
			if pl.target == nil {
				continue
			}
			if fr, ok := pl.payload.(longbits.FixedReader); ok {
				if err := pl.target.ApplyFixedReader(fr); err != nil {
					return nil, err
				}
			}
			if err := pl.target.ApplyMarshalTo(pl.payload); err != nil {
				return nil, err
			}
		}
	}
	return indices, nil
}

func (p *entryWriter) prepareRecord(snapshot bundle.Snapshot, entry *draftEntry, dropOrdinal ledger.Ordinal) (preparedEntry, error) {
	ds, err := snapshot.GetDirectorySection(entry.directory)
	if err != nil {
		return preparedEntry{}, err
	}

	entryIndex := ds.GetNextDirectoryIndex()

	nPayloads := len(entry.payloads)
	var payloadLoc []ledger.StorageLocator
	preparedPayloads := make([]preparedPayload, nPayloads + 1)

	if nPayloads > 0 {
		payloadLoc = make([]ledger.StorageLocator, nPayloads)
		for j := range entry.payloads {
			ps, err := snapshot.GetPayloadSection(entry.payloads[j].section)
			if err != nil {
				return preparedEntry{}, err
			}

			pl := entry.payloads[j].payload
			if pl == nil {
				continue
			}

			size := pl.ProtoSize()
			receptacle, loc, err := ps.AllocatePayloadStorage(size, entry.payloads[j].extension)
			if err != nil {
				return preparedEntry{}, err
			}

			payloadLoc[j] = loc
			preparedPayloads[j] = preparedPayload{
				payload: pl,
				target:  receptacle,
				loc:     loc,
				size:    uint32(size),
			}
		}
	}

	catalogEntry := &entry.draft
	prepareCatalogEntry(catalogEntry, dropOrdinal, payloadLoc, entry.payloads, preparedPayloads[:nPayloads])

	entrySize := catalogEntry.ProtoSize()
	receptacle, entryLoc, err := ds.AllocateEntryStorage(entrySize)
	if err != nil {
		return preparedEntry{}, err
	}

	if err := ds.AppendDirectoryEntry(entryIndex,
		bundle.DirectoryEntry{
			Key: reference.Copy(entry.entryKey),
			Loc: entryLoc,
			Rel: makeRelative(entry.relative, entryIndex),
		},
	); err != nil {
		return preparedEntry{}, err
	}

	preparedPayloads[nPayloads] = preparedPayload{
		payload: catalogEntry,
		target:  receptacle,
		loc:     entryLoc,
		size:    uint32(entrySize),
	}

	return preparedEntry{
		entryIndex: entryIndex,
		entryKey:   entry.entryKey,
		payloads:   preparedPayloads,
	}, nil
}

func makeRelative(relative ledger.DirectoryIndexAndFlags, selfIndex ledger.DirectoryIndex) ledger.DirectoryIndexAndFlags {
	switch {
	case relative == 0:
		return 0
	case relative.Ordinal() == 0:
		return selfIndex.WithFlags(relative.Flags())
	default:
		return relative
	}
}

type preparedEntry struct {
	entryIndex ledger.DirectoryIndex
	entryKey   reference.Holder
	payloads   []preparedPayload
}

type preparedPayload struct {
	payload bundle.MarshalerTo
	target  bundle.PayloadReceptacle
	loc     ledger.StorageLocator
	size    uint32
}

type draftEntry struct {
	entryKey  reference.Holder
	payloads  []sectionPayload
	directory ledger.SectionID
	relative  ledger.DirectoryIndexAndFlags
	draft     catalog.Entry
}

type sectionPayload struct {
	payload   bundle.MarshalerTo
	extension ledger.ExtensionID
	section   ledger.SectionID
}

func draftCatalogEntry(rec lineage.Record) catalog.Entry {
	return catalog.Entry{
		RecordType:         rec.Excerpt.RecordType,
		PayloadDigests:     rec.Excerpt.PayloadDigests,
		PrevRef:			rec.Excerpt.PrevRef,
		RootRef:			rec.Excerpt.RootRef,
		ReasonRef:			rec.Excerpt.ReasonRef,
		RedirectRef:		rec.Excerpt.RedirectRef,
		RejoinRef:			rec.Excerpt.RejoinRef,
		RecapRef: 			rms.NewReference(rec.RecapRef),

		ProducerSignature:  rec.ProducerSignature,
		ProducedBy:         rms.NewReference(rec.ProducedBy),

		RegistrarSignature: rms.NewRaw(rec.RegistrarSignature.GetSignature()).AsBinary(),
		RegisteredBy:       rms.NewReference(rec.RegisteredBy),
	}
}

func prepareCatalogEntry(entry *catalog.Entry, dropOrdinal ledger.Ordinal, loc []ledger.StorageLocator,
	payloads []sectionPayload, preparedPayloads []preparedPayload,
) {
	entry.BodyLoc = loc[0]
	entry.Ordinal =	dropOrdinal
	entry.BodyPayloadSizes = uint64(preparedPayloads[0].size)

	n := len(loc)
	if n == 1 {
		return
	}

	entry.PayloadLoc = loc[1]
	entry.BodyPayloadSizes |= uint64(preparedPayloads[1].size)<<32

	if n == 2 {
		return
	}

	entry.ExtensionLoc.Ext = make([]rms.ExtLocator, n - 2)
	for i := 2; i < n; i++ {
		entry.ExtensionLoc.Ext[i - 2] = rms.ExtLocator{
			ExtensionID: payloads[i].extension,
			PayloadLoc:  loc[i],
			PayloadSize: preparedPayloads[i].size,
		}
	}
}

