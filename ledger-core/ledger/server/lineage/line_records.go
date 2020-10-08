// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type stageNo uint32
type recordNo uint32
type filamentNo uint32

const deadFilament = recordNo(math.MaxUint32)

type updateStage struct {
	seqNo stageNo
	next  *updateStage // latter one

	tracker StageTracker
	firstRec  recordNo

	// TODO full copy will be inefficient with multiple open filaments
	filaments     []filament
}

type updateRecord = resolvedRecord

type StageTracker interface {
	GetFutureAllocation() (isReady bool, allocations []ledger.DirectoryIndex)
}

type filamentState uint8

const (
	_  filamentState = iota // started
	// activated
	ended
	// deactivated
)

type filament struct {
	root, earliest, latest recordNo
	recap recordNo
	resolvedHead ResolvedDependency
	state filamentState
}

func (p *filament) setRoot(rn recordNo) {
	switch {
	case rn == 0:
		panic(throw.IllegalValue())
	case p.root != 0:
		panic(throw.IllegalState())
	}
	p.root = rn
}

func (p *filament) setLatest(rn recordNo) {
	switch {
	case rn == 0:
		panic(throw.IllegalValue())
	case rn == deadFilament:
		p.state |= ended
	case p.earliest == 0:
		switch {
		case p.root == 0:
			p.root = rn
		case p.root > rn:
			panic(throw.IllegalValue())
		}
		p.earliest = rn
		p.latest = rn
	case p.latest >= rn:
		panic(throw.IllegalValue())
	case p.state & ended != 0:
		p.state &^= ended
		p.latest = rn
	default:
		p.latest = rn
	}
}

//nolint //TODO revisit
func (p *filament) getLatest() recordNo {
	if p.state & ended != 0 {
		return deadFilament
	}
	return p.latest
}

type lineRecords struct {
	records [][]updateRecord
}

func (p *lineRecords) getNextRecNo() recordNo {
	return recordNo(p.getCount())
}

func (p *lineRecords) getCount() int {
	if len(p.records) == 0 {
		return 1
	}
	n := len(p.records) - 1
	return n * cap(p.records[0]) + len(p.records[n]) + 1
}

const pageSizeThreshold = 1024
const pageSizeMin = 8

func (p *lineRecords) add(r updateRecord) {
	pgSize := pageSizeMin

	if n := len(p.records); n > 0 {
		pg := &p.records[n-1]
		pgSize = cap(*pg)
		switch {
		case pgSize > len(*pg):
			*pg = append(*pg, r)
			return
		case pgSize >= pageSizeThreshold:
			// full page mode
		case n != 1:
			// only the first page can be small
			panic(throw.Impossible())
		default:
			*pg = append(*pg, r)
			return
		}
	}

	pg := make([]updateRecord, 1, pgSize)
	pg[0] = r
	p.records = append(p.records, pg)
}

func (p *lineRecords) get(recNo recordNo) *updateRecord {
	if recNo == 0 {
		return nil
	}

	n := len(p.records) - 1
	if n < 0 {
		return nil
	}

	recNo--

	pgSize := cap(p.records[0])
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

	pgSize := cap(p.records[0])
	pgIndex := int(recNo) / pgSize
	pgItem := int(recNo) % pgSize

	p.records = p.records[:pgIndex+1]
	p.records[pgIndex] = p.records[pgIndex][:pgItem]
}
