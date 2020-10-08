// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dataextractor

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsbox"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
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

		di := p.storage[p.fullyExtracted]
		entry := reader.GetDirectoryEntry(di)
		switch {
		case entry.IsZero():
			panic(throw.IllegalState()) // TODO errors
		case !reference.Equal(r.RecordRef.Get(), entry.Key):
			panic(throw.IllegalState())
		}

		b, err := reader.GetEntryStorage(entry.Loc)
		switch {
		case err != nil:
			panic(err)
		case b == nil:
			panic(throw.IllegalState())
		}

		ce := &catalog.Entry{}
		if err := readbundle.UnmarshalTo(b, ce); err != nil {
			panic(err)
		}

		if ce.BodyPayloadSizes == 0 {
			panic(throw.IllegalState())
		}

		bodySize := int(ce.BodyPayloadSizes&math.MaxUint32)
		payloadSize := int(ce.BodyPayloadSizes>>32)


		if bodySize > 0 && !ce.BodyLoc.IsZero() {
			b, err := reader.GetPayloadStorage(ce.BodyLoc, bodySize)
			if err != nil {
				panic(err)
			}
			r.RecordBinary.SetBytes(longbits.CopyWithLimit(b, bodySize)) // TODO remove copy when result will be used for serialization
		}

		if payloadSize > 0 && !ce.PayloadLoc.IsZero() {
			b, err := reader.GetPayloadStorage(ce.PayloadLoc, payloadSize)
			if err != nil {
				panic(err)
			}
			r.Payloads = append(r.Payloads, longbits.CopyWithLimit(b, payloadSize))
		}

		for _, ext := range ce.ExtensionLoc.Ext {
			b, err := reader.GetPayloadStorage(ext.PayloadLoc, int(ext.PayloadSize))
			if err != nil {
				panic(err)
			}
			r.Payloads = append(r.Payloads, longbits.CopyWithLimit(b, int(ext.PayloadSize)))
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
