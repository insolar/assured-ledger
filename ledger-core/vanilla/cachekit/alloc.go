// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cachekit

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func newAllocationTracker(pageSize int) allocationTracker {
	if pageSize <= 0 {
		panic(throw.IllegalValue())
	}
	return allocationTracker{
		pages:   []allocPage{ make(allocPage, 0, pageSize) },
		freeIdx: math.MinInt32,
	}
}

type allocationTracker struct {
	pages   []allocPage
	freeIdx int32
	freeCnt uint32
}

type allocPage []int32

func (p *allocationTracker) Add(value int32) int {
	switch {
	case value < 0:
		panic(throw.IllegalValue())
	case p.freeIdx != math.MinInt32:
		return p.reuse(value)
	}

	pgN := len(p.pages) - 1
	pg := &p.pages[pgN]
	pgSize := cap(*pg)
	n := len(*pg)

	if n == pgSize {
		p.pages = append(p.pages, make(allocPage, 0, pgSize))
		pgN++
		n = 0
		pg = &p.pages[pgN]
	}

	*pg = append(*pg, value)
	return pgN * pgSize + n
}

func (p *allocationTracker) get(index int) *int32 {
	pgSize := cap(p.pages[0])
	pgN := index / pgSize
	if pgN >= len(p.pages) {
		return nil
	}
	n := index % pgSize
	if n >= len(p.pages[pgN]) {
		return nil
	}
	return &p.pages[pgN][n]
}


func (p *allocationTracker) AllocatedCount() int {
	n := len(p.pages)
	if n == 0 {
		return 0
	}
	return n * cap(p.pages[0])
}

func (p *allocationTracker) Count() int {
	pgN := len(p.pages)
	if pgN == 0 {
		return 0
	}

	pgSize := cap(p.pages[0])
	pgN--

	return pgN * pgSize + len(p.pages[pgN]) - int(p.freeCnt)
}

func (p *allocationTracker) Get(index int) (int32, bool) {
	switch vi := p.get(index); {
	case vi == nil:
		return -1, false
	case *vi < 0:
		return -1, false
	default:
		return *vi, true
	}
}

func (p *allocationTracker) Set(index int, value int32) bool {
	switch vi := p.get(index); {
	case value < 0:
		panic(throw.IllegalValue())
	case vi == nil:
		return false
	case *vi < 0:
		return false
	default:
		*vi = value
		return true
	}
}

func (p *allocationTracker) Inc(index int) (int32, bool) {
	switch vi := p.get(index); {
	case vi == nil:
		return -1, false
	case *vi < 0:
		return -1, false
	default:
		switch n := *vi; {
		case n < 0:
			panic(throw.IllegalValue())
		case n == math.MaxInt32:
			return math.MaxInt32, true
		default:
			n++
			*vi = n
			return n, true
		}
	}
}

func (p *allocationTracker) Dec(index int) (int32, bool, bool) {
	switch vi := p.get(index); {
	case vi == nil:
		return -1, false, false
	case *vi < 0:
		return -1, false, false
	default:
		switch n := *vi; {
		case n < 0:
			panic(throw.IllegalValue())
		case n == 0:
			return 0, false, true
		default:
			n--
			*vi = n
			return n, true, true
		}
	}
}

func (p *allocationTracker) Delete(index int) bool {
	switch vi := p.get(index); {
	case vi == nil:
		return false
	case *vi < 0:
		return false
	case p.freeIdx >= 0:
		panic(throw.Impossible())
	default:
		*vi = p.freeIdx
		p.freeIdx = -int32(index + 1)
		p.freeCnt++
		return true
	}
}

func (p *allocationTracker) reuse(value int32) int {
	free := p.freeIdx
	switch {
	case free >= 0:
		panic(throw.IllegalState())
	case free == math.MinInt32:
		// this method can't be called then
		panic(throw.Impossible())
	case p.freeCnt == 0:
		panic(throw.Impossible())
	}

	n := -int(free)-1
	switch vi := p.get(n); {
	case vi == nil:
		panic(throw.Impossible())
	case *vi >= 0:
		panic(throw.Impossible())
	default:
		p.freeIdx = *vi
		p.freeCnt--
		*vi = value
	}
	return n
}
