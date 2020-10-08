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
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ SequenceExtractor = &WholeExtractor{}

type WholeExtractor struct {
	ReadAll bool

	records []ExtractedRecord
	expected []SelectedRecord
	// fullyExtracted int
}

func (p *WholeExtractor) AddExpectedRecord(record SelectedRecord) bool {
	if record.Index.IsZero() {
		panic(throw.IllegalValue())
	}
	p.expected = append(p.expected, record)

	return !p.ReadAll
}

func (p *WholeExtractor) NeedsReader() bool {
	return len(p.records) < len(p.expected)
	// return p.fullyExtracted < len(p.records)
}

func (p *WholeExtractor) ExtractRecordsWithReader(reader readbundle.BasicReader) error {
	if dr, ok := reader.(bundle.DirtyReader); ok {
		return p.extractRecordsWithReader(reader, func(er *SelectedRecord) (ledger.StorageLocator, error) {
			entry := dr.GetDirectoryEntry(er.Index)
			switch {
			case entry.IsZero():
				panic(throw.IllegalState()) // TODO errors
			case er.RecordRef != nil && !reference.Equal(er.RecordRef, entry.Key):
				panic(throw.IllegalState())
			}
			return entry.Loc, nil
		})
	}

	return p.extractRecordsWithReader(reader, func(er *SelectedRecord) (ledger.StorageLocator, error) {
		return reader.GetDirectoryEntryLocator(er.Index)
	})
}


func (p *WholeExtractor) extractRecordsWithReader(reader readbundle.BasicReader, dirFn readDirectoryFunc) error {
	for fullyExtracted := len(p.records); fullyExtracted < len(p.expected); fullyExtracted++ {
		er := &p.expected[fullyExtracted]

		loc, err := dirFn(er)
		if err != nil {
			return err
		}

		b, err := reader.GetEntryStorage(loc)
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

		if er.RecordRef != nil && !reference.Equal(er.RecordRef, ce.EntryData.RecordRef.Get()) {
			panic(throw.IllegalState())
		}

		p.records = append(p.records, ExtractedRecord{
			EntryData:    ce.EntryData,
		})
		r := &p.records[fullyExtracted]

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

func (p *WholeExtractor) GetExtractedRecords() []ExtractedRecord {
	if p.NeedsReader() {
		panic(throw.IllegalState())
	}
	return p.records
}

func (p *WholeExtractor) GetExtractedTail() ExtractedTail {
	return ExtractedTail{}
}
