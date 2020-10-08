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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewCollector(sel Selector, lim Limits) EntryCollector {
	return &entryCollector{ sel, limiter{ limits: lim}, nil}
}

type EntryCollector interface {
	Selector() Selector

	AddSelectedRecord(SelectedRecord) bool
	SelectionCompleted() DataCollector
}

type DataCollector interface {
	ExtractData(readbundle.BasicReader) error

	GetExtracted() []ExtractedRecord
}

type Selector struct {
	SelectorRef reference.Holder
	ReasonRef   reference.Holder
	StartRef    reference.Holder

	RootSelector  bool
	PresentToPast bool
	FullEntry     bool
}

var _ EntryCollector = &entryCollector{}
type entryCollector struct {
	sel Selector
	lim limiter
	rec []SelectedRecord
}

func (p *entryCollector) Selector() Selector {
	return p.sel
}

func (p *entryCollector) AddSelectedRecord(record SelectedRecord) bool {
	sz := record.MinSize
	if sz == 0 {
		sz = protokit.MinPolymorphFieldSize
	}
	p.lim.Next(sz, record.RecordRef)

	if p.lim.CanRead() {
		p.rec = append(p.rec, record)
		return true
	}
	return false
}

func (p *entryCollector) SelectionCompleted() DataCollector {
	if len(p.rec) == 0 {
		return nil
	}

	return &dataCollector{ entries: entryCollector{ p.sel, limiter{ limits: p.lim.limits }, p.rec }}
}

var _ DataCollector = &dataCollector{}
type dataCollector struct {
	entries entryCollector

	finished bool
	data []ExtractedRecord
}

func (p *dataCollector) GetExtracted() []ExtractedRecord {
	if p.finished {
		return p.data
	}
	return nil
}

type readDirectoryFunc = func(*SelectedRecord) (ledger.StorageLocator, error)

func (p *dataCollector) ExtractData(reader readbundle.BasicReader) error {
	if dr, ok := reader.(bundle.DirtyReader); ok {
		return p.extractData(reader, func(er *SelectedRecord) (ledger.StorageLocator, error) {
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

	return p.extractData(reader, func(er *SelectedRecord) (ledger.StorageLocator, error) {
		return reader.GetDirectoryEntryLocator(er.Index)
	})
}

func (p *dataCollector) extractData(reader readbundle.BasicReader, dirFn readDirectoryFunc) error {
	if p.finished {
		panic(throw.IllegalState())
	}

	lastSize := 0
	for i := len(p.data); i < len(p.entries.rec); i++ {
		er := &p.entries.rec[i]

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

		recRef := ce.EntryData.RecordRef.Get()
		if er.RecordRef != nil && !reference.Equal(er.RecordRef, recRef) {
			panic(throw.IllegalState())
		}
		if i == 0 {
			// TODO check selector
			if p.entries.lim.limits.ExcludeStart {
				continue
			}
		}
		if ce.BodyPayloadSizes == 0 {
			panic(throw.IllegalState())
		}

		bodySize := int(ce.BodyPayloadSizes&math.MaxUint32)
		payloadSize := int(ce.BodyPayloadSizes>>32)

		p.entries.lim.Next(lastSize, recRef)

		if !p.entries.lim.CanRead() {
			break
		}

		p.data = append(p.data, ExtractedRecord{})
		r := &p.data[i]
		lastSize = 0

		if p.entries.sel.FullEntry {
			lastSize += b.FixedByteSize()
			r.EntryData = ce.EntryData
		} else {
			lastSize += protokit.MinPolymorphFieldSize

			r.EntryData.RecordRef = ce.EntryData.RecordRef
			lastSize += ce.EntryData.RecordRef.ProtoSize()

			r.EntryData.RegistrarSignature = ce.EntryData.RegistrarSignature
			lastSize += ce.EntryData.RegistrarSignature.ProtoSize()

			r.EntryData.RegisteredBy = ce.EntryData.RegisteredBy
			lastSize += ce.EntryData.RegisteredBy.ProtoSize()

			lastSize += protokit.SizeVarint64(uint64(lastSize))
		}

		if p.entries.lim.CanReadRecord() {
			if bodySize > 0 && !ce.BodyLoc.IsZero() {
				b, err := reader.GetPayloadStorage(ce.BodyLoc, bodySize)
				if err != nil {
					panic(err)
				}
				r.RecordBinary.SetBytes(longbits.CopyWithLimit(b, bodySize)) // TODO remove copy when result will be used for serialization

				lastSize += r.RecordBinary.ProtoSize() // TODO + field size
			}
		}

		if !p.entries.lim.CanReadPayloads() {
			continue
		}

		if payloadSize > 0 && !ce.PayloadLoc.IsZero() {
			b, err := reader.GetPayloadStorage(ce.PayloadLoc, payloadSize)
			if err != nil {
				panic(err)
			}
			r.Payloads = append(r.Payloads, longbits.CopyWithLimit(b, payloadSize))

			lastSize += payloadSize // TODO + field size
		}

		for _, ext := range ce.ExtensionLoc.Ext {
			if !p.entries.lim.CanReadExtension(ext.ExtensionID) {
				continue
			}

			extSz := int(ext.PayloadSize)
			b, err := reader.GetPayloadStorage(ext.PayloadLoc, extSz)
			if err != nil {
				panic(err)
			}
			r.Payloads = append(r.Payloads, longbits.CopyWithLimit(b, extSz))

			lastSize += extSz // TODO + field size
		}
	}

	p.finished = true
	return nil
}
