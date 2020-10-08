// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsbox"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ SequenceExtractor = &WholeExtractor{}

type WholeExtractor struct {
	ReadAll bool

	records []ExtractedRecord
	storage []ledger.DirectoryIndex
	fullyExtracted int
}

func (p *WholeExtractor) AddLineRecord(record lineage.ReadRecord) bool {
	if record.StorageIndex.IsZero() {
		panic(throw.IllegalValue())
	}

	p.records = append(p.records, ExtractedRecord{
		RecordType:         record.Excerpt.RecordType,
		// Payloads:           nil,
		// RecordBinary:       nil,
		PayloadDigests:     record.Excerpt.PayloadDigests,
		RecordRef:          rms.NewReference(record.RecRef),
		PrevRef:            record.Excerpt.PrevRef,
		RootRef:            record.Excerpt.RootRef,
		ReasonRef:          record.Excerpt.ReasonRef,
		RedirectRef:        record.Excerpt.RedirectRef,
		RejoinRef:          record.Excerpt.RejoinRef,
		ProducerSignature:  record.ProducerSignature,
		ProducedBy:         rms.NewReference(record.ProducedBy),
		RegistrarSignature: rmsbox.NewRaw(record.RegistrarSignature.GetSignature()).AsBinary(),
		RegisteredBy:       rms.NewReference(record.RegisteredBy),
		RecordSize:         0,
		RecordPayloadsSize: 0,
	})
	p.storage = append(p.storage, record.StorageIndex)

	return !p.ReadAll
}

func (p *WholeExtractor) NeedsDirtyReader() bool {
	return p.fullyExtracted < len(p.records)
}

func (p *WholeExtractor) ExtractAllRecordsWithReader(reader bundle.DirtyReader) error {
	for ;p.fullyExtracted < len(p.records); p.fullyExtracted++ {
		r := &p.records[p.fullyExtracted]

		entry := reader.GetDirectoryEntry(p.storage[p.fullyExtracted])
		switch {
		case entry.IsZero():
			panic(throw.IllegalState())
		case !reference.Equal(r.RecordRef.Get(), entry.Key):
			panic(throw.IllegalState())
		}

		b := reader.GetEntryStorage(entry.Loc)
		if b == nil {
			panic(throw.IllegalState())
		}

		ce := catalog.Entry{}
		if err := ce.Unmarshal(b); err != nil {
			panic(err)
		}

		if ce.BodyPayloadSizes == 0 {
			panic(throw.IllegalState())
		}

		bodySize := uint32(ce.BodyPayloadSizes)
		payloadSize := uint32(ce.BodyPayloadSizes>>32)

		if bodySize > 0 && !ce.BodyLoc.IsZero() {
			b := reader.GetPayloadStorage(ce.BodyLoc)
			r.RecordBinary.SetBytes(append([]byte(nil), b[:bodySize]...))
		}

		if payloadSize > 0 && !ce.PayloadLoc.IsZero() {
			b := reader.GetPayloadStorage(ce.PayloadLoc)
			r.Payloads = append(r.Payloads, append([]byte(nil), b[:payloadSize]...))
		}

		for _, ext := range ce.ExtensionLoc.Ext {
			b := reader.GetPayloadStorage(ext.PayloadLoc)
			r.Payloads = append(r.Payloads, append([]byte(nil), b[:ext.PayloadSize]...))
		}
	}

	return nil
}

func (p *WholeExtractor) ExtractMoreRecords(int) bool {
	panic(throw.Unsupported())
}

func (p *WholeExtractor) GetExtractRecords() []ExtractedRecord {
	if p.NeedsDirtyReader() {
		panic(throw.IllegalState())
	}
	return p.records
}

func (p *WholeExtractor) GetExtractedTail() ExtractedTail {
	return ExtractedTail{}
}
