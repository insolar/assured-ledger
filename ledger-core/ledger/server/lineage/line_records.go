// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

type stageNo uint32
type recordNo uint32
type filamentNo uint32

type updateStage struct {
	seqNo stageNo
	next  *updateStage // latter one

//	future    *buildersvc.Future
	firstRec  recordNo

	filaments    []filamentEndings
	filamentCopy bool
}

type updateRecord = resolvedRecord

type filamentEndings struct {
	earliest, latest recordNo
}


type lineRecords struct {
	records [][]updateRecord
}

func (p *lineRecords) getNextRecordNo() recordNo {
	return recordNo(p.getCount())
}

func (p *lineRecords) getCount() int {
	if len(p.records) == 0 {
		return 1
	}
	n := len(p.records) - 1
	return n * cap(p.records[0]) + len(p.records[n]) + 1
}

const defaultPageSize = 100

func (p *lineRecords) add(r updateRecord) {
	pgSize := defaultPageSize
	if n := len(p.records) - 1; n >= 0 {
		pg := &p.records[n]
		pgSize = cap(*pg)

		if pgSize > len(*pg) {
			*pg = append(*pg, r)
			return
		}
	}

	pg := make([]updateRecord, 1, pgSize)
	pg[0] = r
	p.records = append(p.records, pg)
}

func (p *lineRecords) get(recNo recordNo) *updateRecord {
	n := len(p.records) - 1
	if n < 0 || recNo == 0 {
		return nil
	}

	recNo--

	pgSize := cap(p.records[n])
	pgIndex := int(recNo) / pgSize
	pgItem := int(recNo) % pgSize

	switch {
	case pgIndex < n:
	case pgIndex > n:
		return nil
	case pgItem >= len(p.records[pgIndex]):
		return nil
	}
	return &p.records[pgIndex][pgItem]
}

func (p *lineRecords) truncate(recNo recordNo) {
	n := len(p.records) - 1

	switch {
	case recNo > 0:
	case n < 0:
		return
	default:
		p.records = p.records[:1]
		p.records[0] = p.records[0][:0]
		return
	}

	recNo--

	pgSize := cap(p.records[n])
	pgIndex := int(recNo) / pgSize
	pgItem := int(recNo) % pgSize

	p.records = p.records[:pgIndex+1]
	p.records[pgIndex] = p.records[pgIndex][:pgItem]
}
