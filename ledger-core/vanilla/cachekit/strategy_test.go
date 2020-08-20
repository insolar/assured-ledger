// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cachekit

import (
	"math"
)

func newStrategy(pgSize, maxTotal int, useFence, trimEach bool) cacheStrategy {
	return cacheStrategy{
		pgSize:   pgSize,
		maxTotal: maxTotal,
		useFence: useFence,
		trimEach: trimEach,
	}
}

var _ Strategy = cacheStrategy{}
type cacheStrategy struct {
	pgSize, maxTotal int
	useFence bool
	trimEach bool
}

func (v cacheStrategy) TrimOnEachAddition() bool {
	return v.trimEach
}

func (v cacheStrategy) CurrentAge() Age {
	return 0
}

func (v cacheStrategy) AllocationPageSize() int {
	return v.pgSize
}

func (v cacheStrategy) InitGenerationCapacity() (pageSize int, useFence bool) {
	return v.pgSize, v.useFence
}

func (v cacheStrategy) NextGenerationCapacity(prevLen int, prevCap int) (int, bool) {
	return v.pgSize, v.useFence
}

func (v cacheStrategy) CanTrimGenerations(totalCount, freqGenCount int, recent, rarest, oldest Age) int {
	n := (v.maxTotal / v.pgSize) >> 2
	if n < 10 {
		n = 10
	}
	return freqGenCount - n
}

func (v cacheStrategy) CanTrimEntries(totalCount int, recent, oldest Age) int {
	return totalCount - v.maxTotal
}

func (v cacheStrategy) CanAdvanceGeneration(curLen int, curCap int, hitRemains uint64, start, end Age) (createGeneration bool, hitLimit uint64, ageLimit Age) {
	return false, math.MaxUint64, math.MaxInt64
}

func (v cacheStrategy) InitialAdvanceLimits(curCap int, start Age) (hitLimit uint64, ageLimit Age) {
	return math.MaxUint64, math.MaxInt64
}

